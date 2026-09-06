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
		ModelClaudeSonnet5: {
			Provider:            ProviderAnthropic,
			Vendor:              "Anthropic",
			Name:                "Claude Sonnet 5",
			ID:                  ModelClaudeSonnet5,
			InputUSDPerMillion:  2.00,
			OutputUSDPerMillion: 10.00,
			CacheDiscountRate:   0.90, // 90% OFF
			SourceLabel:         "Anthropic Claude API pricing",
			SourceURL:           "https://platform.claude.com/docs/en/about-claude/pricing",
		},
		ModelClaudeSonnet46: {
			Provider:            ProviderAnthropic,
			Vendor:              "Anthropic",
			Name:                "Claude Sonnet 4.6",
			ID:                  ModelClaudeSonnet46,
			InputUSDPerMillion:  3.00,
			OutputUSDPerMillion: 15.00,
			CacheDiscountRate:   0.90, // 90% OFF
			SourceLabel:         "Anthropic Claude API pricing",
			SourceURL:           "https://platform.claude.com/docs/en/about-claude/pricing",
		},
		ModelClaude37Sonnet: {
			Provider:            ProviderAnthropic,
			Vendor:              "Anthropic",
			Name:                "Claude 3.7 Sonnet",
			ID:                  ModelClaude37Sonnet,
			InputUSDPerMillion:  3.00,
			OutputUSDPerMillion: 15.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Anthropic Claude API pricing",
			SourceURL:           "https://platform.claude.com/docs/en/about-claude/pricing",
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
			SourceURL:           "https://platform.claude.com/docs/en/about-claude/pricing",
		},
		ModelClaudeHaiku45: {
			Provider:            ProviderAnthropic,
			Vendor:              "Anthropic",
			Name:                "Claude Haiku 4.5",
			ID:                  ModelClaudeHaiku45,
			InputUSDPerMillion:  1.00,
			OutputUSDPerMillion: 5.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Anthropic Claude API pricing",
			SourceURL:           "https://platform.claude.com/docs/en/about-claude/pricing",
		},
		ModelClaudeOpus5: {
			Provider:            ProviderAnthropic,
			Vendor:              "Anthropic",
			Name:                "Claude Opus 5",
			ID:                  ModelClaudeOpus5,
			InputUSDPerMillion:  5.00,
			OutputUSDPerMillion: 25.00,
			CacheDiscountRate:   0.90,
			SourceLabel:         "Anthropic Claude API pricing",
			SourceURL:           "https://platform.claude.com/docs/en/about-claude/pricing",
		},
	},
	ProviderOpenAI: {
		ModelGPT5: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "GPT-5",
			ID:                  ModelGPT5,
			InputUSDPerMillion:  1.25,
			OutputUSDPerMillion: 10.00,
			CacheDiscountRate:   0.90, // 90% OFF ($0.125 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelGPT5Mini: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "GPT-5 mini",
			ID:                  ModelGPT5Mini,
			InputUSDPerMillion:  0.25,
			OutputUSDPerMillion: 2.00,
			CacheDiscountRate:   0.90, // 90% OFF ($0.025 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelGPT4o: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "GPT-4o",
			ID:                  ModelGPT4o,
			InputUSDPerMillion:  2.50,
			OutputUSDPerMillion: 10.00,
			CacheDiscountRate:   0.50, // 50% OFF ($1.25 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelGPT4oMini: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "GPT-4o mini",
			ID:                  ModelGPT4oMini,
			InputUSDPerMillion:  0.15,
			OutputUSDPerMillion: 0.60,
			CacheDiscountRate:   0.50, // 50% OFF ($0.075 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelGPT41: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "GPT-4.1",
			ID:                  ModelGPT41,
			InputUSDPerMillion:  2.00,
			OutputUSDPerMillion: 8.00,
			CacheDiscountRate:   0.75, // 75% OFF ($0.50 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelO1: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "o1",
			ID:                  ModelO1,
			InputUSDPerMillion:  15.00,
			OutputUSDPerMillion: 60.00,
			CacheDiscountRate:   0.50, // 50% OFF ($7.50 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelO3: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "o3",
			ID:                  ModelO3,
			InputUSDPerMillion:  2.00,
			OutputUSDPerMillion: 8.00,
			CacheDiscountRate:   0.75, // 75% OFF ($0.50 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelO3Mini: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "o3-mini",
			ID:                  ModelO3Mini,
			InputUSDPerMillion:  1.10,
			OutputUSDPerMillion: 4.40,
			CacheDiscountRate:   0.50, // 50% OFF ($0.55 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
		ModelO4Mini: {
			Provider:            ProviderOpenAI,
			Vendor:              "OpenAI",
			Name:                "o4-mini",
			ID:                  ModelO4Mini,
			InputUSDPerMillion:  1.10,
			OutputUSDPerMillion: 4.40,
			CacheDiscountRate:   0.75, // 75% OFF ($0.275 / MTok)
			SourceLabel:         "OpenAI API pricing",
			SourceURL:           "https://developers.openai.com/api/docs/pricing",
		},
	},
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

