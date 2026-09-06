package core

import (
	"sort"
	"strconv"
)

const unknownStepKind = "UNKNOWN"

// ProjectSessionEvents converts the source-neutral session read model into the
// compatibility event model currently rendered by the TUI. The projection is
// deterministic and never accesses adapter state.
func ProjectSessionEvents(session Session) []UnifiedAgentEvent {
	usageByStep := make(map[int]Generation, len(session.Generations))
	for _, generation := range session.Generations {
		known, exists := usageByStep[generation.StepIndex]
		if !exists || generation.ID > known.ID {
			usageByStep[generation.StepIndex] = generation
		}
	}
	events := make([]UnifiedAgentEvent, 0, len(session.Steps))
	for _, step := range session.Steps {
		events = append(events, projectSingleStepEvent(session, step, usageByStep))
	}
	sort.Slice(events, func(left, right int) bool {
		return events[left].StepIndex < events[right].StepIndex
	})
	return events
}

// ProjectChangedSessionEvents converts only the specified changed step indexes
// into unified agent events without re-projecting the entire session history.
func ProjectChangedSessionEvents(session Session, changedStepIndexes []int) []UnifiedAgentEvent {
	if len(changedStepIndexes) == 0 {
		return nil
	}
	targetMap := make(map[int]struct{}, len(changedStepIndexes))
	for _, idx := range changedStepIndexes {
		targetMap[idx] = struct{}{}
	}

	usageByStep := make(map[int]Generation)
	for _, generation := range session.Generations {
		if _, needed := targetMap[generation.StepIndex]; needed {
			known, exists := usageByStep[generation.StepIndex]
			if !exists || generation.ID > known.ID {
				usageByStep[generation.StepIndex] = generation
			}
		}
	}

	events := make([]UnifiedAgentEvent, 0, len(changedStepIndexes))
	for _, step := range session.Steps {
		if _, needed := targetMap[step.Index]; needed {
			events = append(events, projectSingleStepEvent(session, step, usageByStep))
		}
	}
	sort.Slice(events, func(left, right int) bool {
		return events[left].StepIndex < events[right].StepIndex
	})
	return events
}

func projectSingleStepEvent(session Session, step Step, usageByStep map[int]Generation) UnifiedAgentEvent {
	event := UnifiedAgentEvent{
		SessionID:           session.Ref.SessionID,
		StepIndex:           step.Index,
		Timestamp:           step.Timestamp,
		Source:              step.Source,
		Type:                StepType(step.Kind),
		Status:              step.Status,
		Summary:             step.Content,
		RawContent:          step.Content,
		Thinking:            step.Thinking,
		ToolCalls:           projectToolCalls(step.ToolCalls),
		Scope:               step.Scope,
		AgentRole:           step.AgentRole,
		ParentStepIdx:       step.ParentStepIndex,
		PackagedInStepIdx:   step.PackagedInStepIndex,
		ConsumedStepIndices: append([]int(nil), step.ConsumedStepIndexes...),
		Usage:               projectUsageObservation(usageByStep[step.Index]),
	}
	if event.Type == "" {
		event.Type = StepType(unknownStepKind)
	}
	if event.Scope == "" {
		assignScope(&event)
	}
	return event
}

func projectToolCalls(calls []ToolCall) []ToolCallInfo {
	projected := make([]ToolCallInfo, len(calls))
	for index, call := range calls {
		projected[index] = ToolCallInfo{ToolName: call.Name, Arguments: call.Args}
	}
	return projected
}

func projectUsageObservation(generation Generation) PersistedUsageObservation {
	usage := generation.Usage
	return PersistedUsageObservation{
		Available:                usage.HasObservedContextTokens || usage.HasUncachedInputTokens || usage.HasCachedInputTokens,
		HasObservedContextTokens: usage.HasObservedContextTokens,
		ObservedContextTokens:    usage.ObservedContextTokens,
		HasUncachedInputTokens:   usage.HasUncachedInputTokens,
		UncachedInputTokens:      usage.UncachedInputTokens,
		HasCachedInputTokens:     usage.HasCachedInputTokens,
		CachedInputTokens:        usage.CachedInputTokens,
		HasContextLimit:          usage.HasContextLimit,
		ContextLimit:             usage.ContextLimit,
		ThinkingOutputTokens:     usage.ThinkingOutputTokens,
		OutputContentTokens:      usage.OutputContentTokens,
		TotalTokens:              usage.TotalTokens,
		TimeToFirstTokenMs:       usage.TimeToFirstTokenMs,
		StreamingDurationMs:      usage.StreamingDurationMs,
		UpstreamRequestID:        usage.UpstreamRequestID,
		Provider:                 ProviderVertexAI,
		ModelName:                generation.ModelID,
		Model:                    NormalizeModelID(generation.ModelID),
		GenerationIndex:          generationIndex(generation.ID),
		StepIndex:                generation.StepIndex,
	}
}

func generationIndex(generationID string) int {
	index, err := strconv.Atoi(generationID)
	if err != nil {
		return 0
	}
	return index
}
