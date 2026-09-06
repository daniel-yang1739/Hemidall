package antigravity

import (
	"sort"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/core"
)

// buildSession is the only place where Antigravity parsed facts become the
// source-neutral Session read model. Cross-source joins will be added here as
// additional parsers become available; no consumer should perform those joins.
func buildSession(ref agents.SessionRef, stepsByIndex map[int]agents.Step, generations []agents.Generation, contextSnapshots []agents.ContextSnapshot, revision uint64) agents.Session {
	steps := make([]agents.Step, 0, len(stepsByIndex))
	for _, step := range stepsByIndex {
		steps = append(steps, cloneStep(step))
	}
	sort.SliceStable(steps, func(left, right int) bool {
		return steps[left].Index < steps[right].Index
	})
	assembleStepLifecycle(steps)
	modelsByID := make(map[string]*agents.ModelRecord)
	for _, generation := range generations {
		if generation.ModelID == "" {
			continue
		}
		model := modelsByID[generation.ModelID]
		if model == nil {
			model = &agents.ModelRecord{ModelID: generation.ModelID, Evidence: append([]agents.Evidence(nil), generation.Evidence...)}
			modelsByID[generation.ModelID] = model
		}
		model.GenerationIDs = append(model.GenerationIDs, generation.ID)
	}
	modelRecords := make([]agents.ModelRecord, 0, len(modelsByID))
	for _, model := range modelsByID {
		modelRecords = append(modelRecords, *model)
	}
	sort.Slice(modelRecords, func(left, right int) bool {
		return modelRecords[left].ModelID < modelRecords[right].ModelID
	})
	return agents.Session{
		Ref:              ref,
		Steps:            steps,
		Generations:      append([]agents.Generation(nil), generations...),
		ContextSnapshots: append([]agents.ContextSnapshot(nil), contextSnapshots...),
		Models:           agents.ModelCatalog{Records: modelRecords},
		Revision:         revision,
	}
}

func cloneSession(session agents.Session) agents.Session {
	cloned := session
	cloned.Steps = make([]agents.Step, len(session.Steps))
	for index, step := range session.Steps {
		cloned.Steps[index] = cloneStep(step)
	}
	cloned.Generations = append([]agents.Generation(nil), session.Generations...)
	cloned.ContextSnapshots = append([]agents.ContextSnapshot(nil), session.ContextSnapshots...)
	cloned.Models.Records = append([]agents.ModelRecord(nil), session.Models.Records...)
	for index := range cloned.Models.Records {
		cloned.Models.Records[index].GenerationIDs = append([]string(nil), session.Models.Records[index].GenerationIDs...)
		cloned.Models.Records[index].Evidence = append([]agents.Evidence(nil), session.Models.Records[index].Evidence...)
	}
	return cloned
}

func cloneStep(step agents.Step) agents.Step {
	cloned := step
	cloned.Evidence = append([]agents.Evidence(nil), step.Evidence...)
	cloned.ToolCalls = make([]agents.ToolCall, len(step.ToolCalls))
	for index, toolCall := range step.ToolCalls {
		cloned.ToolCalls[index] = agents.ToolCall{
			Name: toolCall.Name,
			Args: cloneArguments(toolCall.Args),
		}
	}
	return cloned
}

func cloneArguments(arguments map[string]any) map[string]any {
	if arguments == nil {
		return nil
	}
	cloned := make(map[string]any, len(arguments))
	for key, value := range arguments {
		cloned[key] = value
	}
	return cloned
}

func assembleStepLifecycle(steps []agents.Step) {
	events := make([]core.UnifiedAgentEvent, len(steps))
	for index, step := range steps {
		toolCalls := make([]core.ToolCallInfo, len(step.ToolCalls))
		for tcIdx, tc := range step.ToolCalls {
			toolCalls[tcIdx] = core.ToolCallInfo{ToolName: tc.Name, Arguments: tc.Args}
		}
		events[index] = core.UnifiedAgentEvent{
			StepIndex: step.Index,
			Type:      core.StepType(step.Kind),
			Source:    step.Source,
			ToolCalls: toolCalls,
		}
	}
	tracker := core.NewStepLinkageTracker()
	for index := range events {
		tracker.ProcessEvent(&events[index])
	}
	core.BackfillPackagedIn(events)
	for index := range steps {
		steps[index].Scope = events[index].Scope
		steps[index].AgentRole = events[index].GetAgentRole()
		steps[index].ParentStepIndex = events[index].ParentStepIdx
		steps[index].PackagedInStepIndex = events[index].PackagedInStepIdx
		steps[index].ConsumedStepIndexes = append([]int(nil), events[index].ConsumedStepIndices...)
	}
}
