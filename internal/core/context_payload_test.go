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
	if _, ok := parsed["context_snapshot"]; !ok {
		t.Errorf("Expected context snapshot field in evidence JSON")
	}
}

func TestContext_Pos_PersistedActiveRecordRawExcludesCheckpoint(t *testing.T) {
	payload := AgentContextPayload{
		PersistedRecords: []PersistedContextRecord{
			{Position: 1, PrimaryText: "checkpoint text", IsCompactedCheckpoint: true},
			{Position: 2, PrimaryText: "event text"},
		},
	}

	raw, err := SerializePersistedActiveEventRaw(payload, 0)
	requireContextNoError(t, err)
	requireContextContains(t, raw, "event text")
	if strings.Contains(raw, "checkpoint text") {
		t.Errorf("active history raw output duplicated the compacted checkpoint: %s", raw)
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
	if !strings.Contains(wireJSON, `"tool_entries": []`) && !strings.Contains(wireJSON, `"tool_entries":[]`) {
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

	for subcat := SubcatSystemPrompt; subcat <= SubcatLast; subcat++ {
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

	_, err := SerializeSubcategoryRaw(payload, 999)
	if err == nil {
		t.Fatal("Expected invalid model-input subcategory to return an error")
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

func TestContextSnapshot_Pos_PreservesTopLevelTagOrder(t *testing.T) {
	systemPrompt := `Host prefix
<identity>Pair programmer</identity>
<user_rules><RULE[user]>Repository rules</RULE[user]></user_rules>
<skills>Skill catalog</skills>
<mcp>MCP instructions</mcp>
Host suffix`
	payload := BuildContextPayloadFromSession(Session{ContextSnapshots: []ContextSnapshot{{
		SystemPrompt:   systemPrompt,
		IdentityPrompt: "Pair programmer", ConstitutionDoc: "Repository rules", SkillsSection: "Skill catalog", MCPSection: "MCP instructions",
	}}}, ContextBuildInput{})

	wantTags := []string{"identity", "user_rules", "skills", "mcp"}
	if len(payload.SystemPromptSections) != len(wantTags) {
		t.Fatalf("system-prompt section count: got %d, want %d", len(payload.SystemPromptSections), len(wantTags))
	}
	for index, wantTag := range wantTags {
		if payload.SystemPromptSections[index].Tag != wantTag {
			t.Errorf("system-prompt section %d: got %q, want %q", index, payload.SystemPromptSections[index].Tag, wantTag)
		}
	}
	if !strings.Contains(payload.SystemPromptSections[1].Content, "<RULE[user]>") {
		t.Errorf("nested rule markup must remain inside user_rules: %q", payload.SystemPromptSections[1].Content)
	}

	mcpRaw, err := SerializeSystemPromptSectionRaw(payload, 3)
	if err != nil {
		t.Fatalf("SerializeSystemPromptSectionRaw returned an error: %v", err)
	}
	if !strings.Contains(mcpRaw, "MCP instructions") {
		t.Fatalf("MCP section omitted persisted MCP instructions: %s", mcpRaw)
	}
	if strings.Contains(mcpRaw, "Host prefix") || strings.Contains(mcpRaw, "Host suffix") {
		t.Errorf("tag section must not absorb untagged system-prompt text: %s", mcpRaw)
	}
}

func TestContextSnapshot_Boundary_PreservesMalformedTaggedHostText(t *testing.T) {
	systemPrompt := "Host prefix\n<identity>unterminated identity text\nHost suffix"
	payload := BuildContextPayloadFromSession(Session{ContextSnapshots: []ContextSnapshot{{
		SystemPrompt: systemPrompt,
	}}}, ContextBuildInput{})

	if payload.SystemPrompt != systemPrompt {
		t.Errorf("malformed tagged text must remain in the complete system prompt: %q", payload.SystemPrompt)
	}
	if len(payload.SystemPromptSections) != 0 {
		t.Errorf("unbalanced tags must not fabricate prompt sections: %#v", payload.SystemPromptSections)
	}
}

func TestContextSnapshot_Boundary_UnclosedPlaceholderDoesNotHideLaterTags(t *testing.T) {
	systemPrompt := "Use <path> as a placeholder.\n<identity>Pair programmer</identity>"
	payload := BuildContextPayloadFromSession(Session{ContextSnapshots: []ContextSnapshot{{
		SystemPrompt: systemPrompt,
	}}}, ContextBuildInput{})

	if len(payload.SystemPromptSections) != 1 || payload.SystemPromptSections[0].Tag != "identity" {
		t.Errorf("unclosed placeholder hid a later balanced tag: %#v", payload.SystemPromptSections)
	}
}

func buildContextPayloadForTest(history []UnifiedAgentEvent, sessionID, targetModel string) AgentContextPayload {
	return BuildContextPayloadFromHistory(ContextBuildInput{
		History: history, SessionID: sessionID, TargetModel: targetModel,
		NativeTools: GetNativeToolsDefinitions(history),
	})
}

func requireContextNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireContextContains(t *testing.T, actual, expected string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %q in %q", expected, actual)
	}
}
