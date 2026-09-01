package core

import (
	"encoding/json"
	"sync"
)

// SessionContextState tracks only local estimate state. Persisted generation
// metadata is attached by an adapter before analysis and is never mutated here.
type SessionContextState struct {
	SessionID           string
	RawLocalAccumulated int
}

// PayloadAnalyzer calculates a clearly labelled local cl100k_base estimate for
// transcript content. It deliberately does not infer vendor cache state, TTL,
// context windows, model families, or pricing.
type PayloadAnalyzer struct {
	mu       sync.Mutex
	sessions map[string]*SessionContextState
}

// NewPayloadAnalyzer creates an analyzer for local transcript content only.
func NewPayloadAnalyzer() *PayloadAnalyzer {
	return &PayloadAnalyzer{sessions: make(map[string]*SessionContextState)}
}

// AnalyzeStep injects a local content estimate and preserves any persisted
// generation observation already attached to event. Its work is O(1) per event.
func (a *PayloadAnalyzer) AnalyzeStep(event *UnifiedAgentEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	state := a.sessions[event.SessionID]
	if state == nil {
		state = &SessionContextState{SessionID: event.SessionID}
		a.sessions[event.SessionID] = state
	}

	contentTokens := CountTokens(event.RawContent)
	thinkingTokens := CountTokens(event.Thinking)
	argumentTokens := estimateToolArguments(event.ToolCalls)
	stepTokens := contentTokens + thinkingTokens + argumentTokens
	state.RawLocalAccumulated += stepTokens

	estimate := TokenBreakdown{
		HistoryTokens:       state.RawLocalAccumulated - stepTokens,
		ThinkingTokens:      thinkingTokens,
		TotalTokens:         state.RawLocalAccumulated,
		RawLocalAccumulated: state.RawLocalAccumulated,
		StepDelta:           stepTokens,
	}
	if event.IsLocalStep() && event.Type != StepTypeUserInput {
		estimate.ToolResultTokens = stepTokens
	} else {
		estimate.ActiveTurnTokens = contentTokens + argumentTokens
	}
	event.Tokens = estimate

	assignScope(event)
}

func estimateToolArguments(calls []ToolCallInfo) int {
	total := 0
	for _, call := range calls {
		if call.RawArgs != "" {
			total += CountTokens(call.RawArgs)
			continue
		}
		if len(call.Arguments) == 0 {
			continue
		}
		encoded, err := json.Marshal(call.Arguments)
		if err == nil {
			total += CountTokens(string(encoded))
		}
	}
	return total
}

func assignScope(event *UnifiedAgentEvent) {
	switch {
	case event.IsCompactionStep():
		event.Scope = ScopeSystemCompaction
	case event.Type == StepTypeUserInput:
		event.Scope = ScopeUserInteraction
	case event.IsCloudStep():
		event.Scope = ScopeCloudInference
	case event.Type == StepTypeSystemInit:
		event.Scope = ScopeSystemBootstrap
	default:
		event.Scope = ScopeLocalExecution
	}
}
