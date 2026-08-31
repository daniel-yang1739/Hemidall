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
			Available:       true,
			Source:          "fixture",
			GenerationIndex: 22,
			StepIndex:       100,
			ModelName:       "gemini-3.7-flash-high",
			HasTotalTokens:  true,
			TotalTokens:     159_043,
			HasCachedTokens: true,
			CachedTokens:    138_240,
		},
	}

	analyzer.AnalyzeStep(&event)
	uncached, hasUncached := event.Usage.UncachedTokens()

	requireAntigravityEqual(t, 159_043, event.Usage.TotalTokens)
	requireAntigravityEqual(t, 138_240, event.Usage.CachedTokens)
	requireAntigravityEqual(t, 20_803, uncached)
	requireAntigravityEqual(t, true, hasUncached)
	requireAntigravityGreater(t, event.Tokens.TotalTokens, 0)
}
