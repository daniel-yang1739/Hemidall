package antigravity_test

import (
	"os"
	"path/filepath"
	"testing"

	"heimdall/internal/adapters/antigravity"
)

func TestLoadAntigravityHostConfig_CompleteSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "settings.json")

	content := `{
		"model": {
			"name": "gemini-3.7-flash",
			"compressionThreshold": 0.6
		},
		"contextManagement": {
			"historyWindow": {
				"maxTokens": 180000,
				"retainedTokens": 50000
			}
		},
		"modelConfigs": {
			"customOverrides": [
				{
					"model": "gemini-2.5-pro",
					"contextManagement": {
						"historyWindow": {
							"maxTokens": 600000
						}
					}
				}
			]
		},
		"exchange_rate": 32.5,
		"daily_quota_rpd": 10000
	}`

	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write mock settings: %v", err)
	}

	cfg := antigravity.LoadAntigravityHostConfig(configFile)
	if cfg.GlobalContextWindow != 180000 {
		t.Errorf("GlobalContextWindow = %d, want 180000", cfg.GlobalContextWindow)
	}
	if cfg.CompressionThreshold != 0.6 {
		t.Errorf("CompressionThreshold = %f, want 0.6", cfg.CompressionThreshold)
	}
	if cfg.ExchangeRate != 32.5 {
		t.Errorf("ExchangeRate = %f, want 32.5", cfg.ExchangeRate)
	}
	if cfg.DailyQuotaRPD != 10000 {
		t.Errorf("DailyQuotaRPD = %d, want 10000", cfg.DailyQuotaRPD)
	}
	proOverride, exists := cfg.ModelOverrides["gemini-2.5-pro"]
	if !exists {
		t.Fatalf("Expected gemini-2.5-pro model override to exist")
	}
	if proOverride.ContextWindow != 600000 {
		t.Errorf("gemini-2.5-pro ContextWindow = %d, want 600000", proOverride.ContextWindow)
	}
}

func TestLoadAntigravityHostConfig_MissingFile_Defaults(t *testing.T) {
	cfg := antigravity.LoadAntigravityHostConfig("/non/existent/path/settings.json")
	if cfg.GlobalContextWindow != 256000 {
		t.Errorf("GlobalContextWindow = %d, want 256000 (Default fallback)", cfg.GlobalContextWindow)
	}
	if cfg.ExchangeRate != 32.0 {
		t.Errorf("ExchangeRate = %f, want 32.0", cfg.ExchangeRate)
	}
	if cfg.DailyQuotaRPD != 5000 {
		t.Errorf("DailyQuotaRPD = %d, want 5000", cfg.DailyQuotaRPD)
	}
}
