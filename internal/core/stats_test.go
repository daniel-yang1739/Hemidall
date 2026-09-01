package core_test

import (
	"testing"

	"heimdall/internal/core"
)

func TestSessionMetricsAggregatePairedMeteredInputAndCacheObservations(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Tokens: core.TokenBreakdown{TotalTokens: 9_999}},
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasMeteredInputTokens: true, MeteredInputTokens: 30, HasCachedContentTokens: true, CachedContentTokens: 70}},
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-b", HasMeteredInputTokens: true, MeteredInputTokens: 40, HasCachedContentTokens: true, CachedContentTokens: 10}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 70, metrics.TotalStats.MeteredInputTokenSum)
	requireEqual(t, 80, metrics.TotalStats.CachedContentTokenSum)
	requireEqual(t, 150, metrics.TotalStats.TotalProcessedTokenSum)
	requireEqual(t, 2, metrics.TotalStats.ExplicitCacheValueTurnCount)
	requireEqual(t, 0, metrics.TotalStats.DefaultZeroCacheTurnCount)
	requireEqual(t, 2, len(metrics.ModelStats))
}

func TestSessionMetricsTreatOmittedCacheScalarAsProtoDefaultZero(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasMeteredInputTokens: true, MeteredInputTokens: 100}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 100, metrics.TotalStats.MeteredInputTokenSum)
	requireEqual(t, 100, metrics.TotalStats.TotalProcessedTokenSum)
	requireEqual(t, 1, metrics.TotalStats.TurnCount)
	requireEqual(t, 0, metrics.TotalStats.CachedContentTokenSum)
	requireEqual(t, 0, metrics.TotalStats.ExplicitCacheValueTurnCount)
	requireEqual(t, 1, metrics.TotalStats.DefaultZeroCacheTurnCount)
}

func TestSessionMetricsUseSameUsageMessageInsteadOfSeparateContextState(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasObservedContextTokens: true, ObservedContextTokens: 10, HasMeteredInputTokens: true, MeteredInputTokens: 3, HasCachedContentTokens: true, CachedContentTokens: 11}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 3, metrics.TotalStats.MeteredInputTokenSum)
	requireEqual(t, 11, metrics.TotalStats.CachedContentTokenSum)
	requireEqual(t, 14, metrics.TotalStats.TotalProcessedTokenSum)
	requireEqual(t, 1, metrics.TotalStats.ExplicitCacheValueTurnCount)
	requireEqual(t, 10, metrics.TotalStats.ObservedContextTokenSum)
}
