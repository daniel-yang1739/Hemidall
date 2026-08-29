package core

import (
	"strings"
)

// PricingSpec represents official API pricing per million tokens in USD.
type PricingSpec struct {
	CachedPerMillionUSD   float64
	UncachedPerMillionUSD float64
}

// ModelSpec defines the authoritative physical hardware limits, agent default windows,
// GPU prefix cache discount rates, and pricing specifications for an AI model.
type ModelSpec struct {
	ModelID            string
	DisplayName        string
	PhysicalMaxTokens  int
	DefaultAgentWindow int
	CacheDiscountRate  float64
	Pricing            PricingSpec
}

// OfficialModelSpecs defines the built-in registry of verified official model specifications.
var OfficialModelSpecs = map[string]ModelSpec{
	"gemini-3.7-flash": {
		ModelID:            "gemini-3.7-flash",
		DisplayName:        "Gemini 3.7 Flash",
		PhysicalMaxTokens:  1048576,
		DefaultAgentWindow: 256000,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.075,
			UncachedPerMillionUSD: 0.75,
		},
	},
	"gemini-3-flash-preview": {
		ModelID:            "gemini-3-flash-preview",
		DisplayName:        "Gemini 3.0 Flash Preview",
		PhysicalMaxTokens:  1048576,
		DefaultAgentWindow: 256000,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.075,
			UncachedPerMillionUSD: 0.75,
		},
	},
	"gemini-2.5-flash": {
		ModelID:            "gemini-2.5-flash",
		DisplayName:        "Gemini 2.5 Flash",
		PhysicalMaxTokens:  1048576,
		DefaultAgentWindow: 256000,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.075,
			UncachedPerMillionUSD: 0.75,
		},
	},
	"gemini-2.5-pro": {
		ModelID:            "gemini-2.5-pro",
		DisplayName:        "Gemini 2.5 Pro",
		PhysicalMaxTokens:  2097152,
		DefaultAgentWindow: 1048576,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.125,
			UncachedPerMillionUSD: 1.25,
		},
	},
	"gemini-3-pro-preview": {
		ModelID:            "gemini-3-pro-preview",
		DisplayName:        "Gemini 3.0 Pro Preview",
		PhysicalMaxTokens:  2097152,
		DefaultAgentWindow: 1048576,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.125,
			UncachedPerMillionUSD: 1.25,
		},
	},
	"gemini-3.1-pro-preview": {
		ModelID:            "gemini-3.1-pro-preview",
		DisplayName:        "Gemini 3.1 Pro Preview",
		PhysicalMaxTokens:  2097152,
		DefaultAgentWindow: 1048576,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.125,
			UncachedPerMillionUSD: 1.25,
		},
	},
	"gemini-3.1-flash-lite": {
		ModelID:            "gemini-3.1-flash-lite",
		DisplayName:        "Gemini 3.1 Flash Lite",
		PhysicalMaxTokens:  1048576,
		DefaultAgentWindow: 128000,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.01875,
			UncachedPerMillionUSD: 0.075,
		},
	},
	"claude-3-7-sonnet": {
		ModelID:            "claude-3-7-sonnet",
		DisplayName:        "Claude 3.7 Sonnet",
		PhysicalMaxTokens:  200000,
		DefaultAgentWindow: 200000,
		CacheDiscountRate:  0.90,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.30,
			UncachedPerMillionUSD: 3.00,
		},
	},
	"deepseek-r1": {
		ModelID:            "deepseek-r1",
		DisplayName:        "DeepSeek R1",
		PhysicalMaxTokens:  128000,
		DefaultAgentWindow: 64000,
		CacheDiscountRate:  0.75,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.07,
			UncachedPerMillionUSD: 0.27,
		},
	},
	"ollama-local": {
		ModelID:            "ollama-local",
		DisplayName:        "Ollama Local",
		PhysicalMaxTokens:  32768,
		DefaultAgentWindow: 32768,
		CacheDiscountRate:  0.0,
		Pricing: PricingSpec{
			CachedPerMillionUSD:   0.0,
			UncachedPerMillionUSD: 0.0,
		},
	},
}

// DefaultFallbackModelSpec provides a safe fallback specification when a model is unrecognized.
var DefaultFallbackModelSpec = ModelSpec{
	ModelID:            "unknown-model",
	DisplayName:        "Standard LLM Agent",
	PhysicalMaxTokens:  1048576,
	DefaultAgentWindow: DefaultFallbackAgentWindow,
	CacheDiscountRate:  0.75,
	Pricing: PricingSpec{
		CachedPerMillionUSD:   0.0375,
		UncachedPerMillionUSD: 0.15,
	},
}

// ResolveModelSpec matches a raw model string against the official registry and applies
// any active user or host configuration overrides (By-Model -> Global -> Official Registry -> Fallback).
func ResolveModelSpec(modelName string, hostConfig *HostConfig) ModelSpec {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	
	// 1. Identify base official model specification
	matchedSpec := DefaultFallbackModelSpec
	if normalized != "" {
		for key, spec := range OfficialModelSpecs {
			if strings.EqualFold(key, normalized) || strings.Contains(normalized, key) {
				matchedSpec = spec
				break
			}
		}
		// Additional heuristic aliases
		if matchedSpec.ModelID == "unknown-model" {
			if strings.Contains(normalized, "flash-lite") {
				matchedSpec = OfficialModelSpecs["gemini-3.1-flash-lite"]
			} else if strings.Contains(normalized, "flash") {
				matchedSpec = OfficialModelSpecs["gemini-3.7-flash"]
			} else if strings.Contains(normalized, "pro") {
				matchedSpec = OfficialModelSpecs["gemini-2.5-pro"]
			} else if strings.Contains(normalized, "claude") || strings.Contains(normalized, "sonnet") {
				matchedSpec = OfficialModelSpecs["claude-3-7-sonnet"]
			} else if strings.Contains(normalized, "deepseek") {
				matchedSpec = OfficialModelSpecs["deepseek-r1"]
			} else if strings.Contains(normalized, "ollama") || strings.Contains(normalized, "llama") {
				matchedSpec = OfficialModelSpecs["ollama-local"]
			}
		}
	}

	// 2. Apply Host & User Configuration Overrides
	if hostConfig != nil {
		// Tier 1: By-Model Specific Override
		if len(hostConfig.ModelOverrides) > 0 {
			for pattern, override := range hostConfig.ModelOverrides {
				if strings.Contains(normalized, strings.ToLower(pattern)) || strings.Contains(strings.ToLower(matchedSpec.ModelID), strings.ToLower(pattern)) {
					if override.ContextWindow > 0 {
						matchedSpec.DefaultAgentWindow = override.ContextWindow
					}
					if override.CacheDiscountRate > 0 {
						matchedSpec.CacheDiscountRate = override.CacheDiscountRate
					}
					if override.Pricing != nil {
						matchedSpec.Pricing = *override.Pricing
					}
					return matchedSpec
				}
			}
		}

		// Tier 2: Global Host Context Limit Override
		if hostConfig.GlobalContextWindow > 0 {
			matchedSpec.DefaultAgentWindow = hostConfig.GlobalContextWindow
		}
	}

	return matchedSpec
}
