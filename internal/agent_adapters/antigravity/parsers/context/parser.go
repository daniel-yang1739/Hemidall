package context

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	_ "modernc.org/sqlite"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/agent_adapters/antigravity/schema"
	"heimdall/internal/core"
)

const (
	minimumSnapshotBytes   = 10_000
	snapshotRowsQuery      = "SELECT idx, data FROM gen_metadata WHERE length(data) >= ? ORDER BY idx DESC"
	lastStepIndexKey       = "last_step_index"
	compactedSummaryPrefix = "<CONTEXT_SUMMARY>"
)

// Parser reads persisted context snapshots from generation metadata. It labels
// observed wire paths rather than claiming an official protobuf schema.
type Parser struct{}

// ParseLatest returns the latest sufficiently large decodable context snapshot.
func (Parser) ParseLatest(databasePath string, source agents.SourceRef) (core.ContextSnapshot, error) {
	database, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databasePath))
	if err != nil {
		return core.ContextSnapshot{}, fmt.Errorf("open context database: %w", err)
	}
	defer database.Close()
	rows, err := database.Query(snapshotRowsQuery, minimumSnapshotBytes)
	if err != nil {
		return core.ContextSnapshot{}, fmt.Errorf("query context snapshots: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var generationIndex int
		var data []byte
		if rows.Scan(&generationIndex, &data) != nil {
			continue
		}
		snapshot, parseErr := ParseSnapshot(generationIndex, data, source)
		if parseErr == nil {
			return snapshot, nil
		}
	}
	if err := rows.Err(); err != nil {
		return core.ContextSnapshot{}, fmt.Errorf("iterate context snapshots: %w", err)
	}
	return core.ContextSnapshot{}, fmt.Errorf("no parseable persisted context snapshot found")
}

// ParseSnapshot decodes one observed persisted snapshot blob using authoritative proto schema.
func ParseSnapshot(generationIndex int, data []byte, source agents.SourceRef) (core.ContextSnapshot, error) {
	env, err := schema.DecodeRootEnvelope(data)
	if err != nil {
		return core.ContextSnapshot{}, fmt.Errorf("decode context snapshot root: %w", err)
	}
	meta, err := schema.DecodeChatModelMetadata(env.ChatModelData)
	if err != nil {
		return core.ContextSnapshot{}, fmt.Errorf("decode context snapshot field 1: %w", err)
	}
	systemPrompt := printableText([]byte(meta.SystemPrompt))
	if systemPrompt == "" {
		return core.ContextSnapshot{}, fmt.Errorf("snapshot system prompt unavailable")
	}
	snapshot := core.ContextSnapshot{
		GenerationID:    fmt.Sprintf("%d", generationIndex),
		GenerationIndex: generationIndex,
		Source:          source,
		ByteSize:        len(data),
		SystemPrompt:    systemPrompt,
		IdentityPrompt:  taggedSection(systemPrompt, "identity"),
		ConstitutionDoc: taggedSection(systemPrompt, "user_rules"),
		SkillsSection:   taggedSection(systemPrompt, "skills"),
		MCPSection:      taggedSection(systemPrompt, "mcp"),
		Evidence: []core.Evidence{{
			Level: core.EvidenceWireStructure, Source: source, Locator: fmt.Sprintf("gen_metadata.idx=%d", generationIndex), Note: "context snapshot",
		}},
	}
	if lastStepStr, found := meta.CustomMetadata[lastStepIndexKey]; found {
		if inputBoundary, convertErr := strconv.Atoi(strings.TrimSpace(lastStepStr)); convertErr == nil {
			snapshot.InputBoundaryStepIndex = inputBoundary
			snapshot.HasInputBoundary = true
		}
	}
	for i, prompt := range meta.MessagePrompts {
		record := core.PersistedContextRecord{
			Position:              i + 1,
			ByteSize:              prompt.RawByteSize,
			ObservedKind:          uint64(prompt.Role),
			RoleName:              prompt.Role.String(),
			RoleDescription:       prompt.Role.Description(),
			ObservedSequence:      uint64(prompt.SequenceIndex),
			PrimaryText:           prompt.Content,
			HasPrivateContent:     false,
			IsCompactedCheckpoint: (i == 0) && strings.HasPrefix(prompt.Content, compactedSummaryPrefix),
		}
		snapshot.PersistedContextRecords = append(snapshot.PersistedContextRecords, record)
	}
	for _, tool := range meta.Tools {
		if tool.Name != "" {
			snapshot.NativeTools = append(snapshot.NativeTools, core.ToolSignature{
				Name:        tool.Name,
				Signature:   tool.Name,
				Description: tool.Description,
				RawSchema:   tool.InputSchema,
			})
		}
	}
	return snapshot, nil
}


func printableText(data []byte) string {
	if len(data) == 0 || !utf8.Valid(data) {
		return ""
	}
	text := string(data)
	for _, value := range text {
		if !unicode.IsPrint(value) && value != '\n' && value != '\r' && value != '\t' {
			return ""
		}
	}
	return strings.TrimSpace(text)
}

func taggedSection(text, tag string) string {
	openingTag := "<" + tag + ">"
	closingTag := "</" + tag + ">"
	start := strings.Index(text, openingTag)
	if start < 0 {
		return ""
	}
	start += len(openingTag)
	end := strings.Index(text[start:], closingTag)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(text[start : start+end])
}
