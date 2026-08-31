package core

import "sort"

// ComputeSessionAggregateMetrics groups persisted usage observations by their
// recorded model name. Local estimates and unavailable cache fields are never
// promoted into this aggregate.
func ComputeSessionAggregateMetrics(history []UnifiedAgentEvent) SessionAggregateMetrics {
	byModel := make(map[string]*ModelTokenStats)

	for _, event := range history {
		usage := event.Usage
		if !usage.Available || !usage.HasTotalTokens {
			continue
		}
		modelName := usage.ModelName
		if modelName == "" {
			modelName = "unknown"
		}
		stats := byModel[modelName]
		if stats == nil {
			stats = &ModelTokenStats{ModelName: modelName, CompleteUsage: true}
			byModel[modelName] = stats
		}

		stats.TurnCount++
		stats.TotalProcessed += usage.TotalTokens
		if uncached, ok := usage.UncachedTokens(); ok {
			stats.CachedTurnCount++
			stats.TotalCached += usage.CachedTokens
			stats.TotalNew += uncached
			stats.ComparableTokens += usage.TotalTokens
		} else {
			stats.CompleteUsage = false
		}
	}

	models := make([]ModelTokenStats, 0, len(byModel))
	total := ModelTokenStats{ModelName: "TOTAL OBSERVED", CompleteUsage: true}
	for _, stats := range byModel {
		if stats.ComparableTokens > 0 {
			stats.CacheHitRate = float64(stats.TotalCached) / float64(stats.ComparableTokens) * 100.0
		}
		if stats.CachedTurnCount != stats.TurnCount || stats.ComparableTokens == 0 {
			stats.CompleteUsage = false
		}
		total.TurnCount += stats.TurnCount
		total.CachedTurnCount += stats.CachedTurnCount
		total.TotalProcessed += stats.TotalProcessed
		total.TotalCached += stats.TotalCached
		total.TotalNew += stats.TotalNew
		total.ComparableTokens += stats.ComparableTokens
		if !stats.CompleteUsage {
			total.CompleteUsage = false
		}
		models = append(models, *stats)
	}
	if total.ComparableTokens > 0 {
		total.CacheHitRate = float64(total.TotalCached) / float64(total.ComparableTokens) * 100.0
	}
	if total.CachedTurnCount != total.TurnCount || total.ComparableTokens == 0 {
		total.CompleteUsage = false
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].TotalProcessed > models[j].TotalProcessed
	})
	return SessionAggregateMetrics{TotalStats: total, ModelStats: models}
}
