package conversation

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/agent_adapters/antigravity/wire"
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
	database, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databasePath))
	if err != nil {
		return nil, nil, fmt.Errorf("open conversation database: %w", err)
	}
	defer database.Close()
	rows, err := database.Query(generationRowsQuery)
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
	root, err := wire.Decode(data)
	if err != nil {
		return generationMetadata{}, fmt.Errorf("decode generation %d: %w", index, err)
	}
	metadataFields, found := nested(root, rootFieldNumber)
	if !found {
		return generationMetadata{}, fmt.Errorf("generation %d has no observed root message", index)
	}
	modelID := directModelID(metadataFields)
	inputBoundary, hasInputBoundary := inputBoundary(metadataFields)
	usage := decodeUsage(metadataFields)
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
		generation:  agents.Generation{ID: strconv.Itoa(index), StepIndex: stepIndex, ModelID: modelID, Usage: usage, Evidence: evidence},
		modelEnum:   attributeValue(metadataFields, modelEnumKey),
		executionID: attributeValue(metadataFields, lastExecutionIDKey),
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

func directModelID(fields []wire.Field) string {
	modelID := strings.ToLower(string(wire.FirstBytes(fields, modelFieldNumber)))
	if !exactModelPattern.MatchString(modelID) {
		return ""
	}
	return modelID
}

func inputBoundary(fields []wire.Field) (int, bool) {
	value, err := strconv.Atoi(attributeValue(fields, lastStepIndexKey))
	if err != nil {
		return 0, false
	}
	return value, true
}

func attributeValue(fields []wire.Field, key string) string {
	for _, field := range fields {
		if field.Number != attributesFieldNumber {
			continue
		}
		entry, err := wire.Decode(field.Bytes)
		if err != nil || string(wire.FirstBytes(entry, mapKeyFieldNumber)) != key {
			continue
		}
		return string(wire.FirstBytes(entry, mapValueFieldNumber))
	}
	return ""
}

func decodeUsage(fields []wire.Field) agents.UsageObservation {
	usage := agents.UsageObservation{}
	contextUsage, contextFound := nested(fields, contextUsageFieldNumber)
	if contextFound {
		contextDetail, detailFound := nested(contextUsage, contextUsageDetailFieldNumber)
		if detailFound {
			observedTokens, observed := wire.Varint(contextDetail, contextTokenFieldNumber)
			contextLimit, limitObserved := wire.Varint(contextDetail, contextLimitFieldNumber)
			usage.ObservedContextTokens = int(observedTokens)
			usage.HasObservedContextTokens = observed
			usage.ContextLimit = int(contextLimit)
			usage.HasContextLimit = limitObserved
		}
	}
	inputUsage, inputUsageFound := nested(fields, inputUsageFieldNumber)
	if !inputUsageFound {
		fallbackEnvelope, fallbackFound := nested(fields, 17)
		if fallbackFound {
			inputUsage, inputUsageFound = nested(fallbackEnvelope, 2)
		}
	}
	if inputUsageFound {
		uncachedInput, inputObserved := wire.Varint(inputUsage, uncachedInputFieldNumber)
		cachedInput, cacheObserved := wire.Varint(inputUsage, cachedInputFieldNumber)
		thinkingOutput, thinkingObserved := wire.Varint(inputUsage, 3)
		outputContent, outputObserved := wire.Varint(inputUsage, 9)
		reqIDBytes := wire.FirstBytes(inputUsage, 11)

		usage.UncachedInputTokens = int(uncachedInput)
		usage.HasUncachedInputTokens = inputObserved
		usage.CachedInputTokens = int(cachedInput)
		usage.HasCachedInputTokens = cacheObserved
		if thinkingObserved {
			usage.ThinkingOutputTokens = int(thinkingOutput)
		}
		if outputObserved {
			usage.OutputContentTokens = int(outputContent)
		}
		if len(reqIDBytes) > 0 {
			usage.UpstreamRequestID = string(reqIDBytes)
		}
	}

	// Authoritative input calculation:
	// ObservedContextTokens is the true empirical prompt context: CachedInputTokens + UncachedInputTokens
	if usage.UncachedInputTokens > 0 || usage.CachedInputTokens > 0 {
		usage.ObservedContextTokens = usage.UncachedInputTokens + usage.CachedInputTokens
		usage.HasObservedContextTokens = true
	}

	// Decode TTFT (F11)
	if ttftFields, found := nested(fields, 11); found {
		sec, _ := wire.Varint(ttftFields, 1)
		nanos, _ := wire.Varint(ttftFields, 2)
		usage.TimeToFirstTokenMs = int64(sec*1000 + nanos/1_000_000)
	}

	// Decode Streaming Duration (F12)
	if durFields, found := nested(fields, 12); found {
		sec, _ := wire.Varint(durFields, 1)
		nanos, _ := wire.Varint(durFields, 2)
		usage.StreamingDurationMs = int64(sec*1000 + nanos/1_000_000)
	}

	// Calculate TotalTokens = ObservedContextTokens (Input) + OutputTokens
	outputTokens := usage.ThinkingOutputTokens + usage.OutputContentTokens
	usage.TotalTokens = usage.ObservedContextTokens + outputTokens
	return usage
}

func nested(fields []wire.Field, fieldNumber int) ([]wire.Field, bool) {
	data := wire.FirstBytes(fields, fieldNumber)
	if len(data) == 0 {
		return nil, false
	}
	nestedFields, err := wire.Decode(data)
	if err != nil {
		return nil, false
	}
	return nestedFields, true
}
