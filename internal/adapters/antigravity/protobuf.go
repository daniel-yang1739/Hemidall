package antigravity

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	inputBoundaryPattern     = regexp.MustCompile(`last_step_index\x12[\x01-\x08](\d+)`)
	exactModelNamePattern    = regexp.MustCompile(`(?i)^(?:gemini|claude|gpt)-[a-z0-9][a-z0-9.\-]*$`)
	embeddedModelNamePattern = regexp.MustCompile(`(?i)(?:gemini|claude|gpt)-[a-z0-9][a-z0-9.\-]*`)
)

const (
	persistedMetadataRootFieldNumber       = 1
	persistedMetadataModelFieldNumber      = 19
	persistedMetadataAttributesFieldNumber = 20
	persistedMetadataMapKeyFieldNumber     = 1
	persistedMetadataMapValueFieldNumber   = 2
	persistedModelEnumKey                  = "model_enum"
	persistedLastStepIndexKey              = "last_step_index"
	persistedLastExecutionIDKey            = "last_execution_id"
	persistedUsageInputFieldNumber         = 1
	persistedUsageCachedContentFieldNumber = 5
)

// PersistedGenerationMetadata contains schema-inferred observations from an
// Antigravity generation metadata blob. The local wire format is not an
// official public protobuf schema.
type PersistedGenerationMetadata struct {
	GenIndex                 int
	InputBoundaryStepIndex   int
	HasInputBoundary         bool
	ObservedContextTokens    int
	HasObservedContextTokens bool
	MeteredInputTokens       int
	HasMeteredInputTokens    bool
	CachedContentTokens      int
	HasCachedContentTokens   bool
	ContextLimit             int
	HasContextLimit          bool
	ModelName                string
	ModelEnum                string
	LastExecutionID          string
}

// ParsePersistedGenerationMetadata decodes the raw protobuf blob from gen_metadata.
// Compiling the observed text patterns once is material during long-session
// hydration because this function may inspect thousands of rows.
func ParsePersistedGenerationMetadata(genIndex int, data []byte) (*PersistedGenerationMetadata, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty protobuf data")
	}

	meta := &PersistedGenerationMetadata{GenIndex: genIndex}

	decodePersistedMetadataEnvelope(data, meta)

	// The raw form remains a compatibility fallback for records written before
	// the observed metadata envelope. Current records use the structured map.
	if !meta.HasInputBoundary {
		decodeLegacyInputBoundary(data, meta)
	}

	decodePersistedUsageFields(data, meta)

	return meta, nil
}

func decodePersistedMetadataEnvelope(data []byte, meta *PersistedGenerationMetadata) {
	root, err := decodeProtoFields(data)
	if err != nil {
		return
	}
	for _, rootField := range root {
		if rootField.number != persistedMetadataRootFieldNumber || rootField.wireType != 2 {
			continue
		}
		rootMetadata, decodeErr := decodeProtoFields(rootField.bytes)
		if decodeErr != nil {
			continue
		}
		if decodePersistedMetadataFields(rootMetadata, meta) {
			return
		}
	}
}

func decodePersistedMetadataFields(fields []protoField, meta *PersistedGenerationMetadata) bool {
	modelEnum := persistedMetadataMapValue(fields, persistedModelEnumKey)
	lastExecutionID := persistedMetadataMapValue(fields, persistedLastExecutionIDKey)
	if modelEnum == "" && lastExecutionID == "" {
		return false
	}
	meta.ModelEnum = modelEnum
	meta.ModelName = persistedDirectModelName(fields)
	meta.LastExecutionID = lastExecutionID
	decodeStructuredInputBoundary(persistedMetadataMapValue(fields, persistedLastStepIndexKey), meta)
	return true
}

func persistedMetadataMapValue(fields []protoField, targetKey string) string {
	for _, field := range fields {
		if field.number != persistedMetadataAttributesFieldNumber || field.wireType != 2 {
			continue
		}
		entry, decodeErr := decodeProtoFields(field.bytes)
		if decodeErr != nil {
			continue
		}
		key := printableString(firstBytes(entry, persistedMetadataMapKeyFieldNumber))
		if key != targetKey {
			continue
		}
		return printableString(firstBytes(entry, persistedMetadataMapValueFieldNumber))
	}
	return ""
}

