package antigravity

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	inputBoundaryPattern = regexp.MustCompile(`last_step_index\x12[\x01-\x08](\d+)`)
	modelNamePattern     = regexp.MustCompile(`(gemini-[a-zA-Z0-9\.\-]+)`)
)

// PersistedGenerationMetadata contains schema-inferred observations from an
// Antigravity generation metadata blob. The local wire format is not an
// official public protobuf schema.
type PersistedGenerationMetadata struct {
	GenIndex               int
	InputBoundaryStepIndex int
	HasInputBoundary       bool
	TotalTokens            int
	HasTotalTokens         bool
	CachedTokens           int
	HasCachedTokens        bool
	ContextLimit           int
	HasContextLimit        bool
	ModelName              string
}

// ParsePersistedGenerationMetadata decodes the raw protobuf blob from gen_metadata.
// Compiling the observed text patterns once is material during long-session
// hydration because this function may inspect thousands of rows.
func ParsePersistedGenerationMetadata(genIndex int, data []byte) (*PersistedGenerationMetadata, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty protobuf data")
	}

	meta := &PersistedGenerationMetadata{GenIndex: genIndex}

	// Extract the persisted last_step_index input boundary from the raw bytes.
	// It identifies the final transcript input included by the generation, not
	// the generated model-output step itself.
	if match := inputBoundaryPattern.FindSubmatch(data); len(match) > 1 {
		if s, err := strconv.Atoi(string(match[1])); err == nil {
			meta.InputBoundaryStepIndex = s
			meta.HasInputBoundary = true
		}
	}

	// Extract ModelName
	if match := modelNamePattern.FindSubmatch(data); len(match) > 1 {
		meta.ModelName = string(match[1])
	}

	decodePersistedUsageFields(data, meta)

	return meta, nil
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
	decodeCachedUsage(levelOne, meta)
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
		meta.TotalTokens = int(total)
		meta.HasTotalTokens = true
	}
	if limit, found := protoVarintValue(levelTen, 4); found && limit > 0 {
		meta.ContextLimit = int(limit)
		meta.HasContextLimit = true
	}
}

func decodeCachedUsage(levelOne []protoField, meta *PersistedGenerationMetadata) {
	levelFour, directCacheFound := nestedProtoFields(levelOne, 4)
	if directCacheFound {
		if cached, found := protoVarintValue(levelFour, 5); found {
			meta.CachedTokens = int(cached)
			meta.HasCachedTokens = true
			return
		}
	}

	levelSeventeen, ok := nestedProtoFields(levelOne, 17)
	if !ok {
		return
	}
	levelTwo, ok := nestedProtoFields(levelSeventeen, 2)
	if !ok {
		return
	}
	if cached, found := protoVarintValue(levelTwo, 5); found {
		meta.CachedTokens = int(cached)
		meta.HasCachedTokens = true
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
