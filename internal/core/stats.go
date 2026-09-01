package core

import "sort"

const cacheSharePercentageScale = 100.0

// ComputeSessionAggregateMetrics groups observed metered input and cache-content
// fields by recorded model name. Both fields come from the same persisted usage
// message. ObservedContextTokens remains a separate context-state diagnostic.
func ComputeSessionAggregateMetrics(history []UnifiedAgentEvent) SessionAggregateMetrics {
	byModel := make(map[string]*ModelTokenStats)

	for _, event := range history {
		usage := event.Usage
		if !usage.Available || !usage.HasMeteredInputTokens {
			continue
		}
		modelName := usage.ModelName
		if modelName == "" {
			modelName = "unknown"
		}
		stats := byModel[modelName]
		if stats == nil {
			stats = &ModelTokenStats{ModelName: modelName}
			byModel[modelName] = stats
		}

		stats.TurnCount++
		stats.MeteredInputTokenSum += usage.MeteredInputTokens
		stats.TotalProcessedTokenSum += usage.MeteredInputTokens
		if usage.HasCachedContentTokens {
			stats.ExplicitCacheValueTurnCount++
			stats.CachedContentTokenSum += usage.CachedContentTokens
		} else {
			stats.DefaultZeroCacheTurnCount++
		}
		stats.TotalProcessedTokenSum += usage.CachedContentTokens
		if usage.HasObservedContextTokens {
			stats.ObservedContextValueTurnCount++
			stats.ObservedContextTokenSum += usage.ObservedContextTokens
		}
	}

	models := make([]ModelTokenStats, 0, len(byModel))
	total := ModelTokenStats{ModelName: "TOTAL OBSERVED"}
	for _, stats := range byModel {
		if stats.TotalProcessedTokenSum > 0 {
			stats.CacheContentSharePercent = float64(stats.CachedContentTokenSum) / float64(stats.TotalProcessedTokenSum) * cacheSharePercentageScale
		}
		if projection, available := ProjectCacheAdjustedInput(*stats); available {
			stats.EffectiveInputTokenSum = projection.EffectiveInputTokens
			stats.EffectiveProjectionTurnCount = stats.TurnCount
		}
		total.TurnCount += stats.TurnCount
		total.MeteredInputTokenSum += stats.MeteredInputTokenSum
		total.CachedContentTokenSum += stats.CachedContentTokenSum
		total.TotalProcessedTokenSum += stats.TotalProcessedTokenSum
		total.ExplicitCacheValueTurnCount += stats.ExplicitCacheValueTurnCount
		total.DefaultZeroCacheTurnCount += stats.DefaultZeroCacheTurnCount
		total.EffectiveInputTokenSum += stats.EffectiveInputTokenSum
		total.EffectiveProjectionTurnCount += stats.EffectiveProjectionTurnCount
		total.ObservedContextTokenSum += stats.ObservedContextTokenSum
		total.ObservedContextValueTurnCount += stats.ObservedContextValueTurnCount
		models = append(models, *stats)
	}
	if total.TotalProcessedTokenSum > 0 {
		total.CacheContentSharePercent = float64(total.CachedContentTokenSum) / float64(total.TotalProcessedTokenSum) * cacheSharePercentageScale
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].TotalProcessedTokenSum > models[j].TotalProcessedTokenSum
	})
	return SessionAggregateMetrics{TotalStats: total, ModelStats: models}
}
