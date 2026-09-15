package conversation

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/agent_adapters/antigravity/schema"
	"heimdall/internal/core"
)

const (
	generationRowsQuery            = "SELECT idx, data FROM gen_metadata ORDER BY idx ASC"
	generationIncrementalRowsQuery = "SELECT idx, data FROM gen_metadata WHERE idx > ? ORDER BY idx ASC"
	rootFieldNumber                = 1
	modelFieldNumber               = 19
	attributesFieldNumber          = 20
	mapKeyFieldNumber              = 1
	mapValueFieldNumber            = 2
	contextUsageFieldNumber        = 9
	contextUsageDetailFieldNumber  = 10
	inputUsageFieldNumber          = 4
	contextTokenFieldNumber        = 1
	uncachedInputFieldNumber       = 2
	cachedInputFieldNumber         = 5
	contextLimitFieldNumber        = 4
	generatedStepOffset            = 1
	modelEnumKey                   = "model_enum"
	lastStepIndexKey               = "last_step_index"
	lastExecutionIDKey             = "last_execution_id"
	executorMetadataTable          = "executor_metadata"
	executorMetadataTableQuery     = "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'executor_metadata')"
	executorMetadataRowsQuery      = "SELECT data FROM executor_metadata ORDER BY idx ASC"
	executionIDLength              = 36
	executionIDFirstHyphen         = 8
	executionIDSecondHyphen        = 13
	executionIDThirdHyphen         = 18
	executionIDFourthHyphen        = 23
)

var (
	exactModelPattern    = regexp.MustCompile(`(?i)^(?:gemini|claude|gpt)-[a-z0-9][a-z0-9.\-]*$`)
	embeddedModelPattern = regexp.MustCompile(`(?i)(?:gemini|claude|gpt)-[a-z0-9][a-z0-9.\-]*`)
)

// Parser reads observed generation facts from Antigravity's conversation DB.
// Its field names are evidence labels, not an assertion of an official schema.
type Parser struct{}

// ParseDatabase opens the database read-only and returns every decodable generation.
func (Parser) ParseDatabase(databasePath string, source agents.SourceRef) ([]agents.Generation, []error, error) {
	return (Parser{}).ParseDatabaseContext(context.Background(), databasePath, source)
}

// ParseDatabaseContext permits cancellation during historical hydration.
func (Parser) ParseDatabaseContext(ctx context.Context, databasePath string, source agents.SourceRef) ([]agents.Generation, []error, error) {
	database, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databasePath))
	if err != nil {
		return nil, nil, fmt.Errorf("open conversation database: %w", err)
	}
	defer database.Close()
	rows, err := database.QueryContext(ctx, generationRowsQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("query generation metadata: %w", err)
	}
	defer rows.Close()
	return parseDatabaseRows(database, rows, source)
}

// ParseDatabaseIncremental opens the database read-only and returns only generations with idx > afterIndex.
func (Parser) ParseDatabaseIncremental(databasePath string, source agents.SourceRef, afterIndex int) ([]agents.Generation, []error, error) {
	database, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databasePath))
	if err != nil {
		return nil, nil, fmt.Errorf("open conversation database: %w", err)
	}
	defer database.Close()
	rows, err := database.Query(generationIncrementalRowsQuery, afterIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("query incremental generation metadata: %w", err)
	}
	defer rows.Close()
	return parseDatabaseRows(database, rows, source)
}

func parseDatabaseRows(database *sql.DB, rows *sql.Rows, source agents.SourceRef) ([]agents.Generation, []error, error) {
	metadata := make([]generationMetadata, 0)
	diagnostics := make([]error, 0)
	for rows.Next() {
		var generationIndex int
		var data []byte
		if err := rows.Scan(&generationIndex, &data); err != nil {
			return nil, diagnostics, fmt.Errorf("scan generation metadata: %w", err)
		}
		decoded, decodeErr := decodeGenerationMetadata(generationIndex, data, source)
		if decodeErr != nil {
			diagnostics = append(diagnostics, decodeErr)
			continue
		}
		metadata = append(metadata, decoded)
	}
	if err := rows.Err(); err != nil {
		return nil, diagnostics, fmt.Errorf("iterate generation metadata: %w", err)
	}
	resolveExecutorModels(database, metadata)
	resolveUniqueEnumModels(metadata)
	generations := make([]agents.Generation, len(metadata))
	for index, item := range metadata {
		generations[index] = item.generation
	}
	return generations, diagnostics, nil
}

