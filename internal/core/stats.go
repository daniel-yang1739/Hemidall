package core

import (
	"fmt"
	"math"
	"sort"
)

// GetModelDiscount returns the official prompt cache discount rate, price factor, and label
func GetModelDiscount(modelName string) (discountRate float64, priceFactor float64, label string) {
	spec := ResolveModelSpec(modelName, nil)
	discountRate = spec.CacheDiscountRate
	priceFactor = math.Round((1.0-discountRate)*100.0) / 100.0
	label = fmt.Sprintf("%.2fx", priceFactor)
	return discountRate, priceFactor, label
}

// ComputeSessionAggregateMetrics calculates aggregate token statistics grouped by model and in total
func ComputeSessionAggregateMetrics(history []UnifiedAgentEvent) SessionAggregateMetrics {
	modelMap := make(map[string]*ModelTokenStats)

	var totalProcessed, totalCached, totalNew, totalCloudTurns int
	var weightedEffectiveTotal int

	for _, e := range history {
		// Only genuine cloud inference steps participate in session billing & caching aggregates
		if !e.IsCloudStep() {
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

		if cTokens > pTokens {
			cTokens = pTokens
		}
		if nTokens == 0 || nTokens > pTokens {
			nTokens = pTokens - cTokens
		}
		if nTokens < 0 {
			nTokens = 0
		}
		if pTokens < cTokens+nTokens {
			pTokens = cTokens + nTokens
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
			stats.EffectiveTokens = stats.TotalNew + int(math.Round(float64(stats.TotalCached)*stats.PriceFactor))
			stats.TokensSaved = int(math.Round(float64(stats.TotalCached) * stats.DiscountRate))
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
		DiscountLabel:     "0.25x",
		EffectiveTokens:   weightedEffectiveTotal,
		TokensSaved:       totalSavings,
		SavingsPercentage: totalSavingsPct,
	}

	return SessionAggregateMetrics{
		TotalStats: totalStats,
		ModelStats: modelStatsList,
	}
}
