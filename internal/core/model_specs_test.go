package core

import "testing"

const (
	modelSpecCacheMultiplier = 0.10
)

func TestModelPricing_Pos_ResolvesExactVerifiedGeminiModel(t *testing.T) {
	spec, available := ResolveModelPricing(gemini37FlashModelID)
	multiplier, multiplierAvailable := spec.CacheInputMultiplier()

	requireModelSpecEqual(t, true, available)
	requireModelSpecEqual(t, gemini37FlashModelID, spec.ModelID)
	requireModelSpecEqual(t, true, multiplierAvailable)
	requireModelSpecFloatEqual(t, modelSpecCacheMultiplier, multiplier)
}

func TestModelPricing_Pos_ResolvesExactVerifiedGemini38Model(t *testing.T) {
	spec, available := ResolveModelPricing(gemini38FlashModelID)
	multiplier, multiplierAvailable := spec.CacheInputMultiplier()

	requireModelSpecEqual(t, true, available)
	requireModelSpecEqual(t, gemini38FlashModelID, spec.ModelID)
	requireModelSpecEqual(t, true, multiplierAvailable)
	requireModelSpecFloatEqual(t, modelSpecCacheMultiplier, multiplier)
	requireModelSpecFloatEqual(t, gemini37FlashStandardInputUSDPerMillion, spec.StandardInputUSDPerMillion)
	requireModelSpecFloatEqual(t, gemini37FlashCachedInputUSDPerMillion, spec.CachedInputUSDPerMillion)
}

func TestModelPricing_Pos_ResolvesExactVerifiedClaudeModel(t *testing.T) {
	spec, available := ResolveModelPricing(claudeSonnet46ModelID)
	multiplier, multiplierAvailable := spec.CacheInputMultiplier()

	requireModelSpecEqual(t, true, available)
	requireModelSpecEqual(t, claudeSonnet46ModelID, spec.ModelID)
	requireModelSpecEqual(t, true, multiplierAvailable)
	requireModelSpecFloatEqual(t, modelSpecCacheMultiplier, multiplier)
}

func TestModelPricing_Pos_UsesFreeSavingsPolicyOnlyForSafetyModel(t *testing.T) {
	highSpec, highAvailable := ResolveModelPricing(gemini37FlashHighModelID)
	safetySpec, safetyAvailable := ResolveModelPricing(gemini37FlashSafetyLEModelID)

	requireModelSpecEqual(t, true, highAvailable)
	requireModelSpecFloatEqual(t, 1, highSpec.CachedSavingsMultiplier)
	requireModelSpecEqual(t, true, safetyAvailable)
	requireModelSpecFloatEqual(t, 0, safetySpec.StandardInputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0, safetySpec.CachedInputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0, safetySpec.CachedSavingsMultiplier)
	safetyMultiplier, multiplierAvailable := safetySpec.CacheInputMultiplier()
	requireModelSpecEqual(t, true, multiplierAvailable)
	requireModelSpecFloatEqual(t, 0, safetyMultiplier)
}

func TestCacheAdjustedInputProjection_Pos_UsesZeroEquivalentForFreeSafetyModel(t *testing.T) {
	stats := ModelTokenStats{
		ModelName:              gemini37FlashSafetyLEModelID,
		TurnCount:              1,
		UncachedInputTokenSum:  1_000,
		CachedInputTokenSum:    9_000,
		TotalProcessedTokenSum: 10_000,
	}

	projection, available := ProjectCacheAdjustedInput(stats)

	requireModelSpecEqual(t, true, available)
	requireModelSpecFloatEqual(t, 0, projection.CacheInputMultiplier)
	requireModelSpecFloatEqual(t, 0, projection.EffectiveInputTokens)
}

func TestModelPricing_Neg_DoesNotResolveUnknownModel(t *testing.T) {
	_, available := ResolveModelPricing("unknown")

	requireModelSpecEqual(t, false, available)
}

