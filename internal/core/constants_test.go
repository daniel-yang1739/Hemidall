package core_test

import (
	"testing"

	"heimdall/internal/core"
)

func TestConstants_Sanity(t *testing.T) {
	if core.TokensPerMillion != 1000000.0 {
		t.Errorf("TokensPerMillion = %v, want 1000000.0", core.TokensPerMillion)
	}
	if core.DefaultUSDtoTWDExchangeRate != 32.0 {
		t.Errorf("DefaultUSDtoTWDExchangeRate = %v, want 32.0", core.DefaultUSDtoTWDExchangeRate)
	}
	if core.DefaultDailyQuotaRPDPro != 5000 {
		t.Errorf("DefaultDailyQuotaRPDPro = %v, want 5000", core.DefaultDailyQuotaRPDPro)
	}
	if core.CacheHitRateThresholdHit != 80.0 {
		t.Errorf("CacheHitRateThresholdHit = %v, want 80.0", core.CacheHitRateThresholdHit)
	}
	if core.CacheHitRateThresholdPartial != 0.1 {
		t.Errorf("CacheHitRateThresholdPartial = %v, want 0.1", core.CacheHitRateThresholdPartial)
	}
	if core.DefaultFallbackSystemTokens != 3806 {
		t.Errorf("DefaultFallbackSystemTokens = %v, want 3806", core.DefaultFallbackSystemTokens)
	}
	if core.DefaultFallbackToolsDefTokens != 1377 {
		t.Errorf("DefaultFallbackToolsDefTokens = %v, want 1377", core.DefaultFallbackToolsDefTokens)
	}
	if core.DefaultFallbackAgentWindow != 256000 {
		t.Errorf("DefaultFallbackAgentWindow = %v, want 256000", core.DefaultFallbackAgentWindow)
	}
}
