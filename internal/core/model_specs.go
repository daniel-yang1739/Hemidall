package core

import "strings"

const (
	gemini37FlashModelID                     = "gemini-3.7-flash"
	gemini37FlashStandardInputUSDPerMillion  = 0.75
	gemini37FlashCachedInputUSDPerMillion    = 0.075
	claudeSonnet46ModelID                    = "claude-sonnet-4-6"
	claudeSonnet46StandardInputUSDPerMillion = 3.30
	claudeSonnet46CachedInputUSDPerMillion   = 0.33
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
	case claudeSonnet46ModelID:
		return ModelPricingSpec{
			ModelID:                    claudeSonnet46ModelID,
			StandardInputUSDPerMillion: claudeSonnet46StandardInputUSDPerMillion,
			CachedInputUSDPerMillion:   claudeSonnet46CachedInputUSDPerMillion,
			SourceLabel:                "Google Cloud Agent Platform Claude Sonnet 4.6 standard pricing",
			SourceURL:                  "https://cloud.google.com/gemini-enterprise-agent-platform/generative-ai/pricing",
		}, true
	default:
		return ModelPricingSpec{}, false
	}
}

// CacheAdjustedInputProjection is the standard-input-price equivalent for one
// model. It combines the paired metered-input and cached-content counters from
// the observed usage message; it is not an Antigravity invoice.
type CacheAdjustedInputProjection struct {
	ModelID              string
	ObservedTurns        int
	TotalProcessedTokens int
	CachedTokens         int
	MeteredInputTokens   int
	CacheInputMultiplier float64
	EffectiveInputTokens float64
	SourceLabel          string
	SourceURL            string
}

// ProjectCacheAdjustedInput computes metered input plus cached content at the
// official cache-input price ratio. It deliberately does not use the separate
// persisted context-state counter as an input to the formula.
func ProjectCacheAdjustedInput(stats ModelTokenStats) (CacheAdjustedInputProjection, bool) {
	if stats.TurnCount == 0 || stats.TotalProcessedTokenSum <= 0 {
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
		ObservedTurns:        stats.TurnCount,
		TotalProcessedTokens: stats.TotalProcessedTokenSum,
		CachedTokens:         stats.CachedContentTokenSum,
		MeteredInputTokens:   stats.MeteredInputTokenSum,
		CacheInputMultiplier: multiplier,
		EffectiveInputTokens: float64(stats.MeteredInputTokenSum) + float64(stats.CachedContentTokenSum)*multiplier,
		SourceLabel:          spec.SourceLabel,
		SourceURL:            spec.SourceURL,
	}, true
}