func TestCacheAdjustedInputProjection_UsesPairedMeteredUsageCounters(t *testing.T) {
	stats := ModelTokenStats{
		ModelName:              gemini37FlashModelID,
		TurnCount:              1,
		UncachedInputTokenSum:  1_000,
		CachedInputTokenSum:    9_000,
		TotalProcessedTokenSum: 10_000,
	}

	projection, available := ProjectCacheAdjustedInput(stats)

	requireModelSpecEqual(t, true, available)
	requireModelSpecEqual(t, 1_000, projection.UncachedInputTokens)
	requireModelSpecEqual(t, 9_000, projection.CachedTokens)
	requireModelSpecFloatEqual(t, 1_900, projection.EffectiveInputTokens)
}

func TestCacheAdjustedInputProjection_Neg_RejectsUnpricedModel(t *testing.T) {
	stats := ModelTokenStats{
		ModelName:              "unknown",
		TurnCount:              1,
		UncachedInputTokenSum:  1_000,
		CachedInputTokenSum:    9_000,
		TotalProcessedTokenSum: 10_000,
	}

	_, available := ProjectCacheAdjustedInput(stats)

	requireModelSpecEqual(t, false, available)
}

func TestResolveModelInfo_Pos_ResolvesTwoTierCatalogAndAliases(t *testing.T) {
	// Canonical Gemini on Vertex AI
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini25Pro)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini25Pro, info.ID)
	requireModelSpecEqual(t, ProviderVertexAI, info.Provider)
	requireModelSpecEqual(t, "Google", info.Vendor)

	// String alias resolution
	infoAlias, foundAlias := ResolveModelInfoByString(ProviderVertexAI, "gemini-pro")
	requireModelSpecEqual(t, true, foundAlias)
	requireModelSpecEqual(t, ModelGemini25Pro, infoAlias.ID)

	// Claude hosted on Vertex AI
	claudeInfo, claudeFound := ResolveModelInfoByString(ProviderVertexAI, "claude-3-7-sonnet")
	requireModelSpecEqual(t, true, claudeFound)
	requireModelSpecEqual(t, ModelClaude37Sonnet, claudeInfo.ID)
	requireModelSpecEqual(t, "Anthropic", claudeInfo.Vendor)
}

func TestResolveModelInfo_Pos_ComputesDerivedDiscountRates(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini25Flash)
	requireModelSpecEqual(t, true, found)
	requireModelSpecFloatEqual(t, 0.75, info.CacheDiscountRate)
	// 0.075 * (1 - 0.75) = 0.01875
	requireModelSpecFloatEqual(t, 0.01875, info.CachedInputUSDPerMillion())

	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.25, multiplier)
}

func TestResolveModelInfo_Neg_RejectsUnknownProviderAndModel(t *testing.T) {
	_, found := ResolveModelInfo(ProviderOpenAI, ModelUnknown)
	requireModelSpecEqual(t, false, found)

	_, foundRaw := ResolveModelInfoByString(ProviderVertexAI, "non-existent-model-xyz")
	requireModelSpecEqual(t, false, foundRaw)
}

func TestModelInfo_Pos_CalculatesCostCorrectly(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini37Flash)
	requireModelSpecEqual(t, true, found)

	// 1M uncached ($0.75) + 1M cached ($0.075) + 1M output ($3.00) = $3.825
	cost := info.CalculateCost(1_000_000, 1_000_000, 1_000_000)
	requireModelSpecFloatEqual(t, 3.825, cost)
}


func requireModelSpecEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func requireModelSpecFloatEqual(t *testing.T, want, got float64) {
	t.Helper()
	if difference := want - got; difference > modelSpecFloatTolerance || difference < -modelSpecFloatTolerance {
		t.Fatalf("want %v, got %v", want, got)
	}
}

const modelSpecFloatTolerance = 0.000001
