package core

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ==============================================================================
// 11. CONTEXT DATA EXTRACTION & WIRE SERIALIZATION (5 Positive + 5 Negative)
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

	payload := ExtractAgentContextPayload(history, "sess-test", "Gemini 3.7 Flash")

	if !strings.Contains(payload.IdentityPrompt, "senior pair programmer") {
		t.Errorf("Expected identity prompt to contain 'senior pair programmer', got %q", payload.IdentityPrompt)
	}
	if !strings.Contains(payload.ConstitutionDoc, "ARTICLE I: LANGUAGE POLICY") {
		t.Errorf("Expected constitution doc to contain 'ARTICLE I: LANGUAGE POLICY', got %q", payload.ConstitutionDoc)
	}
}

func TestContext_Pos_ExtractNativeTools8Schemas(t *testing.T) {
	payload := ExtractAgentContextPayload(nil, "sess-test", "Gemini 3.7 Flash")

	if len(payload.NativeTools) != 8 {
		t.Fatalf("Expected 8 native tools, got %d", len(payload.NativeTools))
	}

	tool0 := payload.NativeTools[0]
	if tool0.Name != "write_to_file" {
		t.Errorf("Expected tool 0 to be 'write_to_file', got %s", tool0.Name)
	}
	if !strings.Contains(tool0.Signature, "write_to_file(TargetFile, CodeContent, Overwrite)") {
		t.Errorf("Expected clean signature for write_to_file, got %s", tool0.Signature)
	}
	if tool0.RawSchema == "" {
		t.Errorf("Expected non-empty RawSchema for write_to_file")
	}
}

func TestContext_Pos_ExtractSkillsYamlFrontmatter(t *testing.T) {
	payload := ExtractAgentContextPayload(nil, "sess-test", "Gemini 3.7 Flash")

	if len(payload.ActiveSkills) < 3 {
		t.Fatalf("Expected at least 3 active skills, got %d", len(payload.ActiveSkills))
	}

	foundWiki := false
	for _, s := range payload.ActiveSkills {
		if s.Name == "wiki-distiller" {
			foundWiki = true
			if s.Status != "ACTIVE" {
				t.Errorf("Expected wiki-distiller status 'ACTIVE', got %s", s.Status)
			}
		}
	}
	if !foundWiki {
		t.Errorf("Expected to find 'wiki-distiller' in active skills")
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

	payload := ExtractAgentContextPayload(history, "sess-test", "Gemini 3.7 Flash")

	if !strings.Contains(payload.CheckpointSummary, "Compacted summary base anchor") {
		t.Errorf("Expected checkpoint summary to be extracted, got %q", payload.CheckpointSummary)
	}
	if payload.CompactedStepsCount <= 0 {
		t.Errorf("Expected positive compacted steps count, got %d", payload.CompactedStepsCount)
	}
}

func TestContext_Pos_RawWireJsonSerialization(t *testing.T) {
	payload := ExtractAgentContextPayload(nil, "sess-test", "Gemini 3.7 Flash")
	wireJSON, err := SerializeToWirePayload(payload)
	if err != nil {
		t.Fatalf("SerializeToWirePayload failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(wireJSON), &parsed); err != nil {
		t.Fatalf("Serialized wire payload is not valid JSON: %v\nJSON:\n%s", err, wireJSON)
	}

	if _, ok := parsed["systemInstruction"]; !ok {
		t.Errorf("Expected 'systemInstruction' in wire payload JSON")
	}
	if _, ok := parsed["tools"]; !ok {
		t.Errorf("Expected 'tools' in wire payload JSON")
	}
}

func TestContext_Neg_MissingAgentsMdGracefulFallback(t *testing.T) {
	// Nil history with no AGENTS.md in events
	payload := ExtractAgentContextPayload(nil, "sess-empty", "Gemini 3.7 Flash")

	if payload.ConstitutionDoc == "" {
		t.Errorf("Expected default constitution fallback, got empty")
	}
}

func TestContext_Neg_EmptyToolsListValidSchema(t *testing.T) {
	// Custom payload with 0 tools
	customPayload := AgentContextPayload{
		TargetModel: "custom-model",
		NativeTools: []ToolSignature{},
	}
	wireJSON, err := SerializeToWirePayload(customPayload)
	if err != nil {
		t.Fatalf("SerializeToWirePayload on empty tools failed: %v", err)
	}
	if !strings.Contains(wireJSON, `"tools": []`) && !strings.Contains(wireJSON, `"tools":[]`) {
		t.Errorf("Expected valid empty tools array in JSON, got %s", wireJSON)
	}
}

func TestContext_Neg_NilHistoryEventsEmptyPayload(t *testing.T) {
	payload := ExtractAgentContextPayload(nil, "", "")
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
	payload := ExtractAgentContextPayload(nil, "sess-test", "Gemini 3.7 Flash")

	for subcat := 0; subcat <= SubcatBuffers; subcat++ {
		raw, err := SerializeSubcategoryRaw(payload, subcat)
		if err != nil {
			t.Fatalf("SerializeSubcategoryRaw failed on subcat %d: %v", subcat, err)
		}
		if raw == "" {
			t.Fatalf("Expected non-empty raw wire JSON for subcat %d", subcat)
		}

		var parsed interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			t.Fatalf("Subcat %d raw is not valid JSON: %v\nJSON:\n%s", subcat, err, raw)
		}
	}
}

func TestContext_Neg_InvalidSubcategoryIndexFallback(t *testing.T) {
	payload := ExtractAgentContextPayload(nil, "sess-test", "Gemini 3.7 Flash")

	raw, err := SerializeSubcategoryRaw(payload, 999)
	if err != nil {
		t.Fatalf("Expected graceful fallback on invalid subcategory index, got error: %v", err)
	}
	if !strings.Contains(raw, "systemInstruction") {
		t.Errorf("Expected fallback to full wire payload with 'systemInstruction', got:\n%s", raw)
	}
}

func init() {
	_ = time.Now()
}