func persistedDirectModelName(fields []protoField) string {
	modelName := printableString(firstBytes(fields, persistedMetadataModelFieldNumber))
	if !exactModelNamePattern.MatchString(modelName) {
		return ""
	}
	return strings.ToLower(modelName)
}

func decodeStructuredInputBoundary(rawValue string, meta *PersistedGenerationMetadata) {
	stepIndex, err := strconv.Atoi(rawValue)
	if err != nil {
		return
	}
	meta.InputBoundaryStepIndex = stepIndex
	meta.HasInputBoundary = true
}

func decodeLegacyInputBoundary(data []byte, meta *PersistedGenerationMetadata) {
	if match := inputBoundaryPattern.FindSubmatch(data); len(match) > 1 {
		if s, err := strconv.Atoi(string(match[1])); err == nil {
			meta.InputBoundaryStepIndex = s
			meta.HasInputBoundary = true
		}
	}
}

// decodePersistedUsageFields follows only the observed nested wire paths. The
// former generic recursive decoder treated arbitrary text as a protobuf
// sub-message and overwrote repeated fields, which was both costly and an
// unreliable source for telemetry claims.
func decodePersistedUsageFields(data []byte, meta *PersistedGenerationMetadata) {
	root, err := decodeProtoFields(data)
	if err != nil {
		return
	}
	levelOne, ok := nestedProtoFields(root, 1)
	if !ok {
		return
	}

	decodeContextUsage(levelOne, meta)
	decodeMeteredUsage(levelOne, meta)
}

func decodeContextUsage(levelOne []protoField, meta *PersistedGenerationMetadata) {
	levelNine, ok := nestedProtoFields(levelOne, 9)
	if !ok {
		return
	}
	levelTen, ok := nestedProtoFields(levelNine, 10)
	if !ok {
		return
	}
	if total, found := protoVarintValue(levelTen, 1); found {
		meta.ObservedContextTokens = int(total)
		meta.HasObservedContextTokens = true
	}
	if limit, found := protoVarintValue(levelTen, 4); found && limit > 0 {
		meta.ContextLimit = int(limit)
		meta.HasContextLimit = true
	}
}

func decodeMeteredUsage(levelOne []protoField, meta *PersistedGenerationMetadata) {
	levelFour, directUsageFound := nestedProtoFields(levelOne, 4)
	if directUsageFound {
		decodeMeteredUsageFields(levelFour, meta)
		return
	}

	levelSeventeen, ok := nestedProtoFields(levelOne, 17)
	if !ok {
		return
	}
	levelTwo, ok := nestedProtoFields(levelSeventeen, 2)
	if !ok {
		return
	}
	decodeMeteredUsageFields(levelTwo, meta)
}

func decodeMeteredUsageFields(fields []protoField, meta *PersistedGenerationMetadata) {
	if input, found := protoVarintValue(fields, persistedUsageInputFieldNumber); found {
		meta.MeteredInputTokens = int(input)
		meta.HasMeteredInputTokens = true
	}
	if cached, found := protoVarintValue(fields, persistedUsageCachedContentFieldNumber); found {
		meta.CachedContentTokens = int(cached)
		meta.HasCachedContentTokens = true
	}
}

func nestedProtoFields(fields []protoField, fieldNumber int) ([]protoField, bool) {
	bytes := firstBytes(fields, fieldNumber)
	if len(bytes) == 0 {
		return nil, false
	}
	nested, err := decodeProtoFields(bytes)
	if err != nil {
		return nil, false
	}
	return nested, true
}

func protoVarintValue(fields []protoField, fieldNumber int) (uint64, bool) {
	for _, field := range fields {
		if field.number == fieldNumber && field.wireType == 0 {
			return field.integer, true
		}
	}
	return 0, false
}

func readVarint(buf []byte) (uint64, int) {
	var value uint64
	var shift uint
	for index, byteValue := range buf {
		value |= uint64(byteValue&0x7f) << shift
		if byteValue&0x80 == 0 {
			return value, index + 1
		}
		shift += 7
		if shift >= 64 {
			return 0, 0
		}
	}
	return 0, 0
}
