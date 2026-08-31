package antigravity

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	_ "modernc.org/sqlite"

	"heimdall/internal/core"
)

const minimumContextSnapshotBytes = 10_000

type protoField struct {
	number   int
	wireType int
	bytes    []byte
	integer  uint64
}

// ContextSnapshot is Antigravity's persisted generation-context snapshot.
// Field names are semantic labels inferred from repeated observations; the
// numeric wire paths remain the authoritative identifiers until a schema is available.
type ContextSnapshot struct {
	GenIndex                   int
	BlobBytes                  int
	InputBoundaryStepIndex     int
	HasInputBoundary           bool
	SourcePath                 string
	SystemPrompt               string
	Identity                   string
	UserRules                  string
	SkillsSection              string
	MCPSection                 string
	PersistedContextEntryCount int
	PersistedRecords           []core.PersistedContextRecord
	Tools                      []core.ToolSignature
}

// LoadContextSnapshotAtPath reads newest-to-oldest generation records and returns
// the first sufficiently large record that matches the observed snapshot shape.
// Generation index is the only ordering evidence persisted in this table.
func LoadContextSnapshotAtPath(dbPath string) (*ContextSnapshot, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open context database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT idx, data FROM gen_metadata WHERE length(data) >= ? ORDER BY idx DESC", minimumContextSnapshotBytes)
	if err != nil {
		return nil, fmt.Errorf("query context snapshots: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var genIndex int
		var data []byte
		if scanErr := rows.Scan(&genIndex, &data); scanErr != nil {
			continue
		}
		snapshot, parseErr := ParseContextSnapshot(genIndex, data)
		if parseErr != nil {
			continue
		}
		snapshot.SourcePath = dbPath
		return snapshot, nil
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate context snapshots: %w", err)
	}
	return nil, fmt.Errorf("no parseable persisted context snapshot found")
}

// ParseContextSnapshot decodes the stable wire paths observed in Antigravity session databases.
func ParseContextSnapshot(genIndex int, data []byte) (*ContextSnapshot, error) {
	root, err := decodeProtoFields(data)
	if err != nil {
		return nil, fmt.Errorf("decode snapshot root: %w", err)
	}
	contextBytes := firstBytes(root, 1)
	contextFields, err := decodeProtoFields(contextBytes)
	if err != nil {
		return nil, fmt.Errorf("decode snapshot context field 1: %w", err)
	}
	systemPrompt := printableString(firstBytes(contextFields, 1))
	if systemPrompt == "" {
		return nil, fmt.Errorf("snapshot field 1.1 does not contain a system prompt")
	}

	snapshot := &ContextSnapshot{
		GenIndex:                   genIndex,
		BlobBytes:                  len(data),
		SystemPrompt:               systemPrompt,
		Identity:                   extractTaggedSection(systemPrompt, "identity"),
		UserRules:                  extractTaggedSection(systemPrompt, "user_rules"),
		SkillsSection:              extractTaggedSection(systemPrompt, "skills"),
		MCPSection:                 extractTaggedSection(systemPrompt, "mcp"),
		PersistedContextEntryCount: countFields(contextFields, 2),
	}
	metadata, metadataErr := ParsePersistedGenerationMetadata(genIndex, data)
	if metadataErr == nil && metadata.HasInputBoundary {
		snapshot.InputBoundaryStepIndex = metadata.InputBoundaryStepIndex
		snapshot.HasInputBoundary = true
	}
	for _, field := range contextFields {
		if field.number == 2 && field.wireType == 2 {
			record := decodePersistedContextRecord(field.bytes, len(snapshot.PersistedRecords)+1)
			snapshot.PersistedRecords = append(snapshot.PersistedRecords, record)
		}
		if field.number != 8 || field.wireType != 2 {
			continue
		}
		toolFields, decodeErr := decodeProtoFields(field.bytes)
		if decodeErr != nil {
			continue
		}
		name := printableString(firstBytes(toolFields, 1))
		if name == "" {
			continue
		}
		description := printableString(firstBytes(toolFields, 2))
		rawSchema := printableString(firstBytes(toolFields, 3))
		snapshot.Tools = append(snapshot.Tools, core.ToolSignature{
			Name: name, Signature: name, Description: description, RawSchema: rawSchema,
		})
	}
	return snapshot, nil
}

func decodePersistedContextRecord(data []byte, position int) core.PersistedContextRecord {
	record := core.PersistedContextRecord{Position: position, ByteSize: len(data)}
	fields, err := decodeProtoFields(data)
	if err != nil {
		return record
	}
	for _, field := range fields {
		switch {
		case field.number == 2 && field.wireType == 0:
			record.ObservedKind = field.integer
		case field.number == 3 && field.wireType == 2:
			record.PrimaryText = printableString(field.bytes)
		case field.number == 11 && field.wireType == 2 && len(field.bytes) > 0:
			record.HasPrivateContent = true
		case field.number == 18 && field.wireType == 0:
			record.ObservedSequence = field.integer
		}
	}
	record.IsCompactedCheckpoint = position == 1 && strings.HasPrefix(record.PrimaryText, "<CONTEXT_SUMMARY>")
	return record
}

func decodeProtoFields(data []byte) ([]protoField, error) {
	var fields []protoField
	for offset := 0; offset < len(data); {
		tag, consumed := readVarint(data[offset:])
		if consumed == 0 || tag>>3 == 0 {
			return nil, fmt.Errorf("invalid protobuf tag at byte %d", offset)
		}
		offset += consumed
		field := protoField{number: int(tag >> 3), wireType: int(tag & 7)}
		switch field.wireType {
		case 0:
			field.integer, consumed = readVarint(data[offset:])
			if consumed == 0 {
				return nil, fmt.Errorf("invalid varint at byte %d", offset)
			}
			offset += consumed
		case 1:
			if offset+8 > len(data) {
				return nil, fmt.Errorf("truncated fixed64 at byte %d", offset)
			}
			field.bytes = data[offset : offset+8]
			offset += 8
		case 2:
			length, lengthBytes := readVarint(data[offset:])
			if lengthBytes == 0 || length > uint64(len(data)-offset-lengthBytes) {
				return nil, fmt.Errorf("invalid length-delimited field at byte %d", offset)
			}
			offset += lengthBytes
			field.bytes = data[offset : offset+int(length)]
			offset += int(length)
		case 5:
			if offset+4 > len(data) {
				return nil, fmt.Errorf("truncated fixed32 at byte %d", offset)
			}
			field.bytes = data[offset : offset+4]
			offset += 4
		default:
			return nil, fmt.Errorf("unsupported wire type %d", field.wireType)
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func firstBytes(fields []protoField, number int) []byte {
	for _, field := range fields {
		if field.number == number && field.wireType == 2 {
			return field.bytes
		}
	}
	return nil
}

func countFields(fields []protoField, number int) int {
	count := 0
	for _, field := range fields {
		if field.number == number {
			count++
		}
	}
	return count
}

func printableString(data []byte) string {
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

func extractTaggedSection(text, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	start := strings.Index(text, open)
	if start < 0 {
		return ""
	}
	start += len(open)
	end := strings.Index(text[start:], close)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(text[start : start+end])
}
