package core_test

import (
	"testing"

	"heimdall/internal/core"
)

func TestSessionMetricsIncludeOnlyPersistedUsage(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Tokens: core.TokenBreakdown{TotalTokens: 9_999}},
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasTotalTokens: true, TotalTokens: 100, HasCachedTokens: true, CachedTokens: 70}},
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-b", HasTotalTokens: true, TotalTokens: 50, HasCachedTokens: true, CachedTokens: 10}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 150, metrics.TotalStats.TotalProcessed)
	requireEqual(t, 80, metrics.TotalStats.TotalCached)
	requireEqual(t, 70, metrics.TotalStats.TotalNew)
	requireEqual(t, true, metrics.TotalStats.CompleteUsage)
	requireEqual(t, 2, len(metrics.ModelStats))
}

func TestSessionMetricsMarkIncompleteCacheObservations(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasTotalTokens: true, TotalTokens: 100}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 100, metrics.TotalStats.TotalProcessed)
	requireEqual(t, false, metrics.TotalStats.CompleteUsage)
	requireEqual(t, 0, metrics.TotalStats.TotalCached)
}

func TestSessionMetricsKeepComparableCacheSubsetSeparate(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasTotalTokens: true, TotalTokens: 100, HasCachedTokens: true, CachedTokens: 60}},
		{Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, ModelName: "model-a", HasTotalTokens: true, TotalTokens: 50}},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	requireEqual(t, 150, metrics.TotalStats.TotalProcessed)
	requireEqual(t, 100, metrics.TotalStats.ComparableTokens)
	requireEqual(t, 1, metrics.TotalStats.CachedTurnCount)
	requireEqual(t, 60.0, metrics.TotalStats.CacheHitRate)
	requireEqual(t, false, metrics.TotalStats.CompleteUsage)
}
