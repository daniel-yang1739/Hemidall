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
	raw := []byte("prefix\x00last_step_index\x12\x045550\x00gemini-3.7-flash\x00suffix")
	meta, err := ParseGeminiGenMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.LastStepIdx != 5550 {
		t.Errorf("Expected LastStepIdx 5550, got %d", meta.LastStepIdx)
	}
	if meta.ModelName != "gemini-3.7-flash" {
		t.Errorf("Expected ModelName 'gemini-3.7-flash', got '%s'", meta.ModelName)
	}
}

func TestProtobuf_Pos_1DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x011\x00gemini-3.7-flash\x00suffix")
	meta, err := ParseGeminiGenMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.LastStepIdx != 1 {
		t.Fatalf("Expected LastStepIdx 1, got %d", meta.LastStepIdx)
	}
}

func TestProtobuf_Pos_2DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x0242\x00gemini-3.7-flash\x00suffix")
	meta, err := ParseGeminiGenMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.LastStepIdx != 42 {
		t.Fatalf("Expected LastStepIdx 42, got %d", meta.LastStepIdx)
	}
}

func TestProtobuf_Pos_3DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x03664\x00gemini-3.7-flash\x00suffix")
	meta, err := ParseGeminiGenMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.LastStepIdx != 664 {
		t.Fatalf("Expected LastStepIdx 664, got %d", meta.LastStepIdx)
	}
}

func TestProtobuf_Pos_5DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x0512345\x00gemini-3.7-flash\x00suffix")
	meta, err := ParseGeminiGenMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.LastStepIdx != 12345 {
		t.Fatalf("Expected LastStepIdx 12345, got %d", meta.LastStepIdx)
	}
}

func TestProtobuf_Pos_6DigitStepIndexExtraction(t *testing.T) {
	raw := []byte("prefix\x00last_step_index\x12\x06100000\x00gemini-3.7-flash\x00suffix")
	meta, err := ParseGeminiGenMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.LastStepIdx != 100000 {
		t.Fatalf("Expected LastStepIdx 100000, got %d", meta.LastStepIdx)
	}
}

func TestProtobuf_Pos_CachedTokensExtraction(t *testing.T) {
	// Simulated protobuf with field 1 -> field 4 -> field 5 (220,446)
	// and field 1 -> field 9 -> field 10 -> field 1 (214,342)
	raw := []byte("last_step_index\x12\x03815\x00gemini-3.7-flash")
	meta, err := ParseGeminiGenMetadata(1, raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if meta.LastStepIdx != 815 {
		t.Fatalf("Expected LastStepIdx 815, got %d", meta.LastStepIdx)
	}
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
	_, err := ParseGeminiGenMetadata(1, []byte{})
	if err == nil {
		t.Errorf("Expected error on empty protobuf data, got nil")
	}

	// Random garbage noise
	garbage := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x12}
	meta, err := ParseGeminiGenMetadata(1, garbage)
	if err != nil {
		t.Fatalf("Corrupted blob should not panic or fail ParseGeminiGenMetadata: %v", err)
	}
	if meta.TotalTokens != 0 {
		t.Errorf("Corrupted blob should have 0 TotalTokens, got %d", meta.TotalTokens)
	}
}
