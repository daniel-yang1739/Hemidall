package antigravity

import (
	"testing"

	"heimdall/internal/core"
)

func TestPersistedUsageSurvivesLocalAnalysis(t *testing.T) {
	analyzer := core.NewPayloadAnalyzer()
	event := core.UnifiedAgentEvent{
		SessionID:  "calibration-session",
		StepIndex:  100,
		Type:       core.StepTypeModelResponse,
		RawContent: "Hello from assistant",
		Usage: core.PersistedUsageObservation{
			Available:                true,
			Source:                   "fixture",
			GenerationIndex:          22,
			StepIndex:                100,
			ModelName:                "gemini-3.7-flash-high",
			HasObservedContextTokens: true,
			ObservedContextTokens:    159_043,
			HasMeteredInputTokens:    true,
			MeteredInputTokens:       14_812,
			HasCachedContentTokens:   true,
			CachedContentTokens:      138_240,
		},
	}

	analyzer.AnalyzeStep(&event)
	requireAntigravityEqual(t, 159_043, event.Usage.ObservedContextTokens)
	requireAntigravityEqual(t, 14_812, event.Usage.MeteredInputTokens)
	requireAntigravityEqual(t, 138_240, event.Usage.CachedContentTokens)
	requireAntigravityGreater(t, event.Tokens.TotalTokens, 0)
}
