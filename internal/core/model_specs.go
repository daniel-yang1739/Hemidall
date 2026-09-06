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
	switch strings.ToLower(strings.TrimSpace(raw)) {
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
	case string(ModelClaudeSonnet46):
		return ModelClaudeSonnet46
	case string(ModelClaude37Sonnet):
		return ModelClaude37Sonnet
	case string(ModelClaude35Sonnet):
		return ModelClaude35Sonnet
	default:
		return ModelUnknown
	}
}

// modelCatalog is the two-tier declarative registry indexed by ProviderName and ModelID.
var modelCatalog = map[ProviderName]map[ModelID]ModelInfo{
	ProviderVertexAI: {
		ModelGemini25Pro: {
			Provider:            ProviderVertexAI,
			Vendor:              "Google",
			Name:                "Gemini 2.5 Pro",
			ID:                  ModelGemini25Pro,
			InputUSDPerMillion:  1.25,
			OutputUSDPerMillion: 5.00,
			CacheDiscountRate:   0.75, // 75% OFF
			SourceLabel:         "Google Gemini 2.5 Pro pricing",
			SourceURL:           "https://ai.google.dev/gemini-api/docs/pricing",
		},
		ModelGemini25Flash: {
			Provider:            ProviderVertexAI,
			Vendor:              "Google",
			Name:                "Gemini 2.5 Flash",
			ID:                  ModelGemini25Flash,
			InputUSDPerMillion:  0.075,
			OutputUSDPerMillion: 0.30,
			CacheDiscountRate:   0.75, // 75% OFF
			SourceLabel:         "Google Gemini 2.5 Flash pricing",
			SourceURL:           "https://ai.google.dev/gemini-api/docs/pricing",
		},
		ModelGemini37Flash: {
			Provider:            ProviderVertexAI,
			Vendor:              "Google",
			Name:                "Gemini 3.7 Flash",
			ID:                  ModelGemini37Flash,
			InputUSDPerMillion:  0.75,
			OutputUSDPerMillion: 3.00,
			CacheDiscountRate:   0.90, // 90% OFF
			SourceLabel:         "Google Gemini Developer API paid standard pricing",
			SourceURL:           "https://ai.google.dev/gemini-api/docs/pricing",
		},
		ModelGemini38Flash: {
			Provider:            ProviderVertexAI,
			Vendor:              "Google",
			Name:                "Gemini 3.8 Flash",
			ID:                  ModelGemini38Flash,
			InputUSDPerMillion:  0.75,
			OutputUSDPerMillion: 3.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Google Gemini Developer API paid standard pricing",
			SourceURL:           "https://ai.google.dev/gemini-api/docs/pricing",
		},
		ModelGemini37FlashHigh: {
			Provider:            ProviderVertexAI,
			Vendor:              "Google",
			Name:                "Gemini 3.7 Flash High",
			ID:                  ModelGemini37FlashHigh,
			InputUSDPerMillion:  0.75,
			OutputUSDPerMillion: 3.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Gemini Flash High pricing policy",
		},
		ModelGemini37FlashSafety: {
			Provider:            ProviderVertexAI,
			Vendor:              "Google",
			Name:                "Gemini 3.7 Flash Safety LE",
			ID:                  ModelGemini37FlashSafety,
			IsFree:              true,
			SourceLabel:         "Antigravity safety model policy",
		},
		ModelClaudeSonnet46: {
			Provider:            ProviderVertexAI,
			Vendor:              "Anthropic",
			Name:                "Claude Sonnet 4.6",
			ID:                  ModelClaudeSonnet46,
			InputUSDPerMillion:  3.30,
			OutputUSDPerMillion: 15.00,
			CacheDiscountRate:   0.90, // 90% OFF
			SourceLabel:         "Claude Sonnet standard pricing",
			SourceURL:           "https://cloud.google.com/gemini-enterprise-agent-platform/generative-ai/pricing",
		},
		ModelClaude37Sonnet: {
			Provider:            ProviderVertexAI,
			Vendor:              "Anthropic",
			Name:                "Claude 3.7 Sonnet",
			ID:                  ModelClaude37Sonnet,
			InputUSDPerMillion:  3.00,
			OutputUSDPerMillion: 15.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Claude Sonnet standard pricing",
			SourceURL:           "https://cloud.google.com/gemini-enterprise-agent-platform/generative-ai/pricing",
		},
		ModelClaude35Sonnet: {
			Provider:            ProviderVertexAI,
			Vendor:              "Anthropic",
			Name:                "Claude 3.5 Sonnet",
			ID:                  ModelClaude35Sonnet,
			InputUSDPerMillion:  3.00,
			OutputUSDPerMillion: 15.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Claude Sonnet standard pricing",
			SourceURL:           "https://cloud.google.com/gemini-enterprise-agent-platform/generative-ai/pricing",
		},
	},
	ProviderAnthropic: {
		ModelClaude37Sonnet: {
			Provider:            ProviderAnthropic,
			Vendor:              "Anthropic",
			Name:                "Claude 3.7 Sonnet",
			ID:                  ModelClaude37Sonnet,
			InputUSDPerMillion:  3.00,
			OutputUSDPerMillion: 15.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Anthropic Claude API pricing",
			SourceURL:           "https://www.anthropic.com/pricing",
		},
		ModelClaude35Sonnet: {
			Provider:            ProviderAnthropic,
			Vendor:              "Anthropic",
			Name:                "Claude 3.5 Sonnet",
			ID:                  ModelClaude35Sonnet,
			InputUSDPerMillion:  3.00,
			OutputUSDPerMillion: 15.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Anthropic Claude API pricing",
			SourceURL:           "https://www.anthropic.com/pricing",
		},
	},
	ProviderOpenAI: {},
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

