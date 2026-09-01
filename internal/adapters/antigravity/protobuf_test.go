package antigravity

import (
	"testing"
)

// ==============================================================================
// 9. SQLITE PROTOBUF TELEMETRY TESTS (2 Positive + 2 Negative)
// ==============================================================================

func TestProtobuf_Pos_StandardSingleAndMultiByteVarint(t *testing.T) {
	// Single byte: 1
	buf1 := []byte{0x01}
	val1, n1 := readVarint(buf1)
	if val1 != 1 || n1 != 1 {
		t.Errorf("readVarint(0x01) = (%d, %d), expected (1, 1)", val1, n1)
	}

	// Multi-byte: 300 (0xAC, 0x02)
	buf300 := []byte{0xAC, 0x02}
	val300, n300 := readVarint(buf300)
	if val300 != 300 || n300 != 2 {
		t.Errorf("readVarint(300) = (%d, %d), expected (300, 2)", val300, n300)
	}
}

func TestProtobuf_Pos_ValidTelemetryBlobExtraction(t *testing.T) {
	raw := persistedModelFixture(persistedGeminiModelID, persistedGeminiEnum, "5550")
	meta, err := ParsePersistedGenerationMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.InputBoundaryStepIndex != 5550 {
		t.Errorf("Expected InputBoundaryStepIndex 5550, got %d", meta.InputBoundaryStepIndex)
	}
	if !meta.HasInputBoundary {
		t.Error("Expected input boundary marker to be present")
	}
	if meta.ModelName != "gemini-3.7-flash" {
		t.Errorf("Expected ModelName 'gemini-3.7-flash', got '%s'", meta.ModelName)
	}
}

func TestProtobuf_Pos_1DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x011\x00gemini-3.7-flash\x00suffix")
	meta, err := ParsePersistedGenerationMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.InputBoundaryStepIndex != 1 {
		t.Fatalf("Expected InputBoundaryStepIndex 1, got %d", meta.InputBoundaryStepIndex)
	}
}

func TestProtobuf_Pos_2DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x0242\x00gemini-3.7-flash\x00suffix")
	meta, err := ParsePersistedGenerationMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.InputBoundaryStepIndex != 42 {
		t.Fatalf("Expected InputBoundaryStepIndex 42, got %d", meta.InputBoundaryStepIndex)
	}
}

func TestProtobuf_Pos_3DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x03664\x00gemini-3.7-flash\x00suffix")
	meta, err := ParsePersistedGenerationMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.InputBoundaryStepIndex != 664 {
		t.Fatalf("Expected InputBoundaryStepIndex 664, got %d", meta.InputBoundaryStepIndex)
	}
}

func TestProtobuf_Pos_5DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x0512345\x00gemini-3.7-flash\x00suffix")
	meta, err := ParsePersistedGenerationMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.InputBoundaryStepIndex != 12345 {
		t.Fatalf("Expected InputBoundaryStepIndex 12345, got %d", meta.InputBoundaryStepIndex)
	}
}

func TestProtobuf_Pos_6DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x06100000\x00gemini-3.7-flash\x00suffix")
	meta, err := ParsePersistedGenerationMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.InputBoundaryStepIndex != 100000 {
		t.Fatalf("Expected InputBoundaryStepIndex 100000, got %d", meta.InputBoundaryStepIndex)
	}
}

func TestProtobuf_Pos_CachedTokensExtraction(t *testing.T) {
	// Simulated protobuf with field 1 -> field 4 -> field 5 (220,446)
	// and field 1 -> field 9 -> field 10 -> field 1 (214,342)
	raw := []byte("last_step_index\x12\x03815\x00gemini-3.7-flash")
	meta, err := ParsePersistedGenerationMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.InputBoundaryStepIndex != 815 {
		t.Fatalf("Expected InputBoundaryStepIndex 815, got %d", meta.InputBoundaryStepIndex)
	}
}

