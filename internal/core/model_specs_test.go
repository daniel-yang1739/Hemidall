package core

import "testing"

func TestModelInfo_Pos_ResolvesExactVerifiedGemini37Model(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini37Flash)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini37Flash, info.ID)
	requireModelSpecFloatEqual(t, 0.75, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.075, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 3.00, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_ResolvesExactVerifiedGemini38Model(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini38Flash)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini38Flash, info.ID)
	requireModelSpecFloatEqual(t, 0.75, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.075, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 3.00, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_ResolvesExactVerifiedClaudeModel(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelClaudeSonnet46)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelClaudeSonnet46, info.ID)
	requireModelSpecFloatEqual(t, 3.30, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.33, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 15.00, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_UsesFreePolicyOnlyForSafetyModel(t *testing.T) {
	highInfo, highFound := ResolveModelInfo(ProviderVertexAI, ModelGemini37FlashHigh)
	safetyInfo, safetyFound := ResolveModelInfo(ProviderVertexAI, ModelGemini37FlashSafety)

	requireModelSpecEqual(t, true, highFound)
	requireModelSpecEqual(t, false, highInfo.IsFree)

	requireModelSpecEqual(t, true, safetyFound)
	requireModelSpecEqual(t, true, safetyInfo.IsFree)
	requireModelSpecFloatEqual(t, 0, safetyInfo.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0, safetyInfo.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 0, safetyInfo.OutputUSDPerMillion)
	safetyMultiplier, multiplierAvailable := safetyInfo.CacheInputMultiplier()
	requireModelSpecEqual(t, true, multiplierAvailable)
	requireModelSpecFloatEqual(t, 0, safetyMultiplier)

	cost := safetyInfo.CalculateCost(100_000, 100_000, 100_000)
	requireModelSpecFloatEqual(t, 0, cost)
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
