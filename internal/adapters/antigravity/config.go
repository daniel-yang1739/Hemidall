package antigravity

import (
	"encoding/json"
	"os"
	"path/filepath"

	"heimdall/internal/core"
)

// AntigravitySettings matches the verified official schema from https://geminicli.com/docs/reference/configuration/
type AntigravitySettings struct {
	Model struct {
		Name                 string  `json:"name"`
		CompressionThreshold float64 `json:"compressionThreshold"`
	} `json:"model"`
	ContextManagement struct {
		HistoryWindow struct {
			MaxTokens      int `json:"maxTokens"`
			RetainedTokens int `json:"retainedTokens"`
		} `json:"historyWindow"`
	} `json:"contextManagement"`
	ModelConfigs struct {
		CustomOverrides []struct {
			Model             string `json:"model"`
			ContextManagement struct {
				HistoryWindow struct {
					MaxTokens int `json:"maxTokens"`
				} `json:"historyWindow"`
			} `json:"contextManagement"`
		} `json:"customOverrides"`
		Overrides []struct {
			Model             string `json:"model"`
			ContextManagement struct {
				HistoryWindow struct {
					MaxTokens int `json:"maxTokens"`
				} `json:"historyWindow"`
			} `json:"contextManagement"`
		} `json:"overrides"`
	} `json:"modelConfigs"`
	ExchangeRate  float64 `json:"exchange_rate"`
	DailyQuotaRPD int     `json:"daily_quota_rpd"`
}

// LoadAntigravityHostConfig discovers and loads Antigravity settings into a unified core.HostConfig.
func LoadAntigravityHostConfig(customPaths ...string) *core.HostConfig {
	cfg := core.NewDefaultHostConfig()

	// Default candidate paths in priority order
	searchPaths := customPaths
	if len(searchPaths) == 0 {
		home, _ := os.UserHomeDir()
		searchPaths = []string{
			SettingsPath(home),
			filepath.Join(".gemini", "settings.json"),
			filepath.Join(home, ".gemini", "settings.json"),
		}
	}

	for _, p := range searchPaths {
		if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
			var settings AntigravitySettings
			if err := json.Unmarshal(data, &settings); err == nil {
				applyAntigravitySettings(cfg, &settings)
				break
			}
		}
	}

	return cfg
}

func applyAntigravitySettings(cfg *core.HostConfig, settings *AntigravitySettings) {
	if settings.ContextManagement.HistoryWindow.MaxTokens > 0 {
		cfg.GlobalContextWindow = settings.ContextManagement.HistoryWindow.MaxTokens
	}
	if settings.Model.CompressionThreshold > 0 {
		cfg.CompressionThreshold = settings.Model.CompressionThreshold
	}
	if settings.ExchangeRate > 0 {
		cfg.ExchangeRate = settings.ExchangeRate
	}
	if settings.DailyQuotaRPD > 0 {
		cfg.DailyQuotaRPD = settings.DailyQuotaRPD
	}

	// Parse model overrides
	for _, o := range settings.ModelConfigs.CustomOverrides {
		if o.Model != "" && o.ContextManagement.HistoryWindow.MaxTokens > 0 {
			cfg.ModelOverrides[o.Model] = core.ModelOverrideConfig{
				ContextWindow: o.ContextManagement.HistoryWindow.MaxTokens,
			}
		}
	}
	for _, o := range settings.ModelConfigs.Overrides {
		if o.Model != "" && o.ContextManagement.HistoryWindow.MaxTokens > 0 {
			cfg.ModelOverrides[o.Model] = core.ModelOverrideConfig{
				ContextWindow: o.ContextManagement.HistoryWindow.MaxTokens,
			}
		}
	}
}
