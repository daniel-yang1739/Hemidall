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

	if latest.TotalTokens <= 0 {
		t.Errorf("expected positive total tokens, got %d", latest.TotalTokens)
	}
}
