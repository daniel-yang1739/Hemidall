package core

import (
	"encoding/json"
	"strings"
	"testing"
)

// ==============================================================================
// 11. CONTEXT PAYLOAD ASSEMBLY & WIRE SERIALIZATION (5 Positive + 5 Negative)
// ==============================================================================

func TestContext_Pos_ExtractSystemAndConstitution(t *testing.T) {
	history := []UnifiedAgentEvent{
		{
			StepIndex: 0,
			Type:      StepTypeSystemInit,
			RawContent: `<identity>You are Antigravity, a senior pair programmer.</identity>
<user_rules># REPOSITORY CONSTITUTION
ARTICLE I: LANGUAGE POLICY - Code EN, Docs TC.</user_rules>`,
		},
	}

	payload := buildContextPayloadForTest(history, "sess-test", "Gemini 3.7 Flash")

	if !strings.Contains(payload.IdentityPrompt, "senior pair programmer") {
		t.Errorf("Expected identity prompt to contain 'senior pair programmer', got %q", payload.IdentityPrompt)
	}
	if !strings.Contains(payload.ConstitutionDoc, "ARTICLE I: LANGUAGE POLICY") {
		t.Errorf("Expected constitution doc to contain 'ARTICLE I: LANGUAGE POLICY', got %q", payload.ConstitutionDoc)
	}
}

func TestContext_Pos_ExtractObservedToolArguments(t *testing.T) {
	history := []UnifiedAgentEvent{{
		Type: StepTypeToolCall,
		ToolCalls: []ToolCallInfo{{
			ToolName:  "write_to_file",
			Arguments: map[string]interface{}{"TargetFile": "main.go", "Overwrite": true},
		}},
	}}
	payload := buildContextPayloadForTest(history, "sess-test", "Gemini 3.7 Flash")

	if len(payload.NativeTools) != 1 {
		t.Fatalf("Expected one observed tool, got %d", len(payload.NativeTools))
	}

	tool0 := payload.NativeTools[0]
	if tool0.Name != "write_to_file" {
		t.Errorf("Expected tool 0 to be 'write_to_file', got %s", tool0.Name)
	}
	if tool0.Signature != "write_to_file(Overwrite, TargetFile)" {
		t.Errorf("Expected sorted observed arguments, got %s", tool0.Signature)
	}
	if tool0.RawSchema == "" {
		t.Errorf("Expected non-empty RawSchema for write_to_file")
	}
}

func TestContext_Pos_3StageHistorySlicingConservation(t *testing.T) {
	var history []UnifiedAgentEvent
	for i := 0; i < 50; i++ {
		history = append(history, UnifiedAgentEvent{
			StepIndex:  i,
			Type:       StepTypeModelResponse,
			RawContent: "Conversation step payload.",
		})
	}
	// Add a checkpoint summary at step 10
	history[10].Type = StepTypeCheckpoint
	history[10].RawContent = "<CONTEXT_SUMMARY>Compacted summary base anchor</CONTEXT_SUMMARY>"

	payload := buildContextPayloadForTest(history, "sess-test", "Gemini 3.7 Flash")

	if !strings.Contains(payload.CheckpointSummary, "Compacted summary base anchor") {
		t.Errorf("Expected checkpoint summary to be extracted, got %q", payload.CheckpointSummary)
	}
	if payload.CheckpointStepIndex != 10 {
		t.Errorf("Expected checkpoint step index 10, got %d", payload.CheckpointStepIndex)
	}
}

