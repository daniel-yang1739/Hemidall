package core

import "testing"

func TestModelInfo_Pos_ResolvesExactVerifiedGemini37Model(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini37Flash)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini37Flash, info.ID)
	requireModelSpecFloatEqual(t, 0.75, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.075, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 3.75, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_ResolvesExactVerifiedGemini36Flash(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini36Flash)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini36Flash, info.ID)
	requireModelSpecFloatEqual(t, 0.75, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.075, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 3.75, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_ResolvesExactVerifiedGemini35Flash(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini35Flash)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini35Flash, info.ID)
	requireModelSpecFloatEqual(t, 1.50, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.15, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 9.00, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_ResolvesExactVerifiedGemini35FlashLite(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini35FlashLite)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini35FlashLite, info.ID)
	requireModelSpecFloatEqual(t, 0.30, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.03, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 2.50, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_ResolvesExactVerifiedGemini31FlashLite(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini31FlashLite)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini31FlashLite, info.ID)
	requireModelSpecFloatEqual(t, 0.25, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.025, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 1.50, info.OutputUSDPerMillion)
	multiplier, ok := info.CacheInputMultiplier()
	requireModelSpecEqual(t, true, ok)
	requireModelSpecFloatEqual(t, 0.10, multiplier)
}

func TestModelInfo_Pos_ResolvesExactVerifiedGemini31ProPreview(t *testing.T) {
	info, found := ResolveModelInfo(ProviderVertexAI, ModelGemini31ProPreview)
	requireModelSpecEqual(t, true, found)
	requireModelSpecEqual(t, ModelGemini31ProPreview, info.ID)
	requireModelSpecFloatEqual(t, 2.00, info.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.20, info.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 12.00, info.OutputUSDPerMillion)
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
	requireModelSpecFloatEqual(t, 3.75, info.OutputUSDPerMillion)
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
	requireModelSpecFloatEqual(t, 16.50, info.OutputUSDPerMillion)
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

	// 1M uncached ($0.75) + 1M cached ($0.075) + 1M output ($3.75) = $4.575
	cost := info.CalculateCost(1_000_000, 1_000_000, 1_000_000)
	requireModelSpecFloatEqual(t, 4.575, cost)
}

func TestResolveModelInfo_Pos_AnthropicOfficialCatalogAndPrefixes(t *testing.T) {
	// Anthropic direct Claude Sonnet 4.6 ($3.00 / $0.30 / $15.00)
	sonnet46, found46 := ResolveModelInfo(ProviderAnthropic, ModelClaudeSonnet46)
	requireModelSpecEqual(t, true, found46)
	requireModelSpecEqual(t, ProviderAnthropic, sonnet46.Provider)
	requireModelSpecFloatEqual(t, 3.00, sonnet46.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.30, sonnet46.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 15.00, sonnet46.OutputUSDPerMillion)
	requireModelSpecEqual(t, "https://platform.claude.com/docs/en/about-claude/pricing", sonnet46.SourceURL)

	// Vertex AI hosted Claude Sonnet 4.6 has 10% premium ($3.30)
	vertexSonnet46, vertexFound := ResolveModelInfo(ProviderVertexAI, ModelClaudeSonnet46)
	requireModelSpecEqual(t, true, vertexFound)
	requireModelSpecFloatEqual(t, 3.30, vertexSonnet46.InputUSDPerMillion)

	// Anthropic direct Claude Sonnet 5 ($2.00 / $0.20 / $10.00)
	sonnet5, found5 := ResolveModelInfo(ProviderAnthropic, ModelClaudeSonnet5)
	requireModelSpecEqual(t, true, found5)
	requireModelSpecFloatEqual(t, 2.00, sonnet5.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.20, sonnet5.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 10.00, sonnet5.OutputUSDPerMillion)

	// Provider-prefixed alias string resolution
	prefixed, foundPrefixed := ResolveModelInfoByString(ProviderAnthropic, "anthropic/claude-sonnet-4-6")
	requireModelSpecEqual(t, true, foundPrefixed)
	requireModelSpecEqual(t, ModelClaudeSonnet46, prefixed.ID)
}

func TestResolveModelInfo_Pos_OpenAIOfficialCatalogAndPrefixes(t *testing.T) {
	// GPT-4o ($2.50 / $1.25 / $10.00, 50% cache discount)
	gpt4o, found4o := ResolveModelInfo(ProviderOpenAI, ModelGPT4o)
	requireModelSpecEqual(t, true, found4o)
	requireModelSpecEqual(t, ProviderOpenAI, gpt4o.Provider)
	requireModelSpecEqual(t, "OpenAI", gpt4o.Vendor)
	requireModelSpecFloatEqual(t, 2.50, gpt4o.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 1.25, gpt4o.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 10.00, gpt4o.OutputUSDPerMillion)
	requireModelSpecEqual(t, "https://developers.openai.com/api/docs/pricing", gpt4o.SourceURL)

	// GPT-5 ($1.25 / $0.125 / $10.00, 90% cache discount)
	gpt5, found5 := ResolveModelInfo(ProviderOpenAI, ModelGPT5)
	requireModelSpecEqual(t, true, found5)
	requireModelSpecFloatEqual(t, 1.25, gpt5.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.125, gpt5.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 10.00, gpt5.OutputUSDPerMillion)

	// o3 reasoning model ($2.00 / $0.50 / $8.00, 75% cache discount)
	o3, foundO3 := ResolveModelInfo(ProviderOpenAI, ModelO3)
	requireModelSpecEqual(t, true, foundO3)
	requireModelSpecFloatEqual(t, 2.00, o3.InputUSDPerMillion)
	requireModelSpecFloatEqual(t, 0.50, o3.CachedInputUSDPerMillion())
	requireModelSpecFloatEqual(t, 8.00, o3.OutputUSDPerMillion)

	// String alias resolution with provider prefix
	prefixed, foundPrefixed := ResolveModelInfoByString(ProviderOpenAI, "openai/gpt-4o")
	requireModelSpecEqual(t, true, foundPrefixed)
	requireModelSpecEqual(t, ModelGPT4o, prefixed.ID)
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
