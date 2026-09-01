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
			Available:                true,
			Source:                   "fixture",
			GenerationIndex:          7,
			StepIndex:                9,
			ModelName:                "gemini-test",
			HasObservedContextTokens: true,
			ObservedContextTokens:    800,
			HasCachedContentTokens:   true,
			CachedContentTokens:      0,
		},
	}

	analyzer.AnalyzeStep(&event)

	requireEqual(t, 800, event.Usage.ObservedContextTokens)
	requireEqual(t, 0, event.Usage.CachedContentTokens)
	requireEqual(t, true, event.Usage.HasCachedContentTokens)
	requireGreaterThan(t, event.Tokens.StepDelta, 0)
	requireEqual(t, event.Tokens.StepDelta, event.Tokens.RawLocalAccumulated)
}

func TestAnalyzerDoesNotFabricateTokensForEmptyEvent(t *testing.T) {
	analyzer := core.NewPayloadAnalyzer()
	event := core.UnifiedAgentEvent{SessionID: analyzerTestSessionID, Type: core.StepTypeGeneric}

	analyzer.AnalyzeStep(&event)

	requireEqual(t, 0, event.Tokens.StepDelta)
	requireEqual(t, 0, event.Tokens.TotalTokens)
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

func TestPersistedUsageDoesNotDeriveCacheMathFromIndependentFields(t *testing.T) {
	observation := core.PersistedUsageObservation{
		HasObservedContextTokens: true,
		ObservedContextTokens:    10,
		HasCachedContentTokens:   true,
		CachedContentTokens:      11,
	}

	requireEqual(t, 10, observation.ObservedContextTokens)
	requireEqual(t, 11, observation.CachedContentTokens)
}
