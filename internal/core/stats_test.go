package core_test

import (
	"testing"

	"heimdall/internal/core"
)

func TestSessionMetricsAggregatePairedUncachedAndCachedInputObservations(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Tokens: core.TokenBreakdown{TotalTokens: 9_999}},
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasUncachedInputTokens: true, UncachedInputTokens: 30, HasCachedInputTokens: true, CachedInputTokens: 70}},
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-b", HasUncachedInputTokens: true, UncachedInputTokens: 40, HasCachedInputTokens: true, CachedInputTokens: 10}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 70, metrics.TotalStats.UncachedInputTokenSum)
	requireEqual(t, 80, metrics.TotalStats.CachedInputTokenSum)
	requireEqual(t, 150, metrics.TotalStats.TotalProcessedTokenSum)
	requireEqual(t, 2, metrics.TotalStats.ExplicitCacheValueTurnCount)
	requireEqual(t, 0, metrics.TotalStats.InferredZeroCacheTurnCount)
	requireEqual(t, 2, len(metrics.ModelStats))
}

func TestSessionMetricsTreatOmittedCacheScalarAsInferredZeroForCompleteUsage(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasUncachedInputTokens: true, UncachedInputTokens: 100}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 100, metrics.TotalStats.UncachedInputTokenSum)
	requireEqual(t, 100, metrics.TotalStats.TotalProcessedTokenSum)
	requireEqual(t, 1, metrics.TotalStats.TurnCount)
	requireEqual(t, 0, metrics.TotalStats.CachedInputTokenSum)
	requireEqual(t, 0, metrics.TotalStats.ExplicitCacheValueTurnCount)
	requireEqual(t, 1, metrics.TotalStats.InferredZeroCacheTurnCount)
}

func TestSessionMetricsSkipIncompleteUsageObservation(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasObservedContextTokens: true, ObservedContextTokens: 160_000}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 0, metrics.TotalStats.TurnCount)
	requireEqual(t, 0, metrics.TotalStats.TotalProcessedTokenSum)
	requireEqual(t, 0, len(metrics.ModelStats))
}

func TestSessionMetricsUseSameUsageMessageInsteadOfSeparateContextState(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasObservedContextTokens: true, ObservedContextTokens: 10, HasUncachedInputTokens: true, UncachedInputTokens: 3, HasCachedInputTokens: true, CachedInputTokens: 11}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 3, metrics.TotalStats.UncachedInputTokenSum)
	requireEqual(t, 11, metrics.TotalStats.CachedInputTokenSum)
	requireEqual(t, 14, metrics.TotalStats.TotalProcessedTokenSum)
	requireEqual(t, 1, metrics.TotalStats.ExplicitCacheValueTurnCount)
	requireEqual(t, 10, metrics.TotalStats.ObservedContextTokenSum)
}
