package core_test

import (
	"testing"

	"heimdall/internal/core"
)

func TestResolveModelSpec_GeminiFlash_Default(t *testing.T) {
	spec := core.ResolveModelSpec("gemini-3.7-flash", nil)
	if spec.ModelID != "gemini-3.7-flash" {
		t.Errorf("ModelID = %s, want gemini-3.7-flash", spec.ModelID)
	}
	if spec.DefaultAgentWindow != 256000 {
		t.Errorf("DefaultAgentWindow = %d, want 256000", spec.DefaultAgentWindow)
	}
	if spec.PhysicalMaxTokens != 1048576 {
		t.Errorf("PhysicalMaxTokens = %d, want 1048576", spec.PhysicalMaxTokens)
	}
	if spec.CacheDiscountRate != 0.90 {
		t.Errorf("CacheDiscountRate = %f, want 0.90", spec.CacheDiscountRate)
	}
	if spec.Pricing.CachedPerMillionUSD != 0.075 {
		t.Errorf("Pricing.CachedPerMillionUSD = %f, want 0.075", spec.Pricing.CachedPerMillionUSD)
	}
}

func TestResolveModelSpec_GeminiPro_Default(t *testing.T) {
	spec := core.ResolveModelSpec("gemini-2.5-pro", nil)
	if spec.ModelID != "gemini-2.5-pro" {
		t.Errorf("ModelID = %s, want gemini-2.5-pro", spec.ModelID)
	}
	if spec.DefaultAgentWindow != 1048576 {
		t.Errorf("DefaultAgentWindow = %d, want 1048576", spec.DefaultAgentWindow)
	}
	if spec.PhysicalMaxTokens != 2097152 {
		t.Errorf("PhysicalMaxTokens = %d, want 2097152", spec.PhysicalMaxTokens)
	}
}

func TestResolveModelSpec_ClaudeSonnet_Default(t *testing.T) {
	spec := core.ResolveModelSpec("claude-3-7-sonnet", nil)
	if spec.ModelID != "claude-3-7-sonnet" {
		t.Errorf("ModelID = %s, want claude-3-7-sonnet", spec.ModelID)
	}
	if spec.DefaultAgentWindow != 200000 {
		t.Errorf("DefaultAgentWindow = %d, want 200000", spec.DefaultAgentWindow)
	}
}

func TestResolveModelSpec_UnknownModel_Fallback(t *testing.T) {
	spec := core.ResolveModelSpec("some-custom-local-model", nil)
	if spec.ModelID != "unknown-model" {
		t.Errorf("ModelID = %s, want unknown-model", spec.ModelID)
	}
	if spec.DefaultAgentWindow != core.DefaultFallbackAgentWindow {
		t.Errorf("DefaultAgentWindow = %d, want %d", spec.DefaultAgentWindow, core.DefaultFallbackAgentWindow)
	}
}

func TestResolveModelSpec_GlobalHostConfigOverride(t *testing.T) {
	hostCfg := &core.HostConfig{
		GlobalContextWindow: 150000,
	}
	spec := core.ResolveModelSpec("gemini-3.7-flash", hostCfg)
	if spec.DefaultAgentWindow != 150000 {
		t.Errorf("DefaultAgentWindow = %d, want 150000 (Global override)", spec.DefaultAgentWindow)
	}
}

func TestResolveModelSpec_ByModelOverride_PrecedenceOverGlobal(t *testing.T) {
	hostCfg := &core.HostConfig{
		GlobalContextWindow: 150000,
		ModelOverrides: map[string]core.ModelOverrideConfig{
			"gemini-2.5-pro": {
				ContextWindow: 500000,
			},
		},
	}

	specPro := core.ResolveModelSpec("gemini-2.5-pro", hostCfg)
	if specPro.DefaultAgentWindow != 500000 {
		t.Errorf("DefaultAgentWindow = %d, want 500000 (By-model override)", specPro.DefaultAgentWindow)
	}

	specFlash := core.ResolveModelSpec("gemini-3.7-flash", hostCfg)
	if specFlash.DefaultAgentWindow != 150000 {
		t.Errorf("DefaultAgentWindow = %d, want 150000 (Global fallback)", specFlash.DefaultAgentWindow)
	}
}
