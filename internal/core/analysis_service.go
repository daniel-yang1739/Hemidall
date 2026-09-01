package core

import "sort"

const percentageMultiplier = 100.0

// AnalysisService derives aggregate metrics exclusively through SessionQuery.
// It cannot access an adapter, filesystem, database, or parser implementation.
type AnalysisService struct {
	query SessionQuery
}

// NewAnalysisService creates a query-backed analysis boundary.
func NewAnalysisService(query SessionQuery) *AnalysisService {
	return &AnalysisService{query: query}
}

// SessionMetrics calculates observed usage aggregates for the active session.
func (service *AnalysisService) SessionMetrics() SessionAggregateMetrics {
	statsByModel := make(map[string]*ModelTokenStats)
	for _, generation := range service.query.Session().Generations {
		if !generation.Usage.HasUncachedInputTokens {
			continue
		}
		modelName := generation.ModelID
		if modelName == "" {
			modelName = "unknown"
		}
		stats := statsByModel[modelName]
		if stats == nil {
			stats = &ModelTokenStats{ModelName: modelName}
			statsByModel[modelName] = stats
		}
		applyGenerationUsage(stats, generation.Usage)
	}
	modelStats := make([]ModelTokenStats, 0, len(statsByModel))
	total := ModelTokenStats{ModelName: "TOTAL"}
	for _, stats := range statsByModel {
		finalizeTokenStats(stats)
		modelStats = append(modelStats, *stats)
		addTokenStats(&total, *stats)
	}
	sort.Slice(modelStats, func(left, right int) bool {
		return modelStats[left].ModelName < modelStats[right].ModelName
	})
	finalizeTokenStats(&total)
	return SessionAggregateMetrics{TotalStats: total, ModelStats: modelStats}
}

func applyGenerationUsage(stats *ModelTokenStats, usage UsageObservation) {
	if !usage.HasUncachedInputTokens {
		return
	}
	stats.TurnCount++
	stats.UncachedInputTokenSum += usage.UncachedInputTokens
	if usage.HasCachedInputTokens {
		stats.CachedInputTokenSum += usage.CachedInputTokens
		stats.ExplicitCacheValueTurnCount++
	} else {
		stats.InferredZeroCacheTurnCount++
	}
	if usage.HasObservedContextTokens {
		stats.ObservedContextTokenSum += usage.ObservedContextTokens
		stats.ObservedContextValueTurnCount++
	}
}

func addTokenStats(total *ModelTokenStats, stats ModelTokenStats) {
	total.TurnCount += stats.TurnCount
	total.UncachedInputTokenSum += stats.UncachedInputTokenSum
	total.CachedInputTokenSum += stats.CachedInputTokenSum
	total.ExplicitCacheValueTurnCount += stats.ExplicitCacheValueTurnCount
	total.InferredZeroCacheTurnCount += stats.InferredZeroCacheTurnCount
	total.ObservedContextTokenSum += stats.ObservedContextTokenSum
	total.ObservedContextValueTurnCount += stats.ObservedContextValueTurnCount
}

func finalizeTokenStats(stats *ModelTokenStats) {
	stats.TotalProcessedTokenSum = stats.UncachedInputTokenSum + stats.CachedInputTokenSum
	if stats.TotalProcessedTokenSum > 0 {
		stats.CacheInputSharePercent = float64(stats.CachedInputTokenSum) * percentageMultiplier / float64(stats.TotalProcessedTokenSum)
	}
}
