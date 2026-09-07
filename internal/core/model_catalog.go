package core

// ProviderName identifies the hosting/execution infrastructure.
type ProviderName string

const (
	ProviderVertexAI  ProviderName = "vertex_ai"
	ProviderAnthropic ProviderName = "anthropic"
	ProviderOpenAI    ProviderName = "openai"
	ProviderUnknown   ProviderName = "unknown"
)

// ModelID identifies the canonical model revision.
type ModelID string

const (
	ModelGemini25Pro         ModelID = "gemini-2.5-pro"
	ModelGemini25Flash       ModelID = "gemini-2.5-flash"
	ModelGemini37Flash       ModelID = "gemini-3.7-flash"
	ModelGemini38Flash       ModelID = "gemini-3.8-flash"
	ModelGemini36Flash       ModelID = "gemini-3.6-flash"
	ModelGemini35Flash       ModelID = "gemini-3.5-flash"
	ModelGemini35FlashLite   ModelID = "gemini-3.5-flash-lite"
	ModelGemini31FlashLite   ModelID = "gemini-3.1-flash-lite"
	ModelGemini31ProPreview  ModelID = "gemini-3.1-pro-preview"
	ModelGemini37FlashHigh   ModelID = "gemini-3.7-flash-high"
	ModelGemini37FlashSafety ModelID = "gemini-3.7-flash-safety-le"
	ModelClaudeSonnet5       ModelID = "claude-sonnet-5"
	ModelClaudeSonnet46      ModelID = "claude-sonnet-4-6"
	ModelClaude37Sonnet      ModelID = "claude-3-7-sonnet"
	ModelClaude35Sonnet      ModelID = "claude-3-5-sonnet"
	ModelClaudeHaiku45       ModelID = "claude-haiku-4-5"
	ModelClaudeOpus5         ModelID = "claude-opus-5"
	ModelClaudeFable51       ModelID = "claude-fable-5-1"
	ModelClaudeFable5        ModelID = "claude-fable-5"
	ModelGPT5                ModelID = "gpt-5"
	ModelGPT5Mini            ModelID = "gpt-5-mini"
	ModelGPT4o               ModelID = "gpt-4o"
	ModelGPT4oMini           ModelID = "gpt-4o-mini"
	ModelGPT41               ModelID = "gpt-4.1"
	ModelO1                  ModelID = "o1"
	ModelO3                  ModelID = "o3"
	ModelO3Mini              ModelID = "o3-mini"
	ModelO4Mini              ModelID = "o4-mini"
	ModelGPT6Astra           ModelID = "gpt-6-astra"
	ModelGPT56Sol            ModelID = "gpt-5.6-sol"
	ModelGPT56Terra          ModelID = "gpt-5.6-terra"
	ModelGPT56Luna           ModelID = "gpt-5.6-luna"
	ModelUnknown             ModelID = "unknown"
)

// ModelInfo represents canonical vendor specifications, hosting provider, and pricing metrics.
type ModelInfo struct {
	Provider            ProviderName
	Vendor              string
	Name                string
	ID                  ModelID
	InputUSDPerMillion  float64
	OutputUSDPerMillion float64
	CacheDiscountRate   float64
	IsFree              bool
	SourceLabel         string
	SourceURL           string
}

// CachedInputUSDPerMillion computes the cached input rate derived from the discount rate.
func (m ModelInfo) CachedInputUSDPerMillion() float64 {
	if m.IsFree {
		return 0
	}
	return m.InputUSDPerMillion * (1.0 - m.CacheDiscountRate)
}

// CacheInputMultiplier returns the cached-input price as a fraction of standard input price.
func (m ModelInfo) CacheInputMultiplier() (float64, bool) {
	if m.IsFree {
		return 0, true
	}
	if m.InputUSDPerMillion <= 0 {
		return 0, false
	}
	return (1.0 - m.CacheDiscountRate), true
}

// StandardInputMultiplier returns 1 for metered models and 0 for explicitly free models.
func (m ModelInfo) StandardInputMultiplier() (float64, bool) {
	if m.IsFree {
		return 0, true
	}
	if m.InputUSDPerMillion > 0 {
		return 1, true
	}
	return 0, false
}

// CalculateCost computes the estimated USD cost based on uncached input, cached input, and output tokens.
func (m ModelInfo) CalculateCost(uncachedInput, cachedInput, outputTokens int) float64 {
	if m.IsFree {
		return 0
	}
	costUncached := (float64(uncachedInput) / 1_000_000.0) * m.InputUSDPerMillion
	costCached := (float64(cachedInput) / 1_000_000.0) * m.CachedInputUSDPerMillion()
	costOutput := (float64(outputTokens) / 1_000_000.0) * m.OutputUSDPerMillion
	return costUncached + costCached + costOutput
}

// modelCatalog is the two-tier declarative registry indexed by ProviderName and ModelID.
var modelCatalog = map[ProviderName]map[ModelID]ModelInfo{
	ProviderVertexAI:  vertexAIModelCatalog,
	ProviderAnthropic: anthropicModelCatalog,
	ProviderOpenAI:    openAIModelCatalog,
}

