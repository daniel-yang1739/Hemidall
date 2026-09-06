package core

import "sort"

const cacheSharePercentageScale = 100.0

// ComputeSessionAggregateMetrics groups schema-inferred uncached and cached input
// fields by recorded model name. Both fields come from the same persisted usage
// message. ObservedContextTokens remains a separate context-state diagnostic.
func ComputeSessionAggregateMetrics(history []UnifiedAgentEvent) SessionAggregateMetrics {
	byModel := make(map[string]*ModelTokenStats)

	for _, event := range history {
		usage := event.Usage
		if !usage.Available || !usage.HasUncachedInputTokens {
			continue
		}
		modelName := usage.ModelName
		if modelName == "" {
			modelName = "unknown"
		}
		stats := byModel[modelName]
		if stats == nil {
			provider := usage.Provider
			if provider == "" {
				provider = ProviderVertexAI
			}
			stats = &ModelTokenStats{
				Provider:  provider,
				ModelName: modelName,
				Model:     usage.Model,
			}
			if stats.Model == "" {
				stats.Model = NormalizeModelID(modelName)
			}
			byModel[modelName] = stats
		}

		stats.TurnCount++
		stats.UncachedInputTokenSum += usage.UncachedInputTokens
		stats.TotalProcessedTokenSum += usage.UncachedInputTokens
		if usage.HasCachedInputTokens {
			stats.ExplicitCacheValueTurnCount++
			stats.CachedInputTokenSum += usage.CachedInputTokens
		} else {
			stats.InferredZeroCacheTurnCount++
		}
		stats.TotalProcessedTokenSum += usage.CachedInputTokens
		if usage.HasObservedContextTokens {
			stats.ObservedContextValueTurnCount++
			stats.ObservedContextTokenSum += usage.ObservedContextTokens
		}
		turnOutput := usage.ThinkingOutputTokens + usage.OutputContentTokens
		stats.ThinkingOutputTokenSum += usage.ThinkingOutputTokens
		stats.ContentOutputTokenSum += usage.OutputContentTokens
		stats.TotalOutputTokenSum += turnOutput
	}

	models := make([]ModelTokenStats, 0, len(byModel))
	total := ModelTokenStats{ModelName: "TOTAL OBSERVED"}
	for _, stats := range byModel {
		if stats.TotalProcessedTokenSum > 0 {
			stats.CacheInputSharePercent = float64(stats.CachedInputTokenSum) / float64(stats.TotalProcessedTokenSum) * cacheSharePercentageScale
		}
		if info, found := ResolveModelInfo(stats.Provider, stats.Model); found {
			stats.EstimatedCostUSD = info.CalculateCost(stats.UncachedInputTokenSum, stats.CachedInputTokenSum, stats.TotalOutputTokenSum)
			stats.HasEstimatedCost = true
		} else if info, found := ResolveModelInfoByString(stats.Provider, stats.ModelName); found {
			stats.EstimatedCostUSD = info.CalculateCost(stats.UncachedInputTokenSum, stats.CachedInputTokenSum, stats.TotalOutputTokenSum)
			stats.HasEstimatedCost = true
		}
		total.TurnCount += stats.TurnCount
		total.UncachedInputTokenSum += stats.UncachedInputTokenSum
		total.CachedInputTokenSum += stats.CachedInputTokenSum
		total.TotalProcessedTokenSum += stats.TotalProcessedTokenSum
		total.TotalOutputTokenSum += stats.TotalOutputTokenSum
		total.ThinkingOutputTokenSum += stats.ThinkingOutputTokenSum
		total.ContentOutputTokenSum += stats.ContentOutputTokenSum
		if stats.HasEstimatedCost {
			total.EstimatedCostUSD += stats.EstimatedCostUSD
			total.HasEstimatedCost = true
		}
		total.ExplicitCacheValueTurnCount += stats.ExplicitCacheValueTurnCount
		total.InferredZeroCacheTurnCount += stats.InferredZeroCacheTurnCount
		total.ObservedContextTokenSum += stats.ObservedContextTokenSum
		total.ObservedContextValueTurnCount += stats.ObservedContextValueTurnCount
		models = append(models, *stats)
	}
	if total.TotalProcessedTokenSum > 0 {
		total.CacheInputSharePercent = float64(total.CachedInputTokenSum) / float64(total.TotalProcessedTokenSum) * cacheSharePercentageScale
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].TotalProcessedTokenSum > models[j].TotalProcessedTokenSum
	})
	return SessionAggregateMetrics{TotalStats: total, ModelStats: models}
}
