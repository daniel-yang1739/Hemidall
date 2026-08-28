package core

import (
	"reflect"
	"testing"
)

// ==============================================================================
// 3. CAUSALITY LINKAGE & STATE MACHINE (3 Positive + 3 Negative)
// ==============================================================================

func TestLinkage_Pos_SingleToolCallToResult(t *testing.T) {
	tracker := NewStepLinkageTracker()

	events := []UnifiedAgentEvent{
		{StepIndex: 1, Type: StepTypeUserInput},
		{StepIndex: 2, Type: StepTypeToolCall},
		{StepIndex: 3, Type: StepTypeRunCommand},
		{StepIndex: 4, Type: StepTypeModelResponse},
	}

	for i := range events {
		tracker.ProcessEvent(&events[i])
	}
	BackfillPackagedIn(events)

	if events[2].ParentStepIdx != 2 {
		t.Errorf("Step 3 RunCommand parent should be 2 (ToolCall), got %d", events[2].ParentStepIdx)
	}
	if events[2].PackagedInStepIdx != 4 {
		t.Errorf("Step 3 RunCommand packaged in should be 4 (ModelResponse), got %d", events[2].PackagedInStepIdx)
	}
}

func TestLinkage_Pos_ParallelToolCallsParent(t *testing.T) {
	tracker := NewStepLinkageTracker()

	events := []UnifiedAgentEvent{
		{StepIndex: 1, Type: StepTypeUserInput},
		{StepIndex: 2, Type: StepTypeToolCall},    // Spawns 2 parallel tools
		{StepIndex: 3, Type: StepTypeViewFile},     // Child 1
		{StepIndex: 4, Type: StepTypeViewFile},     // Child 2
		{StepIndex: 5, Type: StepTypeToolCall},     // Next Cloud Turn (consumes 3, 4, spawns tool 6)
		{StepIndex: 6, Type: StepTypeRunCommand},   // Child 3
		{StepIndex: 7, Type: StepTypeModelResponse},// Final Model Response (consumes 6)
	}

	for i := range events {
		tracker.ProcessEvent(&events[i])
	}
	BackfillPackagedIn(events)

	// Verify Scopes
	if events[0].Scope != ScopeUserInteraction {
		t.Errorf("Step 1: expected ScopeUserInteraction, got %v", events[0].Scope)
	}
	if events[1].Scope != ScopeCloudInference {
		t.Errorf("Step 2: expected ScopeCloudInference, got %v", events[1].Scope)
	}
	if events[2].Scope != ScopeLocalExecution || events[3].Scope != ScopeLocalExecution {
		t.Errorf("Steps 3,4: expected ScopeLocalExecution, got %v, %v", events[2].Scope, events[3].Scope)
	}

	// Verify ParentStepIdx
	if events[1].ParentStepIdx != 1 {
		t.Errorf("Step 2 Parent: expected 1 (User Input), got %d", events[1].ParentStepIdx)
	}
	if events[2].ParentStepIdx != 2 || events[3].ParentStepIdx != 2 {
		t.Errorf("Steps 3,4 Parent: expected 2 (Tool Call), got %d, %d", events[2].ParentStepIdx, events[3].ParentStepIdx)
	}
	if events[5].ParentStepIdx != 5 {
		t.Errorf("Step 6 Parent: expected 5 (Tool Call), got %d", events[5].ParentStepIdx)
	}

	// Verify ConsumedStepIndices on Cloud Turns
	if !reflect.DeepEqual(events[4].ConsumedStepIndices, []int{3, 4}) {
		t.Errorf("Step 5 Consumed: expected [3, 4], got %v", events[4].ConsumedStepIndices)
	}
	if !reflect.DeepEqual(events[6].ConsumedStepIndices, []int{6}) {
		t.Errorf("Step 7 Consumed: expected [6], got %v", events[6].ConsumedStepIndices)
	}

	// Verify Backfilled PackagedInStepIdx on Local Steps
	if events[2].PackagedInStepIdx != 5 || events[3].PackagedInStepIdx != 5 {
		t.Errorf("Steps 3,4 PackagedIn: expected 5, got %d, %d", events[2].PackagedInStepIdx, events[3].PackagedInStepIdx)
	}
	if events[5].PackagedInStepIdx != 7 {
		t.Errorf("Step 6 PackagedIn: expected 7, got %d", events[5].PackagedInStepIdx)
	}
}

func TestLinkage_Pos_ConsumedStepIndicesAggregation(t *testing.T) {
	tracker := NewStepLinkageTracker()

	events := []UnifiedAgentEvent{
		{StepIndex: 1, Type: StepTypeToolCall},
		{StepIndex: 2, Type: StepTypeViewFile},
		{StepIndex: 3, Type: StepTypeCodeAction},
		{StepIndex: 4, Type: StepTypeRunCommand},
		{StepIndex: 5, Type: StepTypeModelResponse},
	}

	for i := range events {
		tracker.ProcessEvent(&events[i])
	}
	BackfillPackagedIn(events)

	expectedConsumed := []int{2, 3, 4}
	if !reflect.DeepEqual(events[4].ConsumedStepIndices, expectedConsumed) {
		t.Fatalf("Expected ConsumedStepIndices %v, got %v", expectedConsumed, events[4].ConsumedStepIndices)
	}
}

func TestLinkage_Neg_OrphanToolStepFallback(t *testing.T) {
	tracker := NewStepLinkageTracker()

	// Tool step arrives without any prior ToolCall in tracker
	orphanEvent := UnifiedAgentEvent{
		StepIndex: 5,
		Type:      StepTypeRunCommand,
	}
	tracker.ProcessEvent(&orphanEvent)

	// Should not crash and should safely assign default parent
	if orphanEvent.ParentStepIdx < 0 {
		t.Errorf("ParentStepIdx should not be negative for orphan step, got %d", orphanEvent.ParentStepIdx)
	}
}

func TestLinkage_Neg_SubagentIndependentParentLinkage(t *testing.T) {
	tracker := NewStepLinkageTracker()

	events := []UnifiedAgentEvent{
		{StepIndex: 1, Type: StepTypeUserInput},
		{StepIndex: 2, Type: StepTypeToolCall}, // Spawns subagent
		{StepIndex: 3, Type: StepTypeModelResponse, IsSubagent: true, AgentRole: "SUBAGENT", Scope: ScopeSubagent},
	}

	for i := range events {
		tracker.ProcessEvent(&events[i])
	}

	if events[2].GetAgentRole() != "SUBAGENT" {
		t.Errorf("Expected GetAgentRole() 'SUBAGENT', got '%s'", events[2].GetAgentRole())
	}
}

func TestLinkage_Neg_UnpackagedLocalStepGracefulZero(t *testing.T) {
	events := []UnifiedAgentEvent{
		{StepIndex: 1, Type: StepTypeUserInput},
		{StepIndex: 2, Type: StepTypeToolCall},
		{StepIndex: 3, Type: StepTypeViewFile}, // Still in execution, not yet packaged
	}

	tracker := NewStepLinkageTracker()
	for i := range events {
		tracker.ProcessEvent(&events[i])
	}
	BackfillPackagedIn(events)

	// Step 3 was not yet consumed by a downstream cloud turn, PackagedInStepIdx should be 0
	if events[2].PackagedInStepIdx != 0 {
		t.Errorf("Unpackaged step PackagedInStepIdx should be 0, got %d", events[2].PackagedInStepIdx)
	}
}

