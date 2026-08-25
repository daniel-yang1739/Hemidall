package core

import (
	"strings"
	"sync"
)

// SessionContextState tracks cumulative context tokens and prefix caching benchmarks per session
type SessionContextState struct {
	SessionID        string
	SystemTokens     int
	ToolsDefTokens   int
	ToolResultTokens int
	HistoryTokens    int
	TotalTokens      int
	HasInitialized   bool // Indicates if the initial cache write turn has completed
	PrevTotalTokens  int  // Previous turn's total context tokens (LCP comparison baseline)
}

// PayloadAnalyzer evaluates unified events and updates context state machine
type PayloadAnalyzer struct {
	mu       sync.Mutex
	sessions map[string]*SessionContextState
}

// NewPayloadAnalyzer creates a new analyzer instance
func NewPayloadAnalyzer() *PayloadAnalyzer {
	return &PayloadAnalyzer{
		sessions: make(map[string]*SessionContextState),
	}
}

// AnalyzeStep processes a single event and injects 5-dimension token breakdown and cache metrics
func (a *PayloadAnalyzer) AnalyzeStep(event *UnifiedAgentEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	state, exists := a.sessions[event.SessionID]
	if !exists {
		state = &SessionContextState{
			SessionID:      event.SessionID,
			SystemTokens:   4618, // Baseline static system instructions (identity, rules, skills)
			ToolsDefTokens: 3200, // Baseline MCP Tools JSON Schema definitions
			HasInitialized: false,
		}
		a.sessions[event.SessionID] = state
	}

	// 1. Calculate tokens for this step
	contentTokens := CountTokens(event.RawContent)
	thinkingTokens := CountTokens(event.Thinking)

	// Calculate tool call arguments tokens
	toolCallArgsTokens := 0
	for _, tc := range event.ToolCalls {
		if tc.RawArgs != "" {
			toolCallArgsTokens += CountTokens(tc.RawArgs)
		} else if len(tc.Arguments) > 0 {
			toolCallArgsTokens += 50
		}
	}

	// 2. Accumulate tokens based on StepType
	switch event.Type {
	case StepTypeUserInput:
		event.Tokens.ActiveTurnTokens = contentTokens
	case StepTypeModelResponse:
		event.Tokens.ActiveTurnTokens = contentTokens
		event.Tokens.ThinkingTokens = thinkingTokens
	case StepTypeToolCall:
		event.Tokens.ActiveTurnTokens = toolCallArgsTokens
		event.Tokens.ThinkingTokens = thinkingTokens
	case StepTypeRunCommand, StepTypeViewFile, StepTypeCodeAction, StepTypeListDirectory, StepTypeToolResult:
		state.ToolResultTokens += contentTokens
		event.Tokens.ToolResultTokens = contentTokens
	default:
		if strings.Contains(string(event.Type), "SYSTEM") {
			state.SystemTokens += contentTokens
		} else if contentTokens > 0 {
			state.HistoryTokens += contentTokens
		}
	}

	// 3. Compute total context volume
	currentDelta := event.Tokens.ActiveTurnTokens + event.Tokens.ThinkingTokens
	totalContext := state.SystemTokens + state.ToolsDefTokens + state.ToolResultTokens + state.HistoryTokens + currentDelta

	event.Tokens.SystemTokens = state.SystemTokens
	event.Tokens.ToolsDefTokens = state.ToolsDefTokens
	event.Tokens.ToolResultTokens = state.ToolResultTokens
	event.Tokens.HistoryTokens = state.HistoryTokens
	event.Tokens.TotalTokens = totalContext

	// 4. Prefix Caching LCP (Longest Common Prefix) calculation
	if !state.HasInitialized {
		// Initial turn: Cache Write
		event.Tokens.CachedTokens = 0
		event.Tokens.NewTokens = totalContext
		event.Tokens.CacheHitRate = 0.0
		event.CacheStatus = "WRITE"
		state.HasInitialized = true
	} else {
		// Subsequent turns: reuse static prefix + solidified history
		cachedTokens := state.PrevTotalTokens
		if cachedTokens > totalContext {
			cachedTokens = totalContext
		}

		newTokens := totalContext - cachedTokens
		if newTokens < 0 {
			newTokens = 0
		}

		hitRate := 0.0
		if totalContext > 0 {
			hitRate = float64(cachedTokens) / float64(totalContext) * 100.0
		}

		event.Tokens.CachedTokens = cachedTokens
		event.Tokens.NewTokens = newTokens
		event.Tokens.CacheHitRate = hitRate

		if hitRate >= 80.0 {
			event.CacheStatus = "HIT"
		} else if hitRate > 0.0 {
			event.CacheStatus = "PARTIAL"
		} else {
			event.CacheStatus = "MISS"
		}
	}

	// 5. Update state baseline
	state.PrevTotalTokens = totalContext
	if event.Type == StepTypeUserInput || event.Type == StepTypeModelResponse {
		state.HistoryTokens += contentTokens
	}
}