type generationMetadata struct {
	generation  agents.Generation
	modelEnum   string
	executionID string
}

func decodeGenerationMetadata(index int, data []byte, source agents.SourceRef) (generationMetadata, error) {
	env, err := schema.DecodeRootEnvelope(data)
	if err != nil {
		return generationMetadata{}, fmt.Errorf("decode generation %d: %w", index, err)
	}
	if len(env.ChatModelData) == 0 {
		return generationMetadata{}, fmt.Errorf("generation %d has no observed root message", index)
	}
	meta, err := schema.DecodeChatModelMetadata(env.ChatModelData)
	if err != nil {
		return generationMetadata{}, fmt.Errorf("decode ChatModelMetadata %d: %w", index, err)
	}
	modelID := directModelID(meta.ResponseModel)
	inputBoundary, hasInputBoundary := extractInputBoundary(meta.CustomMetadata)
	usage := decodeUsage(meta)
	stepIndex := inputBoundary
	if hasInputBoundary {
		stepIndex += generatedStepOffset
	}
	locator := fmt.Sprintf("gen_metadata.idx=%d", index)
	evidence := []agents.Evidence{{Level: agents.EvidenceWireStructure, Source: source, Locator: locator, Note: "gen_metadata"}}
	if modelID != "" {
		evidence = append(evidence, agents.Evidence{Level: agents.EvidenceWireStructure, Source: source, Locator: locator, Note: "direct model field"})
	}
	return generationMetadata{
		generation:  agents.Generation{ID: strconv.Itoa(index), StepIndex: stepIndex, HasStepIndex: hasInputBoundary, Provider: core.ProviderVertexAI, ModelID: modelID, Usage: usage, Evidence: evidence},
		modelEnum:   meta.CustomMetadata[modelEnumKey],
		executionID: meta.CustomMetadata[lastExecutionIDKey],
	}, nil
}

func resolveExecutorModels(database *sql.DB, metadata []generationMetadata) {
	modelByExecutionID := executorModels(database)
	for index := range metadata {
		item := &metadata[index]
		if item.generation.ModelID != "" || item.executionID == "" {
			continue
		}
		modelID := modelByExecutionID[item.executionID]
		if modelID == "" {
			continue
		}
		item.generation.ModelID = modelID
		item.generation.Evidence = append(item.generation.Evidence, agents.Evidence{Level: agents.EvidenceArtifactEquality, Source: item.generation.Evidence[0].Source, Locator: item.generation.Evidence[0].Locator, Note: "executor metadata execution ID match"})
	}
}

func executorModels(database *sql.DB) map[string]string {
	models := make(map[string]string)
	ambiguousIDs := make(map[string]struct{})
	var tableExists int
	if database.QueryRow(executorMetadataTableQuery).Scan(&tableExists) != nil || tableExists == 0 {
		return models
	}
	rows, err := database.Query(executorMetadataRowsQuery)
	if err != nil {
		return models
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if rows.Scan(&data) != nil {
			continue
		}
		modelID := embeddedModelID(data)
		if modelID == "" {
			continue
		}
		for _, executionID := range observedExecutionIDs(data) {
			if _, ambiguous := ambiguousIDs[executionID]; ambiguous {
				continue
			}
			knownModelID, exists := models[executionID]
			if !exists || knownModelID == modelID {
				models[executionID] = modelID
				continue
			}
			delete(models, executionID)
			ambiguousIDs[executionID] = struct{}{}
		}
	}
	return models
}

func embeddedModelID(data []byte) string {
	matches := embeddedModelPattern.FindAllString(string(data), -1)
	if len(matches) == 0 {
		return ""
	}
	modelID := strings.ToLower(matches[0])
	for _, match := range matches[1:] {
		if strings.ToLower(match) != modelID {
			return ""
		}
	}
	return modelID
}

