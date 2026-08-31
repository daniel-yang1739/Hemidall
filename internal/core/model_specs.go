package core

import "strings"

const (
	gemini37FlashModelID                    = "gemini-3.7-flash"
	gemini37FlashStandardInputUSDPerMillion = 0.75
	gemini37FlashCachedInputUSDPerMillion   = 0.075
)

// ModelPricingSpec is a verified, provider-specific input-pricing profile.
// It supports an input-price-equivalent projection only; it does not prove the
// account, endpoint tier, storage fees, output fees, or invoice used by a host.
type ModelPricingSpec struct {
	ModelID                    string
	StandardInputUSDPerMillion float64
	CachedInputUSDPerMillion   float64
	SourceLabel                string
	SourceURL                  string
}

// CacheInputMultiplier returns the cached-input price as a fraction of the
// standard input price. A ratio is available only for a complete positive spec.
func (spec ModelPricingSpec) CacheInputMultiplier() (float64, bool) {
	if spec.StandardInputUSDPerMillion <= 0 || spec.CachedInputUSDPerMillion < 0 {
		return 0, false
	}
	return spec.CachedInputUSDPerMillion / spec.StandardInputUSDPerMillion, true
}

// ResolveModelPricing returns a profile only for an exact, verified model ID.
// It intentionally does not use fuzzy family matching because cache prices can
// differ across model revisions, endpoint tiers, and providers.
func ResolveModelPricing(modelID string) (ModelPricingSpec, bool) {
	switch strings.ToLower(strings.TrimSpace(modelID)) {
	case gemini37FlashModelID:
		return ModelPricingSpec{
			ModelID:                    gemini37FlashModelID,
			StandardInputUSDPerMillion: gemini37FlashStandardInputUSDPerMillion,
			CachedInputUSDPerMillion:   gemini37FlashCachedInputUSDPerMillion,
			SourceLabel:                "Google Gemini Developer API paid standard pricing",
			SourceURL:                  "https://ai.google.dev/gemini-api/docs/pricing",
		}, true
	default:
		return ModelPricingSpec{}, false
	}
}

// CacheAdjustedInputProjection is the standard-input-price equivalent for the
// comparable cache subset of one model. It is not a provider invoice.
type CacheAdjustedInputProjection struct {
	ModelID              string
	ComparableTurns      int
	ComparableTokens     int
	CachedTokens         int
	UncachedTokens       int
	CacheInputMultiplier float64
	EffectiveInputTokens float64
	SourceLabel          string
	SourceURL            string
}

// ProjectCacheAdjustedInput computes uncached + cached * cache-price-ratio.
// The caller must keep different model profiles separate because a cross-model
// sum of price-equivalent tokens has no common token-price baseline.
func ProjectCacheAdjustedInput(stats ModelTokenStats) (CacheAdjustedInputProjection, bool) {
	if stats.CachedTurnCount == 0 || stats.ComparableTokens <= 0 {
		return CacheAdjustedInputProjection{}, false
	}
	spec, found := ResolveModelPricing(stats.ModelName)
	if !found {
		return CacheAdjustedInputProjection{}, false
	}
	multiplier, valid := spec.CacheInputMultiplier()
	if !valid {
		return CacheAdjustedInputProjection{}, false
	}
	return CacheAdjustedInputProjection{
		ModelID:              spec.ModelID,
		ComparableTurns:      stats.CachedTurnCount,
		ComparableTokens:     stats.ComparableTokens,
		CachedTokens:         stats.TotalCached,
		UncachedTokens:       stats.TotalNew,
		CacheInputMultiplier: multiplier,
		EffectiveInputTokens: float64(stats.TotalNew) + float64(stats.TotalCached)*multiplier,
		SourceLabel:          spec.SourceLabel,
		SourceURL:            spec.SourceURL,
	}, true
}
