package schema

import (
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	modelUsageTotalOutputField    = 3
	modelUsageThinkingField       = 9
	modelUsageContentOutputField  = 10
	modelUsageSplitTotalTokens    = 2_750
	modelUsageSplitThinkingTokens = 1_190
	modelUsageSplitContentTokens  = 1_560
	modelUsageContentOnlyTokens   = 133
)

func TestMessageRole_StringAndDescription(t *testing.T) {
	cases := []struct {
		role         MessageRole
		expectedStr  string
		expectedDesc string
	}{
		{RoleUser, "USER", "User Request / Context Summary"},
		{RoleAssistant, "ASSISTANT", "Model Turn / Tool Call"},
		{RoleToolResult, "TOOL_RESULT", "Tool Execution Output"},
		{RoleSystem, "SYSTEM", "System Prompt"},
		{MessageRole(99), "UNKNOWN_99", "Unspecified Role"},
	}

	for _, c := range cases {
		t.Run(c.expectedStr, func(t *testing.T) {
			if s := c.role.String(); s != c.expectedStr {
				t.Errorf("expected string %q, got %q", c.expectedStr, s)
			}
			if d := c.role.Description(); d != c.expectedDesc {
				t.Errorf("expected desc %q, got %q", c.expectedDesc, d)
			}
		})
	}
}

func TestDecodeDuration(t *testing.T) {
	// Field 1: sec=4 (varint), Field 2: nanos=500000000 (varint)
	// tag 1: (1<<3)|0 = 0x08, val 4 = 0x04
	// tag 2: (2<<3)|0 = 0x10, val 500000000 = 0x80, 0x94, 0xeb, 0xee, 0x01
	data := []byte{0x08, 0x04, 0x10, 0x80, 0xca, 0xb5, 0xee, 0x01}
	dur := decodeDuration(data)
	expected := 4*time.Second + 500*time.Millisecond
	if dur != expected {
		t.Fatalf("expected duration %v, got %v", expected, dur)
	}
}

func TestDecodeModelUsage_Boundary_SeparatesTotalThinkingAndContentOutput(t *testing.T) {
	tests := []struct {
		name                string
		data                []byte
		expectedTotal       int64
		expectedThinking    int64
		expectedContent     int64
		expectedHasTotal    bool
		expectedHasThinking bool
		expectedHasContent  bool
	}{
		{
			name: "split output",
			data: concatUsageFields(
				usageVarintField(modelUsageTotalOutputField, modelUsageSplitTotalTokens),
				usageVarintField(modelUsageThinkingField, modelUsageSplitThinkingTokens),
				usageVarintField(modelUsageContentOutputField, modelUsageSplitContentTokens),
			),
			expectedTotal: modelUsageSplitTotalTokens, expectedThinking: modelUsageSplitThinkingTokens, expectedContent: modelUsageSplitContentTokens,
			expectedHasTotal: true, expectedHasThinking: true, expectedHasContent: true,
		},
		{
			name: "content without thinking",
			data: concatUsageFields(
				usageVarintField(modelUsageTotalOutputField, modelUsageContentOnlyTokens),
				usageVarintField(modelUsageContentOutputField, modelUsageContentOnlyTokens),
			),
			expectedTotal: modelUsageContentOnlyTokens, expectedContent: modelUsageContentOnlyTokens,
			expectedHasTotal: true, expectedHasThinking: false, expectedHasContent: true,
		},
		{name: "missing output", data: []byte{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			usage, err := DecodeModelUsage(test.data)
			if err != nil {
				t.Fatal(err)
			}
			if usage.TotalOutputTokens != test.expectedTotal || usage.ThinkingOutputTokens != test.expectedThinking || usage.OutputContentTokens != test.expectedContent {
				t.Fatalf("decoded output tokens: total=%d thinking=%d content=%d", usage.TotalOutputTokens, usage.ThinkingOutputTokens, usage.OutputContentTokens)
			}
			if usage.HasTotalOutputTokens != test.expectedHasTotal || usage.HasThinkingOutputTokens != test.expectedHasThinking || usage.HasOutputContentTokens != test.expectedHasContent {
				t.Fatalf("decoded output availability: total=%t thinking=%t content=%t", usage.HasTotalOutputTokens, usage.HasThinkingOutputTokens, usage.HasOutputContentTokens)
			}
		})
	}
}

func usageVarintField(number protowire.Number, value uint64) []byte {
	data := protowire.AppendTag(nil, number, protowire.VarintType)
	return protowire.AppendVarint(data, value)
}

func concatUsageFields(fields ...[]byte) []byte {
	data := make([]byte, 0)
	for _, field := range fields {
		data = append(data, field...)
	}
	return data
}
