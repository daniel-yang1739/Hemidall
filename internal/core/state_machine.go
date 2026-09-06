package core

// StepLinkageTracker tracks causality links (parent/child) across agent events in a single pass
type StepLinkageTracker struct {
	lastUserPromptIdx     int
	lastParentToolCallIdx int
	pendingStagedSteps    []int
}

// NewStepLinkageTracker creates a new tracker
func NewStepLinkageTracker() *StepLinkageTracker {
	return &StepLinkageTracker{}
}

// ProcessEvent processes an incoming event, assigns Scope, and links Parent/Consumed indices
func (t *StepLinkageTracker) ProcessEvent(e *UnifiedAgentEvent) {
	if e.Type == StepTypeUserInput {
		e.Scope = ScopeUserInteraction
		t.lastUserPromptIdx = e.StepIndex
		t.lastParentToolCallIdx = 0
		t.pendingStagedSteps = nil
	} else if e.IsCompactionStep() {
		e.Scope = ScopeSystemCompaction
		t.lastParentToolCallIdx = 0
		t.pendingStagedSteps = nil
	} else if e.IsLocalStep() {
		e.Scope = ScopeLocalExecution
		e.ParentStepIdx = t.lastParentToolCallIdx
		t.pendingStagedSteps = append(t.pendingStagedSteps, e.StepIndex)
	} else if e.Type == StepTypeToolCall || (e.Type == StepTypeModelResponse && len(e.ToolCalls) > 0) {
		e.Scope = ScopeCloudInference
		e.ParentStepIdx = t.lastUserPromptIdx
		if len(t.pendingStagedSteps) > 0 {
			e.ConsumedStepIndices = make([]int, len(t.pendingStagedSteps))
			copy(e.ConsumedStepIndices, t.pendingStagedSteps)
			t.pendingStagedSteps = nil
		}
		t.lastParentToolCallIdx = e.StepIndex
	} else if e.Type == StepTypeModelResponse {
		e.Scope = ScopeCloudInference
		e.ParentStepIdx = t.lastUserPromptIdx
		if len(t.pendingStagedSteps) > 0 {
			e.ConsumedStepIndices = make([]int, len(t.pendingStagedSteps))
			copy(e.ConsumedStepIndices, t.pendingStagedSteps)
			t.pendingStagedSteps = nil
		}
		t.lastParentToolCallIdx = 0
	} else {
		e.Scope = ScopeSystemBootstrap
	}
}

// BackfillPackagedIn backfills PackagedInStepIdx for all local steps consumed by subsequent cloud turns
func BackfillPackagedIn(events []UnifiedAgentEvent) {
	consumedByMap := make(map[int]int)
	for _, e := range events {
		if len(e.ConsumedStepIndices) > 0 {
			for _, childIdx := range e.ConsumedStepIndices {
				consumedByMap[childIdx] = e.StepIndex
			}
		}
	}

	for i := range events {
		if events[i].IsLocalStep() {
			if cloudIdx, exists := consumedByMap[events[i].StepIndex]; exists {
				events[i].PackagedInStepIdx = cloudIdx
			}
		}

		// For USER_INPUT, record which cloud turn evaluated this prompt without copying its cache status
		if events[i].Type == StepTypeUserInput {
			for j := i + 1; j < len(events); j++ {
				if events[j].IsCloudStep() {
					events[i].PackagedInStepIdx = events[j].StepIndex
					break
				}
			}
		}
	}
}
