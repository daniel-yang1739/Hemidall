package core

import "strings"

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

// NormalizeModelID normalizes a raw model string or alias into a canonical ModelID.
func NormalizeModelID(raw string) ModelID {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "anthropic/")
	s = strings.TrimPrefix(s, "google/")
	s = strings.TrimPrefix(s, "openai/")

	switch s {
	case string(ModelGemini25Pro), "gemini-pro":
		return ModelGemini25Pro
	case string(ModelGemini25Flash), "gemini-flash":
		return ModelGemini25Flash
	case string(ModelGemini37Flash):
		return ModelGemini37Flash
	case string(ModelGemini38Flash):
		return ModelGemini38Flash
	case string(ModelGemini37FlashHigh):
		return ModelGemini37FlashHigh
	case string(ModelGemini37FlashSafety):
		return ModelGemini37FlashSafety
	case string(ModelClaudeSonnet5), "claude-sonnet-5.0", "claude-5-sonnet":
		return ModelClaudeSonnet5
	case string(ModelClaudeSonnet46), "claude-sonnet-4.6", "claude-4-6-sonnet":
		return ModelClaudeSonnet46
	case string(ModelClaude37Sonnet), "claude-3.7-sonnet":
		return ModelClaude37Sonnet
	case string(ModelClaude35Sonnet), "claude-3.5-sonnet":
		return ModelClaude35Sonnet
	case string(ModelClaudeHaiku45), "claude-haiku-4.5", "claude-4-5-haiku":
		return ModelClaudeHaiku45
	case string(ModelClaudeOpus5), "claude-opus-5.0", "claude-5-opus":
		return ModelClaudeOpus5
	case string(ModelGPT5):
		return ModelGPT5
	case string(ModelGPT5Mini):
		return ModelGPT5Mini
	case string(ModelGPT4o):
		return ModelGPT4o
	case string(ModelGPT4oMini):
		return ModelGPT4oMini
	case string(ModelGPT41), "gpt-4-1":
		return ModelGPT41
	case string(ModelO1):
		return ModelO1
	case string(ModelO3):
		return ModelO3
	case string(ModelO3Mini):
		return ModelO3Mini
	case string(ModelO4Mini):
		return ModelO4Mini
	default:
		return ModelUnknown
	}
}

// ResolveModelInfo resolves a ModelInfo record for a given provider and canonical model ID.
func ResolveModelInfo(provider ProviderName, modelID ModelID) (ModelInfo, bool) {
	if provider == "" || provider == ProviderUnknown {
		// Default to ProviderVertexAI for AGY, or search across all providers
		if v, ok := modelCatalog[ProviderVertexAI][modelID]; ok {
			return v, true
		}
		for _, models := range modelCatalog {
			if v, ok := models[modelID]; ok {
				return v, true
			}
		}
		return ModelInfo{}, false
	}
	models, ok := modelCatalog[provider]
	if !ok {
		return ModelInfo{}, false
	}
	info, found := models[modelID]
	return info, found
}

// ResolveModelInfoByString resolves a ModelInfo record using provider and raw model name string.
func ResolveModelInfoByString(provider ProviderName, rawModel string) (ModelInfo, bool) {
	modelID := NormalizeModelID(rawModel)
	if modelID == ModelUnknown {
		return ModelInfo{}, false
	}
	return ResolveModelInfo(provider, modelID)
}