func observedExecutionIDs(data []byte) []string {
	executionIDs := make([]string, 0)
	seen := make(map[string]struct{})
	for start := 0; start+executionIDLength <= len(data); start++ {
		candidate := string(data[start : start+executionIDLength])
		if !isCanonicalExecutionID(candidate) {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		executionIDs = append(executionIDs, candidate)
	}
	return executionIDs
}

func isCanonicalExecutionID(value string) bool {
	if len(value) != executionIDLength {
		return false
	}
	for index, character := range value {
		if index == executionIDFirstHyphen || index == executionIDSecondHyphen || index == executionIDThirdHyphen || index == executionIDFourthHyphen {
			if character != '-' {
				return false
			}
			continue
		}
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') && (character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}

func resolveUniqueEnumModels(metadata []generationMetadata) {
	modelsByEnum := make(map[string]string)
	ambiguousEnums := make(map[string]struct{})
	for _, item := range metadata {
		if item.modelEnum == "" || item.generation.ModelID == "" {
			continue
		}
		known, exists := modelsByEnum[item.modelEnum]
		if !exists || known == item.generation.ModelID {
			modelsByEnum[item.modelEnum] = item.generation.ModelID
			continue
		}
		delete(modelsByEnum, item.modelEnum)
		ambiguousEnums[item.modelEnum] = struct{}{}
	}
	for index := range metadata {
		item := &metadata[index]
		if item.generation.ModelID != "" || item.modelEnum == "" {
			continue
		}
		if _, ambiguous := ambiguousEnums[item.modelEnum]; ambiguous {
			continue
		}
		modelID := modelsByEnum[item.modelEnum]
		if modelID == "" {
			continue
		}
		item.generation.ModelID = modelID
		item.generation.Evidence = append(item.generation.Evidence, agents.Evidence{Level: agents.EvidenceStrongInference, Source: item.generation.Evidence[0].Source, Locator: item.generation.Evidence[0].Locator, Note: "unique model enum mapping"})
	}
}

func directModelID(responseModel string) string {
	modelID := strings.ToLower(responseModel)
	if !exactModelPattern.MatchString(modelID) {
		return ""
	}
	return modelID
}

func extractInputBoundary(customMetadata map[string]string) (int, bool) {
	val, exists := customMetadata[lastStepIndexKey]
	if !exists {
		return 0, false
	}
	intVal, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return 0, false
	}
	return intVal, true
}

func decodeUsage(meta schema.ChatModelMetadata) agents.UsageObservation {
	usage := agents.UsageObservation{}
	if meta.ChatStart.HasObservedContextTokens || meta.ChatStart.ObservedContextTokens > 0 {
		usage.ObservedContextTokens = int(meta.ChatStart.ObservedContextTokens)
		usage.HasObservedContextTokens = true
	}
	if meta.ChatStart.HasContextLimitTokens || meta.ChatStart.ContextLimitTokens > 0 {
		usage.ContextLimit = int(meta.ChatStart.ContextLimitTokens)
		usage.HasContextLimit = true
	}

	u := meta.Usage
	if u.HasUncachedPromptTokens || u.UncachedPromptTokens > 0 {
		usage.UncachedInputTokens = int(u.UncachedPromptTokens)
		usage.HasUncachedInputTokens = true
	}
	if u.HasCachedContentTokens || u.CachedContentTokens > 0 {
		usage.CachedInputTokens = int(u.CachedContentTokens)
		usage.HasCachedInputTokens = true
	}
	if u.HasThinkingOutputTokens || u.ThinkingOutputTokens > 0 {
		usage.ThinkingOutputTokens = int(u.ThinkingOutputTokens)
		usage.HasThinkingOutputTokens = true
	}
	if u.HasOutputContentTokens || u.OutputContentTokens > 0 {
		usage.OutputContentTokens = int(u.OutputContentTokens)
		usage.HasOutputContentTokens = true
	}
	if u.UpstreamRequestID != "" {
		usage.UpstreamRequestID = u.UpstreamRequestID
	}

	// Keep the context-state observation distinct from billing input counters.

	if meta.TimeToFirstToken > 0 {
		usage.TimeToFirstTokenMs = meta.TimeToFirstToken.Milliseconds()
	}
	if meta.StreamingDuration > 0 {
		usage.StreamingDurationMs = meta.StreamingDuration.Milliseconds()
	}

	// Calculate TotalTokens = ObservedContextTokens (Input) + OutputTokens
	outputTokens := usage.ThinkingOutputTokens + usage.OutputContentTokens
	usage.TotalTokens = usage.UncachedInputTokens + usage.CachedInputTokens + outputTokens
	return usage
}
