package core_test

import (
	"testing"

	"heimdall/internal/core"
)

const analyzerTestSessionID = "analyzer-test-session"

func TestAnalyzerPreservesPersistedUsageAndBuildsLocalEstimate(t *testing.T) {
	analyzer := core.NewPayloadAnalyzer()
	event := core.UnifiedAgentEvent{
		SessionID:  analyzerTestSessionID,
		Type:       core.StepTypeModelResponse,
		RawContent: "A persisted observation must remain unchanged.",
		Usage: core.PersistedUsageObservation{
			Available:       true,
			Source:          "fixture",
			GenerationIndex: 7,
			StepIndex:       9,
			ModelName:       "gemini-test",
			HasTotalTokens:  true,
			TotalTokens:     800,
			HasCachedTokens: true,
			CachedTokens:    0,
		},
	}

	analyzer.AnalyzeStep(&event)

	requireEqual(t, 800, event.Usage.TotalTokens)
	requireEqual(t, 0, event.Usage.CachedTokens)
	requireEqual(t, true, event.Usage.HasCachedTokens)
	requireEqual(t, "MISS", event.CacheStatus)
	requireGreaterThan(t, event.Tokens.StepDelta, 0)
	requireEqual(t, event.Tokens.StepDelta, event.Tokens.RawLocalAccumulated)
}

func TestAnalyzerDoesNotFabricateTokensForEmptyEvent(t *testing.T) {
	analyzer := core.NewPayloadAnalyzer()
	event := core.UnifiedAgentEvent{SessionID: analyzerTestSessionID, Type: core.StepTypeGeneric}

	analyzer.AnalyzeStep(&event)

	requireEqual(t, 0, event.Tokens.StepDelta)
	requireEqual(t, 0, event.Tokens.TotalTokens)
	requireEqual(t, "UNKNOWN", event.CacheStatus)
}

func TestAnalyzerEstimatesObservedToolArgumentsWithoutFallbackLiteral(t *testing.T) {
	analyzer := core.NewPayloadAnalyzer()
	event := core.UnifiedAgentEvent{
		SessionID: analyzerTestSessionID,
		Type:      core.StepTypeToolCall,
		ToolCalls: []core.ToolCallInfo{{
			ToolName:  "view_file",
			Arguments: map[string]interface{}{"path": "README.md"},
		}},
	}

	analyzer.AnalyzeStep(&event)

	requireGreaterThan(t, event.Tokens.StepDelta, 0)
	requireEqual(t, event.Tokens.StepDelta, event.Tokens.ActiveTurnTokens)
}

func TestPersistedUsageDerivationsRequireBothDecodedFields(t *testing.T) {
	complete := core.PersistedUsageObservation{HasTotalTokens: true, TotalTokens: 100, HasCachedTokens: true, CachedTokens: 70}
	uncached, hasUncached := complete.UncachedTokens()
	hitRate, hasHitRate := complete.CacheHitRate()
	partial := core.PersistedUsageObservation{HasTotalTokens: true, TotalTokens: 100}
	_, partialHasUncached := partial.UncachedTokens()
	_, partialHasHitRate := partial.CacheHitRate()

	requireEqual(t, 30, uncached)
	requireEqual(t, true, hasUncached)
	requireEqual(t, 70.0, hitRate)
	requireEqual(t, true, hasHitRate)
	requireEqual(t, false, partialHasUncached)
	requireEqual(t, false, partialHasHitRate)
}
