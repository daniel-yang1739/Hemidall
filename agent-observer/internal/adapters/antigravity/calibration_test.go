package antigravity

import (
	"testing"
	"time"

	"agent-observer/internal/core"
)

func TestOfficialCalibrationAndMathConsistency(t *testing.T) {
	analyzer := core.NewPayloadAnalyzer()

	// 1. Simulate an event with official Google API telemetry attached
	officialTotal := 159043
	officialCached := 138240
	event := core.UnifiedAgentEvent{
		SessionID:  "test-verify-session",
		StepIndex:  100,
		Timestamp:  time.Now(),
		Type:       core.StepTypeModelResponse,
		RawContent: "Hello from assistant",
		Tokens: core.TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    officialTotal,
			CachedTokens:   officialCached,
			OfficialModel:  "gemini-3.7-flash-high",
		},
	}

	// 2. Process step through analyzer
	analyzer.AnalyzeStep(&event)

	// 3. Verify Equality 1: D1 + D2 + D3 + D4 + D5 == TotalTokens
	d := event.Tokens
	sumDimensions := d.SystemTokens + d.ToolsDefTokens + d.ToolResultTokens + d.HistoryTokens + d.ActiveTurnTokens
	if sumDimensions != officialTotal {
		t.Fatalf("Dimension sum mismatch! D1..D5 sum=%d, officialTotal=%d (diff=%d)",
			sumDimensions, officialTotal, sumDimensions-officialTotal)
	}

	// 4. Verify Equality 2: CachedTokens + NewTokens == TotalTokens
	sumCache := d.CachedTokens + d.NewTokens
	if sumCache != officialTotal {
		t.Fatalf("Cache sum mismatch! Cached+New=%d, officialTotal=%d (diff=%d)",
			sumCache, officialTotal, sumCache-officialTotal)
	}

	// 5. Verify Equality 3: HitRate formula
	expectedHitRate := float64(officialCached) / float64(officialTotal) * 100.0
	if d.CacheHitRate != expectedHitRate {
		t.Fatalf("HitRate mismatch! got=%.4f, expected=%.4f", d.CacheHitRate, expectedHitRate)
	}

	t.Logf("✅ 5-Dimension Sum: %d + %d + %d + %d + %d = %d",
		d.SystemTokens, d.ToolsDefTokens, d.ToolResultTokens, d.HistoryTokens, d.ActiveTurnTokens, sumDimensions)
	t.Logf("✅ Cache Sum: %d (Cached) + %d (New) = %d (Total)",
		d.CachedTokens, d.NewTokens, sumCache)
	t.Logf("✅ Cache Hit Rate: %.2f%%", d.CacheHitRate)
}
