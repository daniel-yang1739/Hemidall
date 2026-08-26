package core

import (
	"math"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultCacheTTL is the standard LLM KV Cache lifetime in GPU HBM (5 minutes)
	DefaultCacheTTL = 5 * time.Minute
)

// StepTokenRecord stores token metrics for an individual step to enable reverse sliding window calculation
type StepTokenRecord struct {
	StepIndex int
	Type      StepType
	Tokens    int
}

// SessionContextState tracks cumulative context tokens and step history per session
type SessionContextState struct {
	SessionID       string
	BaseSystem      int               // Baseline static system instructions (identity, rules, skills)
	BaseToolsDef    int               // Baseline MCP Tools JSON Schema definitions
	StepRecords     []StepTokenRecord // Historical log of all steps for reverse sliding window extraction
	HasInitialized  bool              // Indicates if the initial cache write turn has completed
	PrevTotalTokens int               // Previous turn's total context tokens (LCP comparison baseline)
	LastEventTime   time.Time         // Timestamp of previous turn to detect TTL expiration
}

// PayloadAnalyzer evaluates unified events and updates context state machine using reverse sliding window
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

// AnalyzeStep processes a single event and injects 5-dimension token breakdown via reverse sliding window
func (a *PayloadAnalyzer) AnalyzeStep(event *UnifiedAgentEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	state, exists := a.sessions[event.SessionID]
	if !exists {
		state = &SessionContextState{
			SessionID:      event.SessionID,
			BaseSystem:     3806, // Baseline static system instructions (~2.2% of 175k)
			BaseToolsDef:   1377, // Baseline MCP Tools JSON Schema definitions (~0.8% of 175k)
			StepRecords:    make([]StepTokenRecord, 0, 1000),
			HasInitialized: false,
		}
		a.sessions[event.SessionID] = state
	}

	// 1. Calculate tokens for this step using local BPE tokenizer
	contentTokens := CountTokens(event.RawContent)
	thinkingTokens := CountTokens(event.Thinking)

	toolCallArgsTokens := 0
	for _, tc := range event.ToolCalls {
		if tc.RawArgs != "" {
			toolCallArgsTokens += CountTokens(tc.RawArgs)
		} else if len(tc.Arguments) > 0 {
			toolCallArgsTokens += 50
		}
	}

	stepTokens := contentTokens + thinkingTokens + toolCallArgsTokens
	if stepTokens == 0 {
		stepTokens = 20
	}

	// Record this step in state history
	state.StepRecords = append(state.StepRecords, StepTokenRecord{
		StepIndex: event.StepIndex,
		Type:      event.Type,
		Tokens:    stepTokens,
	})

	// Calculate all-time raw accumulated log tokens
	rawAllTimeTokens := state.BaseSystem + state.BaseToolsDef
	for _, rec := range state.StepRecords {
		rawAllTimeTokens += rec.Tokens
	}
	event.Tokens.RawLocalAccumulated = rawAllTimeTokens

	// 2. Dual-Track Official Telemetry Fusion with Reverse Sliding Window
	if event.Tokens.IsOfficialData && event.Tokens.TotalTokens > 0 {
		officialTotal := event.Tokens.TotalTokens

		// Fixed baseline dimensions
		d1 := state.BaseSystem
		d2 := state.BaseToolsDef
		if officialTotal <= d1+d2 {
			d1 = int(float64(officialTotal) * 0.7)
			d2 = officialTotal - d1
		}
		budget := officialTotal - (d1 + d2)
		if budget < 0 {
			budget = 0
		}

		// 3. REVERSE SLIDING WINDOW: Iterate backwards from newest step to fill budget
		accResults := 0
		accHistory := 0
		accActive := 0
		currentAccumulated := 0

		for i := len(state.StepRecords) - 1; i >= 0; i-- {
			rec := state.StepRecords[i]
			needed := rec.Tokens
			isLastStep := (i == len(state.StepRecords)-1)

			if currentAccumulated+needed > budget {
				needed = budget - currentAccumulated
			}

			if isLastStep {
				accActive += needed
			} else {
				switch rec.Type {
				case StepTypeRunCommand, StepTypeViewFile, StepTypeCodeAction, StepTypeListDirectory, StepTypeToolResult:
					accResults += needed
				case StepTypeUserInput, StepTypeModelResponse, StepTypeToolCall:
					accHistory += needed
				default:
					if strings.Contains(string(rec.Type), "SYSTEM") {
						d1 += needed
					} else {
						accHistory += needed
					}
				}
			}

			currentAccumulated += needed
			if currentAccumulated >= budget {
				break
			}
		}

		// Proportional allocation of sliding window budget
		accWindowTotal := accResults + accHistory + accActive
		var d3, d4, d5 int
		if accWindowTotal > 0 && budget > 0 {
			d3 = int(math.Round(float64(accResults) / float64(accWindowTotal) * float64(budget)))
			d4 = int(math.Round(float64(accHistory) / float64(accWindowTotal) * float64(budget)))
			d5 = budget - (d3 + d4)
			if d5 < 0 {
				d5 = 0
			}
		}

		event.Tokens.SystemTokens = d1
		event.Tokens.ToolsDefTokens = d2
		event.Tokens.ToolResultTokens = d3
		event.Tokens.HistoryTokens = d4
		event.Tokens.ActiveTurnTokens = d5
		event.Tokens.ThinkingTokens = 0

		// Ensure NewTokens = Total - Cached
		event.Tokens.NewTokens = officialTotal - event.Tokens.CachedTokens
		if event.Tokens.NewTokens < 0 {
			event.Tokens.NewTokens = 0
		}
		if officialTotal > 0 {
			event.Tokens.CacheHitRate = float64(event.Tokens.CachedTokens) / float64(officialTotal) * 100.0
		}
		event.CacheStatus = determineCacheStatus(event.Tokens.CachedTokens, officialTotal, event.Tokens.CacheHitRate)
		state.PrevTotalTokens = officialTotal
	} else {
		// ==================== FALLBACK: INCREMENTAL SLIDING WINDOW ====================
		event.Tokens.SystemTokens = state.BaseSystem
		event.Tokens.ToolsDefTokens = state.BaseToolsDef

		var currentStepTokens int = stepTokens
		switch event.Type {
		case StepTypeUserInput, StepTypeModelResponse, StepTypeToolCall:
			event.Tokens.ActiveTurnTokens = currentStepTokens
		case StepTypeRunCommand, StepTypeViewFile, StepTypeCodeAction, StepTypeListDirectory, StepTypeToolResult:
			event.Tokens.ToolResultTokens = currentStepTokens
		default:
			event.Tokens.ActiveTurnTokens = currentStepTokens
		}

		isTTLExpired := false
		if !state.LastEventTime.IsZero() && !event.Timestamp.IsZero() {
			if event.Timestamp.Sub(state.LastEventTime) > DefaultCacheTTL {
				isTTLExpired = true
			}
		}

		if isTTLExpired && state.PrevTotalTokens > 0 {
			// Physical TTL Expired: Context is still ~165k, but GPU cache is 0! All 165k tokens are New billable!
			totalTokens := state.PrevTotalTokens + stepTokens
			event.Tokens.TotalTokens = totalTokens
			event.Tokens.CachedTokens = 0
			event.Tokens.NewTokens = totalTokens
			event.Tokens.CacheHitRate = 0.0
			event.CacheStatus = "EXPIRED"
			state.PrevTotalTokens = totalTokens

			hist := totalTokens - (state.BaseSystem + state.BaseToolsDef + currentStepTokens)
			if hist < 0 {
				hist = 0
			}
			event.Tokens.HistoryTokens = hist
		} else if !state.HasInitialized || state.PrevTotalTokens == 0 {
			// First-ever Cold Start turn
			event.Tokens.TotalTokens = state.BaseSystem + state.BaseToolsDef + stepTokens
			event.Tokens.CachedTokens = 0
			event.Tokens.NewTokens = event.Tokens.TotalTokens
			event.Tokens.CacheHitRate = 0.0
			event.CacheStatus = "WRITE"
			event.Tokens.HistoryTokens = 0
			state.HasInitialized = true
			state.PrevTotalTokens = event.Tokens.TotalTokens
		} else {
			// Hot Cache Incremental Turn
			maxContextLimit := event.Tokens.OfficialContextLimit
			if maxContextLimit == 0 {
				maxContextLimit = 256000
			}

			cachedTokens := state.PrevTotalTokens
			newTokens := stepTokens
			totalTokens := cachedTokens + newTokens

			if totalTokens > maxContextLimit {
				totalTokens = maxContextLimit
				cachedTokens = totalTokens - newTokens
				if cachedTokens < 0 {
					cachedTokens = 0
				}
			}

			hitRate := 0.0
			if totalTokens > 0 {
				hitRate = float64(cachedTokens) / float64(totalTokens) * 100.0
			}

			event.Tokens.TotalTokens = totalTokens
			event.Tokens.CachedTokens = cachedTokens
			event.Tokens.NewTokens = newTokens
			event.Tokens.CacheHitRate = hitRate
			event.CacheStatus = determineCacheStatus(cachedTokens, totalTokens, hitRate)
			// Do NOT overwrite state.PrevTotalTokens on intermediate steps so they don't compound

			hist := totalTokens - (state.BaseSystem + state.BaseToolsDef + currentStepTokens)
			if hist < 0 {
				hist = 0
			}
			event.Tokens.HistoryTokens = hist
		}
	}

	if !event.Timestamp.IsZero() {
		state.LastEventTime = event.Timestamp
	}
}

func determineCacheStatus(cached, total int, hitRate float64) string {
	if cached == 0 {
		return "MISS"
	}
	if hitRate >= 80.0 {
		return "HIT"
	} else if hitRate > 0.0 {
		return "PARTIAL"
	}
	return "MISS"
}
