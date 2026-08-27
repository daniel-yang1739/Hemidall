package core

import (
	"reflect"
	"testing"
)

func TestStepLinkageTracker_ParallelToolsAndTurns(t *testing.T) {
	tracker := NewStepLinkageTracker()

	events := []UnifiedAgentEvent{
		{StepIndex: 1, Type: StepTypeUserInput},
		{StepIndex: 2, Type: StepTypeToolCall}, // Spawns 2 parallel tools
		{StepIndex: 3, Type: StepTypeViewFile},  // Child 1
		{StepIndex: 4, Type: StepTypeViewFile},  // Child 2
		{StepIndex: 5, Type: StepTypeToolCall}, // Next Cloud Turn (consumes 3, 4, spawns tool 6)
		{StepIndex: 6, Type: StepTypeRunCommand}, // Child 3
		{StepIndex: 7, Type: StepTypeModelResponse}, // Final Model Response (consumes 6)
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
