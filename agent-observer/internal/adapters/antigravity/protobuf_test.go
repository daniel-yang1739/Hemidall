package antigravity

import (
	"testing"
)

func TestSQLiteTelemetryReader_RealDB(t *testing.T) {
	dbPath := "/Users/daniel_y_yang/.gemini/antigravity-cli/conversations/aa726359-08e2-4687-a15c-073a2f4a705b.db"
	reader := NewSQLiteTelemetryReader(dbPath)

	err := reader.PollLatest()
	if err != nil {
		t.Fatalf("PollLatest failed: %v", err)
	}

	latest := reader.GetLatestTelemetry()
	if latest == nil {
		t.Fatalf("expected latest telemetry, got nil")
	}

	t.Logf("Latest Gen: %d | Step: %d | Total: %d | Cached: %d | HitRate: %.2f%% | Model: %s",
		latest.GenIndex, latest.LastStepIdx, latest.TotalTokens, latest.CachedTokens, latest.CacheHitRate, latest.ModelName)

	for step := 2280; step <= 2340; step++ {
		if m := reader.GetTelemetryForStep(step); m != nil {
			t.Logf("Found Gen for Step %d: Total=%d, Cached=%d, New=%d, HitRate=%.2f%%, Model=%s",
				step, m.TotalTokens, m.CachedTokens, m.TotalTokens-m.CachedTokens, m.CacheHitRate, m.ModelName)
		}
	}
}
