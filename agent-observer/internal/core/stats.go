package core

import (
	"sort"
	"strings"
)

// GetModelDiscount returns the official prompt cache discount rate, price factor, and label
func GetModelDiscount(modelName string) (discountRate float64, priceFactor float64, label string) {
	name := strings.ToLower(modelName)
	switch {
	case strings.Contains(name, "claude"):
		return 0.90, 0.10, "0.10x (90% OFF)"
	case strings.Contains(name, "gemini"):
		return 0.75, 0.25, "0.25x (75% OFF)"
	case strings.Contains(name, "gpt-4") || strings.Contains(name, "o1") || strings.Contains(name, "o3"):
		return 0.50, 0.50, "0.50x (50% OFF)"
	case strings.Contains(name, "deepseek"):
		return 0.90, 0.10, "0.10x (90% OFF)"
	default:
		return 0.75, 0.25, "0.25x (75% OFF)" // Default Gemini standard
	}
}

// ComputeSessionAggregateMetrics calculates aggregate token statistics grouped by model and in total
func ComputeSessionAggregateMetrics(history []UnifiedAgentEvent) SessionAggregateMetrics {
	modelMap := make(map[string]*ModelTokenStats)

	var totalProcessed, totalCached, totalNew, totalCloudTurns int
	var weightedEffectiveTotal int

	for _, e := range history {
		if !e.IsCloudStep() && e.Tokens.TotalTokens == 0 {
			continue
		}

		mName := e.Tokens.OfficialModel
		if mName == "" {
			mName = "gemini-3.7-flash"
		}

		stats, exists := modelMap[mName]
		if !exists {
			discRate, pFactor, dLabel := GetModelDiscount(mName)
			stats = &ModelTokenStats{
				ModelName:     mName,
				DiscountRate:  discRate,
				PriceFactor:   pFactor,
				DiscountLabel: dLabel,
			}
			modelMap[mName] = stats
		}

		pTokens := e.Tokens.TotalTokens
		cTokens := e.Tokens.CachedTokens
		nTokens := e.Tokens.NewTokens
		if nTokens == 0 && pTokens > cTokens {
			nTokens = pTokens - cTokens
		}

		stats.TurnCount++
		stats.TotalProcessed += pTokens
		stats.TotalCached += cTokens
		stats.TotalNew += nTokens

		totalCloudTurns++
		totalProcessed += pTokens
		totalCached += cTokens
		totalNew += nTokens
	}

	var modelStatsList []ModelTokenStats
	for _, stats := range modelMap {
		if stats.TotalProcessed > 0 {
			stats.CacheHitRate = float64(stats.TotalCached) / float64(stats.TotalProcessed) * 100.0
			stats.EffectiveTokens = stats.TotalNew + int(float64(stats.TotalCached)*stats.PriceFactor)
			stats.TokensSaved = stats.TotalProcessed - stats.EffectiveTokens
			stats.SavingsPercentage = float64(stats.TokensSaved) / float64(stats.TotalProcessed) * 100.0
		} else {
			stats.EffectiveTokens = stats.TotalNew
			stats.TokensSaved = 0
			stats.SavingsPercentage = 0.0
		}
		weightedEffectiveTotal += stats.EffectiveTokens
		modelStatsList = append(modelStatsList, *stats)
	}

	// Sort models by TotalProcessed descending
	sort.Slice(modelStatsList, func(i, j int) bool {
		return modelStatsList[i].TotalProcessed > modelStatsList[j].TotalProcessed
	})

	totalSavings := totalProcessed - weightedEffectiveTotal
	if totalSavings < 0 {
		totalSavings = 0
	}
	var totalHitRate, totalSavingsPct float64
	if totalProcessed > 0 {
		totalHitRate = float64(totalCached) / float64(totalProcessed) * 100.0
		totalSavingsPct = float64(totalSavings) / float64(totalProcessed) * 100.0
	}

	totalStats := ModelTokenStats{
		ModelName:         "TOTAL SUMMARY",
		TurnCount:         totalCloudTurns,
		TotalProcessed:    totalProcessed,
		TotalCached:       totalCached,
		TotalNew:          totalNew,
		CacheHitRate:      totalHitRate,
		DiscountRate:      0.75,
		PriceFactor:       0.25,
		DiscountLabel:     "0.25x (75% OFF)",
		EffectiveTokens:   weightedEffectiveTotal,
		TokensSaved:       totalSavings,
		SavingsPercentage: totalSavingsPct,
	}

	return SessionAggregateMetrics{
		TotalStats: totalStats,
		ModelStats: modelStatsList,
	}
}

// ExtractTurnTrendSeries extracts chronological cloud turns for trend visualization
func ExtractTurnTrendSeries(history []UnifiedAgentEvent, maxPoints int) TurnTrendSeries {
	var points []TurnTrendPoint
	var maxCtx, peakNew, latestCached int
	var sumHitRate float64

	for _, e := range history {
		if !e.IsCloudStep() && e.Tokens.TotalTokens == 0 {
			continue
		}

		p := TurnTrendPoint{
			StepIndex:    e.StepIndex,
			TotalTokens:  e.Tokens.TotalTokens,
			CachedTokens: e.Tokens.CachedTokens,
			NewTokens:    e.Tokens.NewTokens,
			CacheHitRate: e.Tokens.CacheHitRate,
		}

		if p.TotalTokens > maxCtx {
			maxCtx = p.TotalTokens
		}
		if p.NewTokens > peakNew {
			peakNew = p.NewTokens
		}
		latestCached = p.CachedTokens
		sumHitRate += p.CacheHitRate

		points = append(points, p)
	}

	avgHit := 0.0
	if len(points) > 0 {
		avgHit = sumHitRate / float64(len(points))
	}

	// If maxPoints > 0 and len(points) > maxPoints, slice the latest maxPoints
	if maxPoints > 0 && len(points) > maxPoints {
		points = points[len(points)-maxPoints:]
	}

	return TurnTrendSeries{
		Points:       points,
		MaxContext:   maxCtx,
		PeakNew:      peakNew,
		AvgHitRate:   avgHit,
		LatestCached: latestCached,
	}
}