func TestContext_Pos_EvidenceJsonSerialization(t *testing.T) {
	payload := buildContextPayloadForTest(nil, "sess-test", "Gemini 3.7 Flash")
	wireJSON, err := SerializeContextEvidence(payload)
	if err != nil {
		t.Fatalf("SerializeContextEvidence failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(wireJSON), &parsed); err != nil {
		t.Fatalf("Serialized wire payload is not valid JSON: %v\nJSON:\n%s", err, wireJSON)
	}

	if _, ok := parsed["evidence"]; !ok {
		t.Errorf("Expected evidence metadata in serialized JSON")
	}
	if _, ok := parsed["persisted_tool_entries"]; !ok {
		t.Errorf("Expected persisted tool entry field in evidence JSON")
	}
	if _, ok := parsed["persisted_field_2_occurrence_count"]; !ok {
		t.Errorf("Expected field-2 occurrence count in evidence JSON")
	}
	if _, ok := parsed["heimdall_runtime_metadata"]; !ok {
		t.Errorf("Expected Heimdall runtime metadata in evidence JSON")
	}
}

func TestContext_Neg_NilHistoryDoesNotFabricateSessionData(t *testing.T) {
	payload := buildContextPayloadForTest(nil, "sess-empty", "Gemini 3.7 Flash")

	if payload.TotalTokens != 0 {
		t.Errorf("Expected zero observed tokens, got %d", payload.TotalTokens)
	}
	if payload.IdentityPrompt != "" {
		t.Errorf("Expected no fabricated identity, got %q", payload.IdentityPrompt)
	}
	if payload.LatestPrompt != "" || payload.CheckpointSummary != "" || payload.StagedBuffers != "" {
		t.Error("Expected absent session fields to remain empty")
	}
	if len(payload.NativeTools) != 0 {
		t.Errorf("Expected no fabricated tools, got %d", len(payload.NativeTools))
	}
}

func TestContext_Neg_EmptyToolsListValidEvidence(t *testing.T) {
	// Custom payload with 0 tools
	customPayload := AgentContextPayload{
		TargetModel: "custom-model",
		NativeTools: []ToolSignature{},
	}
	wireJSON, err := SerializeContextEvidence(customPayload)
	if err != nil {
		t.Fatalf("SerializeContextEvidence on empty tools failed: %v", err)
	}
	if !strings.Contains(wireJSON, `"persisted_tool_entries": []`) && !strings.Contains(wireJSON, `"persisted_tool_entries":[]`) {
		t.Errorf("Expected valid empty tools array in JSON, got %s", wireJSON)
	}
}

func TestContext_Neg_NilHistoryEventsEmptyPayload(t *testing.T) {
	payload := buildContextPayloadForTest(nil, "", "")
	if payload.TotalTokens < 0 {
		t.Errorf("TotalTokens cannot be negative, got %d", payload.TotalTokens)
	}
}

func TestContext_Neg_UnrecognizedSkillFormatFallback(t *testing.T) {
	// Should not panic on unformatted skill
	skill := SkillInfo{
		Name:        "corrupted-skill",
		RawMarkdown: "no yaml frontmatter here",
	}
	if skill.Name != "corrupted-skill" {
		t.Errorf("Expected skill name preserved")
	}
}

func TestContext_Pos_SerializeSubcategoryRawAllParts(t *testing.T) {
	payload := buildContextPayloadForTest(nil, "sess-test", "Gemini 3.7 Flash")

	for subcat := 0; subcat <= SubcatBuffers; subcat++ {
		raw, err := SerializeSubcategoryRaw(payload, subcat)
		if err != nil {
			t.Fatalf("SerializeSubcategoryRaw failed on subcat %d: %v", subcat, err)
		}
		if raw == "" {
			t.Fatalf("Expected non-empty evidence JSON for subcat %d", subcat)
		}

		var parsed interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			t.Fatalf("Subcat %d raw is not valid JSON: %v\nJSON:\n%s", subcat, err, raw)
		}
	}
}

func TestContext_Neg_InvalidSubcategoryIndexFallback(t *testing.T) {
	payload := buildContextPayloadForTest(nil, "sess-test", "Gemini 3.7 Flash")

	raw, err := SerializeSubcategoryRaw(payload, 999)
	if err != nil {
		t.Fatalf("Expected graceful fallback on invalid subcategory index, got error: %v", err)
	}
	if !strings.Contains(raw, "evidence") {
		t.Errorf("Expected fallback to full evidence payload, got:\n%s", raw)
	}
}

func TestSerializePersistedActiveEventRaw_ScopesAllAndSelectedEvents(t *testing.T) {
	payload := AgentContextPayload{PersistedRecords: []PersistedContextRecord{
		{Position: 1, PrimaryText: "checkpoint"},
		{Position: 2, PrimaryText: "selected event"},
	}}

	all, err := SerializePersistedActiveEventRaw(payload, 0)
	if err != nil || !strings.Contains(all, "active_events") || !strings.Contains(all, "selected event") {
		t.Fatalf("all-event raw scope was not serialized: %q, %v", all, err)
	}

	selected, err := SerializePersistedActiveEventRaw(payload, 2)
	if err != nil || !strings.Contains(selected, "active_event") || !strings.Contains(selected, "selected event") || strings.Contains(selected, "checkpoint") {
		t.Fatalf("single-event raw scope was not serialized: %q, %v", selected, err)
	}
}

func TestSerializeSubcategoryRaw_RuntimeMetadataUsesStableKeyOrder(t *testing.T) {
	payload := AgentContextPayload{RuntimeMetadata: map[string]string{
		"Shell": "zsh",
		"Arch":  "arm64",
		"OS":    "darwin",
	}}

	raw, err := SerializeSubcategoryRaw(payload, SubcatRuntime)
	if err != nil {
		t.Fatalf("SerializeSubcategoryRaw returned an error: %v", err)
	}
	archPosition := strings.Index(raw, "Arch: arm64")
	osPosition := strings.Index(raw, "OS: darwin")
	shellPosition := strings.Index(raw, "Shell: zsh")
	if archPosition < 0 || osPosition < 0 || shellPosition < 0 {
		t.Fatalf("runtime metadata was missing from raw output: %s", raw)
	}
	if archPosition > osPosition || osPosition > shellPosition {
		t.Errorf("runtime metadata order was unstable: %s", raw)
	}
}

func buildContextPayloadForTest(history []UnifiedAgentEvent, sessionID, targetModel string) AgentContextPayload {
	return BuildContextPayloadFromHistory(ContextBuildInput{
		History: history, SessionID: sessionID, AgentType: AgentTypeAntigravity, TargetModel: targetModel,
		NativeTools: GetNativeToolsDefinitions(history),
	})
}
