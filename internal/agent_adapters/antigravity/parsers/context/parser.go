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
	"heimdall/internal/agent_adapters/antigravity/wire"
	"heimdall/internal/core"
)

const (
	minimumSnapshotBytes      = 10_000
	snapshotRowsQuery         = "SELECT idx, data FROM gen_metadata WHERE length(data) >= ? ORDER BY idx DESC"
	snapshotRootField         = 1
	snapshotSystemPromptField = 1
	snapshotRecordField       = 2
	snapshotToolField         = 8
	recordKindField           = 2
	recordTextField           = 3
	recordPrivateContentField = 11
	recordSequenceField       = 18
	toolNameField             = 1
	toolDescriptionField      = 2
	toolSchemaField           = 3
	metadataAttributeField    = 20
	metadataMapKeyField       = 1
	metadataMapValueField     = 2
	lastStepIndexKey          = "last_step_index"
	compactedSummaryPrefix    = "<CONTEXT_SUMMARY>"
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

// ParseSnapshot decodes one observed persisted snapshot blob.
func ParseSnapshot(generationIndex int, data []byte, source agents.SourceRef) (core.ContextSnapshot, error) {
	root, err := wire.Decode(data)
	if err != nil {
		return core.ContextSnapshot{}, fmt.Errorf("decode context snapshot root: %w", err)
	}
	contextFields, err := wire.Decode(wire.FirstBytes(root, snapshotRootField))
	if err != nil {
		return core.ContextSnapshot{}, fmt.Errorf("decode context snapshot field 1: %w", err)
	}
	systemPrompt := printableText(wire.FirstBytes(contextFields, snapshotSystemPromptField))
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
	if inputBoundary, found := snapshotInputBoundary(contextFields); found {
		snapshot.InputBoundaryStepIndex = inputBoundary
		snapshot.HasInputBoundary = true
	}
	for _, field := range contextFields {
		if field.Number == snapshotRecordField && field.WireType == 2 {
			snapshot.PersistedContextRecords = append(snapshot.PersistedContextRecords, parseRecord(field.Bytes, len(snapshot.PersistedContextRecords)+1))
		}
		if field.Number == snapshotToolField && field.WireType == 2 {
			if tool, found := parseTool(field.Bytes); found {
				snapshot.NativeTools = append(snapshot.NativeTools, tool)
			}
		}
	}
	return snapshot, nil
}

func snapshotInputBoundary(fields []wire.Field) (int, bool) {
	for _, field := range fields {
		if field.Number != metadataAttributeField || field.WireType != 2 {
			continue
		}
		entry, err := wire.Decode(field.Bytes)
		if err != nil || string(wire.FirstBytes(entry, metadataMapKeyField)) != lastStepIndexKey {
			continue
		}
		inputBoundary, convertErr := strconv.Atoi(strings.TrimSpace(string(wire.FirstBytes(entry, metadataMapValueField))))
		if convertErr == nil {
			return inputBoundary, true
		}
	}
	return 0, false
}

func parseRecord(data []byte, position int) core.PersistedContextRecord {
	record := core.PersistedContextRecord{Position: position, ByteSize: len(data)}
	fields, err := wire.Decode(data)
	if err != nil {
		return record
	}
	if kind, found := wire.Varint(fields, recordKindField); found {
		record.ObservedKind = kind
	}
	if sequence, found := wire.Varint(fields, recordSequenceField); found {
		record.ObservedSequence = sequence
	}
	record.PrimaryText = printableText(wire.FirstBytes(fields, recordTextField))
	record.HasPrivateContent = len(wire.FirstBytes(fields, recordPrivateContentField)) > 0
	record.IsCompactedCheckpoint = position == 1 && strings.HasPrefix(record.PrimaryText, compactedSummaryPrefix)
	return record
}

func parseTool(data []byte) (core.ToolSignature, bool) {
	fields, err := wire.Decode(data)
	if err != nil {
		return core.ToolSignature{}, false
	}
	name := printableText(wire.FirstBytes(fields, toolNameField))
	if name == "" {
		return core.ToolSignature{}, false
	}
	return core.ToolSignature{Name: name, Signature: name, Description: printableText(wire.FirstBytes(fields, toolDescriptionField)), RawSchema: printableText(wire.FirstBytes(fields, toolSchemaField))}, true
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
