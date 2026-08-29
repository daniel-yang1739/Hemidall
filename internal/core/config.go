package core

import (
	"encoding/json"
	"os"
)

// ModelOverrideConfig specifies per-model configuration overrides.
type ModelOverrideConfig struct {
	ContextWindow     int          `json:"context_window,omitempty"`
	CacheDiscountRate float64      `json:"cache_discount_rate,omitempty"`
	Pricing           *PricingSpec `json:"pricing,omitempty"`
}

// HostConfig encapsulates resolved host configurations from Antigravity/Gemini or OpenCode settings.
type HostConfig struct {
	// GlobalContextWindow represents the user's custom agent context limit (e.g. 150000 from historyWindow.maxTokens)
	GlobalContextWindow int `json:"global_context_window,omitempty"`

	// DailyQuotaRPD represents user's daily request tier limit (default: 5000)
	DailyQuotaRPD int `json:"daily_quota_rpd,omitempty"`

	// ExchangeRate represents fiat currency conversion rate (USD to TWD / JPY, default: 32.0)
	ExchangeRate float64 `json:"exchange_rate,omitempty"`

	// CompressionThreshold represents context fraction before triggering compaction (default: 0.5)
	CompressionThreshold float64 `json:"compression_threshold,omitempty"`

	// ModelOverrides maps model identifier patterns to custom model overrides
	ModelOverrides map[string]ModelOverrideConfig `json:"model_overrides,omitempty"`
}

// NewDefaultHostConfig returns a HostConfig initialized with authoritative default values.
func NewDefaultHostConfig() *HostConfig {
	return &HostConfig{
		GlobalContextWindow:  DefaultFallbackAgentWindow,
		DailyQuotaRPD:        DefaultDailyQuotaRPDPro,
		ExchangeRate:         DefaultUSDtoTWDExchangeRate,
		CompressionThreshold: 0.5,
		ModelOverrides:       make(map[string]ModelOverrideConfig),
	}
}

// LoadHostConfigFile attempts to parse a raw JSON config file into a generic map or HostConfig.
func LoadHostConfigFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// UnmarshalHostConfig decodes JSON bytes into HostConfig.
func UnmarshalHostConfig(data []byte) (*HostConfig, error) {
	cfg := NewDefaultHostConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