func TestPersistedGenerationMetadata_Pos_DecodesStructuredUsageFields(t *testing.T) {
	raw := protoBytes(1, protoMessage(
		protoBytes(9, protoMessage(
			protoBytes(10, protoMessage(
				protoVarint(1, 120000),
				protoVarint(4, 160000),
			)),
		)),
		protoBytes(4, protoMessage(
			protoVarint(1, 10000),
			protoVarint(5, 110000),
		)),
	))

	metadata, parseErr := ParsePersistedGenerationMetadata(1, raw)

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, true, metadata.HasObservedContextTokens)
	requireAntigravityEqual(t, 120000, metadata.ObservedContextTokens)
	requireAntigravityEqual(t, true, metadata.HasContextLimit)
	requireAntigravityEqual(t, 160000, metadata.ContextLimit)
	requireAntigravityEqual(t, true, metadata.HasMeteredInputTokens)
	requireAntigravityEqual(t, 10000, metadata.MeteredInputTokens)
	requireAntigravityEqual(t, true, metadata.HasCachedContentTokens)
	requireAntigravityEqual(t, 110000, metadata.CachedContentTokens)
}

func TestPersistedGenerationMetadata_Boundary_UsesAlternateCachePath(t *testing.T) {
	raw := protoBytes(1, protoMessage(
		protoBytes(17, protoMessage(
			protoBytes(2, protoMessage(
				protoVarint(1, 5000),
				protoVarint(5, 75000),
			)),
		)),
	))

	metadata, parseErr := ParsePersistedGenerationMetadata(1, raw)

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, true, metadata.HasMeteredInputTokens)
	requireAntigravityEqual(t, 5000, metadata.MeteredInputTokens)
	requireAntigravityEqual(t, true, metadata.HasCachedContentTokens)
	requireAntigravityEqual(t, 75000, metadata.CachedContentTokens)
}

func TestPersistedGenerationMetadata_Boundary_LeavesOmittedCacheScalarAtProtoDefault(t *testing.T) {
	raw := protoBytes(1, protoMessage(
		protoBytes(9, protoMessage(
			protoBytes(10, protoMessage(protoVarint(1, 89344))),
		)),
		protoBytes(4, protoMessage(protoVarint(1, 1298))),
	))

	metadata, parseErr := ParsePersistedGenerationMetadata(1, raw)

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, true, metadata.HasObservedContextTokens)
	requireAntigravityEqual(t, 89344, metadata.ObservedContextTokens)
	requireAntigravityEqual(t, true, metadata.HasMeteredInputTokens)
	requireAntigravityEqual(t, 1298, metadata.MeteredInputTokens)
	requireAntigravityEqual(t, false, metadata.HasCachedContentTokens)
	requireAntigravityEqual(t, 0, metadata.CachedContentTokens)
}

func TestProtobuf_Neg_TruncatedVarintNoInfiniteLoop(t *testing.T) {
	// Truncated varint: MSB is 1 (0x80) but buffer ends immediately
	truncated := []byte{0x80}
	val, n := readVarint(truncated)
	if n != 0 || val != 0 {
		t.Errorf("Truncated varint should return (0, 0), got (%d, %d)", val, n)
	}
}

func TestProtobuf_Neg_CorruptedBlobGracefulSkip(t *testing.T) {
	// Empty data
	_, err := ParsePersistedGenerationMetadata(1, []byte{})
	if err == nil {
		t.Errorf("Expected error on empty protobuf data, got nil")
	}

	// Random garbage noise
	garbage := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x12}
	meta, err := ParsePersistedGenerationMetadata(1, garbage)
	if err != nil {
		t.Fatalf("Corrupted blob should not panic or fail ParsePersistedGenerationMetadata: %v", err)
	}
	if meta.ObservedContextTokens != 0 {
		t.Errorf("Corrupted blob should have 0 ObservedContextTokens, got %d", meta.ObservedContextTokens)
	}
}
