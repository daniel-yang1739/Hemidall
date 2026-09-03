package core

import "strings"

const (
	gemini37FlashModelID                     = "gemini-3.7-flash"
	gemini38FlashModelID                     = "gemini-3.8-flash"
	gemini37FlashStandardInputUSDPerMillion  = 0.75
	gemini37FlashCachedInputUSDPerMillion    = 0.075
	freeInputUSDPerMillion                   = 0.0
	gemini37FlashHighModelID                 = "gemini-3.7-flash-high"
	gemini37FlashSafetyLEModelID             = "gemini-3.7-flash-safety-le"
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
	CachedSavingsMultiplier    float64
	SourceLabel                string
	SourceURL                  string
}

// CacheInputMultiplier returns the cached-input price as a fraction of the
// standard input price. A zero ratio is also valid for an explicitly free model.
func (spec ModelPricingSpec) CacheInputMultiplier() (float64, bool) {
	if spec.StandardInputUSDPerMillion == freeInputUSDPerMillion && spec.CachedInputUSDPerMillion == freeInputUSDPerMillion {
		return freeInputUSDPerMillion, true
	}
	if spec.StandardInputUSDPerMillion <= freeInputUSDPerMillion || spec.CachedInputUSDPerMillion < freeInputUSDPerMillion {
		return 0, false
	}
	return spec.CachedInputUSDPerMillion / spec.StandardInputUSDPerMillion, true
}

// StandardInputMultiplier returns one for metered models and zero for an
// explicitly free model. It keeps the price-equivalent formula well-defined
// without treating a free model as a discounted paid model.
func (spec ModelPricingSpec) StandardInputMultiplier() (float64, bool) {
	if spec.StandardInputUSDPerMillion == freeInputUSDPerMillion && spec.CachedInputUSDPerMillion == freeInputUSDPerMillion {
		return freeInputUSDPerMillion, true
	}
	if spec.StandardInputUSDPerMillion > freeInputUSDPerMillion {
		return 1, true
	}
	return freeInputUSDPerMillion, false
}

// ResolveModelPricing returns a profile only for an exact, verified model ID.
// It intentionally does not use fuzzy family matching because cache prices can
// differ across model revisions, endpoint tiers, and providers.
func ResolveModelPricing(modelID string) (ModelPricingSpec, bool) {
	switch strings.ToLower(strings.TrimSpace(modelID)) {
	case gemini37FlashModelID, gemini38FlashModelID:
		resolvedID := gemini37FlashModelID
		if strings.ToLower(strings.TrimSpace(modelID)) == gemini38FlashModelID {
			resolvedID = gemini38FlashModelID
		}
		return ModelPricingSpec{
			ModelID:                    resolvedID,
			StandardInputUSDPerMillion: gemini37FlashStandardInputUSDPerMillion,
			CachedInputUSDPerMillion:   gemini37FlashCachedInputUSDPerMillion,
			CachedSavingsMultiplier:    1,
			SourceLabel:                "Google Gemini Developer API paid standard pricing",
			SourceURL:                  "https://ai.google.dev/gemini-api/docs/pricing",
		}, true
	case gemini37FlashHighModelID:
		return ModelPricingSpec{ModelID: gemini37FlashHighModelID, StandardInputUSDPerMillion: gemini37FlashStandardInputUSDPerMillion, CachedInputUSDPerMillion: gemini37FlashCachedInputUSDPerMillion, CachedSavingsMultiplier: 1, SourceLabel: "Gemini Flash High pricing policy", SourceURL: ""}, true
	case gemini37FlashSafetyLEModelID:
		return ModelPricingSpec{ModelID: gemini37FlashSafetyLEModelID, StandardInputUSDPerMillion: freeInputUSDPerMillion, CachedInputUSDPerMillion: freeInputUSDPerMillion, CachedSavingsMultiplier: freeInputUSDPerMillion, SourceLabel: "Antigravity safety model policy", SourceURL: ""}, true
	case claudeSonnet46ModelID:
		return ModelPricingSpec{
			ModelID:                    claudeSonnet46ModelID,
			StandardInputUSDPerMillion: claudeSonnet46StandardInputUSDPerMillion,
			CachedInputUSDPerMillion:   claudeSonnet46CachedInputUSDPerMillion,
			CachedSavingsMultiplier:    1,
			SourceLabel:                "Google Cloud Agent Platform Claude Sonnet 4.6 standard pricing",
			SourceURL:                  "https://cloud.google.com/gemini-enterprise-agent-platform/generative-ai/pricing",
		}, true
	default:
		return ModelPricingSpec{}, false
	}
}

// CacheAdjustedInputProjection is the standard-input-price equivalent for one
// model. It combines the paired uncached-input and cached-input counters from
// the observed usage message; it is not an Antigravity invoice.
type CacheAdjustedInputProjection struct {
	ModelID              string
	ObservedTurns        int
	TotalProcessedTokens int
	CachedTokens         int
	UncachedInputTokens  int
	CacheInputMultiplier float64
	EffectiveInputTokens float64
	SourceLabel          string
	SourceURL            string
}

// ProjectCacheAdjustedInput computes uncached input plus cached input at the
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
	standardMultiplier, standardValid := spec.StandardInputMultiplier()
	if !standardValid {
		return CacheAdjustedInputProjection{}, false
	}
	return CacheAdjustedInputProjection{
		ModelID:              spec.ModelID,
		ObservedTurns:        stats.TurnCount,
		TotalProcessedTokens: stats.TotalProcessedTokenSum,
		CachedTokens:         stats.CachedInputTokenSum,
		UncachedInputTokens:  stats.UncachedInputTokenSum,
		CacheInputMultiplier: multiplier,
		EffectiveInputTokens: (float64(stats.UncachedInputTokenSum) + float64(stats.CachedInputTokenSum)*multiplier) * standardMultiplier,
		SourceLabel:          spec.SourceLabel,
		SourceURL:            spec.SourceURL,
	}, true
}
