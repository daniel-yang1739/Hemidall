package antigravity

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"heimdall/internal/core"
)

func TestParseContextSnapshot_Positive_PreservesPersistedSectionsAndRepeatedFields(t *testing.T) {
	system := "<identity>Pair programmer</identity><user_rules>English code</user_rules><skills>review</skills>繁體中文"
	tool := protoMessage(
		protoString(1, "view_file"),
		protoString(2, "Reads a file"),
		protoString(3, `{"type":"object"}`),
	)
	context := protoMessage(
		protoString(1, system),
		protoBytes(2, protoMessage(protoVarint(2, 1), protoString(3, "<CONTEXT_SUMMARY>older history</CONTEXT_SUMMARY>"), protoVarint(18, 41))),
		protoBytes(2, protoMessage(protoVarint(2, 2), protoString(3, "model output"), protoString(11, "private"), protoVarint(18, 42))),
		protoBytes(8, tool),
	)
	raw := protoBytes(1, context)

	snapshot, err := ParseContextSnapshot(7, raw)
	if err != nil {
		t.Fatalf("ParseContextSnapshot returned an error: %v", err)
	}
	if snapshot.GenIndex != 7 || snapshot.BlobBytes != len(raw) {
		t.Errorf("unexpected snapshot identity: index=%d bytes=%d", snapshot.GenIndex, snapshot.BlobBytes)
	}
	if snapshot.Identity != "Pair programmer" || snapshot.UserRules != "English code" {
		t.Errorf("persisted system sections were not preserved: identity=%q rules=%q", snapshot.Identity, snapshot.UserRules)
	}
	if snapshot.SkillsSection != "review" || !strings.Contains(snapshot.SystemPrompt, "繁體中文") {
		t.Errorf("Unicode system prompt or skills section was lost: %#v", snapshot)
	}
	if snapshot.PersistedContextEntryCount != 2 {
		t.Errorf("expected two repeated context entries, got %d", snapshot.PersistedContextEntryCount)
	}
	if len(snapshot.PersistedRecords) != 2 || !snapshot.PersistedRecords[0].IsCompactedCheckpoint || snapshot.PersistedRecords[1].PrimaryText != "model output" {
		t.Errorf("persisted records were not safely decoded: %#v", snapshot.PersistedRecords)
	}
	if !snapshot.PersistedRecords[1].HasPrivateContent || snapshot.PersistedRecords[1].ObservedSequence != 42 {
		t.Errorf("record metadata was not retained safely: %#v", snapshot.PersistedRecords[1])
	}
	if len(snapshot.Tools) != 1 || snapshot.Tools[0].Name != "view_file" || snapshot.Tools[0].RawSchema == "" {
		t.Errorf("persisted tool definition was not decoded: %#v", snapshot.Tools)
	}
}

func TestParseContextSnapshot_Negative_RejectsInvalidSnapshots(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "malformed wire", data: []byte{0xff}},
		{name: "missing context field", data: protoString(2, "metadata")},
		{name: "missing system prompt", data: protoBytes(1, protoBytes(2, protoString(3, "history")))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseContextSnapshot(1, test.data)
			if err == nil {
				t.Errorf("expected invalid snapshot to fail")
			}
		})
	}
}

func TestParseContextSnapshot_Boundary_SkipsMalformedToolWithoutDroppingSnapshot(t *testing.T) {
	context := protoMessage(
		protoString(1, "<identity>Valid</identity>"),
		protoBytes(8, []byte{0xff}),
		protoBytes(8, protoMessage(protoString(1, "valid_tool"))),
	)
	snapshot, err := ParseContextSnapshot(3, protoBytes(1, context))
	if err != nil {
		t.Fatalf("ParseContextSnapshot returned an error: %v", err)
	}
	if len(snapshot.Tools) != 1 || snapshot.Tools[0].Name != "valid_tool" {
		t.Errorf("expected only the valid tool, got %#v", snapshot.Tools)
	}
}

func TestExtractAgentContextPayload_Fallback_LabelsObservedToolSource(t *testing.T) {
	history := []core.UnifiedAgentEvent{{
		Type: core.StepTypeToolCall,
		ToolCalls: []core.ToolCallInfo{{
			ToolName:  "read_file",
			Arguments: map[string]interface{}{"path": "README.md"},
		}},
	}}

	payload := ExtractAgentContextPayload(history, "", "")
	if payload.SourceKind != core.ContextEvidenceTranscriptFallback {
		t.Fatalf("expected transcript fallback, got %q", payload.SourceKind)
	}
	if payload.Provenance["tools"] != "inferred_from_observed_tool_arguments" {
		t.Errorf("unexpected fallback tool provenance: %q", payload.Provenance["tools"])
	}
	if payload.Provenance["latest_prompt"] != "observed_from_antigravity_session_history" {
		t.Errorf("unexpected fallback prompt provenance: %q", payload.Provenance["latest_prompt"])
	}
	if payload.Provenance["tokens"] != "latest_positive_history_token_breakdown; may be telemetry or Heimdall-derived" {
		t.Errorf("unexpected fallback token provenance: %q", payload.Provenance["tokens"])
	}

	raw, err := core.SerializeSubcategoryRaw(payload, core.SubcatTools)
	if err != nil {
		t.Fatalf("SerializeSubcategoryRaw returned an error: %v", err)
	}
	if !strings.Contains(raw, `"source": "inferred_from_observed_tool_arguments"`) {
		t.Errorf("raw fallback tool evidence omitted provenance: %s", raw)
	}
}

func protoMessage(fields ...[]byte) []byte {
	return bytes.Join(fields, nil)
}

func protoString(number int, value string) []byte {
	return protoBytes(number, []byte(value))
}

func protoBytes(number int, value []byte) []byte {
	var tag [binary.MaxVarintLen64]byte
	tagLength := binary.PutUvarint(tag[:], uint64(number<<3|2))
	var size [binary.MaxVarintLen64]byte
	sizeLength := binary.PutUvarint(size[:], uint64(len(value)))
	result := make([]byte, 0, tagLength+sizeLength+len(value))
	result = append(result, tag[:tagLength]...)
	result = append(result, size[:sizeLength]...)
	return append(result, value...)
}

func protoVarint(number int, value uint64) []byte {
	var tag [binary.MaxVarintLen64]byte
	tagLength := binary.PutUvarint(tag[:], uint64(number<<3))
	var encoded [binary.MaxVarintLen64]byte
	encodedLength := binary.PutUvarint(encoded[:], value)
	result := make([]byte, 0, tagLength+encodedLength)
	result = append(result, tag[:tagLength]...)
	return append(result, encoded[:encodedLength]...)
}
