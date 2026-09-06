package core

import "strings"

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
	case string(ModelClaudeFable51), "claude-fable-5.1":
		return ModelClaudeFable51
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
	case string(ModelGPT6Astra):
		return ModelGPT6Astra
	case string(ModelGPT56Sol):
		return ModelGPT56Sol
	case string(ModelGPT56Terra):
		return ModelGPT56Terra
	case string(ModelGPT56Luna):
		return ModelGPT56Luna
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

