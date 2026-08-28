package antigravity

import (
	"testing"
	"heimdall/internal/core"
)

func TestDiscoverAllSessions(t *testing.T) {
	sessions, err := DiscoverAllSessions()
	if err != nil {
		t.Fatalf("DiscoverAllSessions failed: %v", err)
	}

	if len(sessions) == 0 {
		t.Log("No local sessions found on disk")
		return
	}

	t.Logf("Discovered %d sessions:", len(sessions))
	for _, s := range sessions {
		t.Logf("• [%s] ID: %s | Steps: %d | Size: %.2fMB",
			s.LastModified.Format("2006-01-02 15:04:05"), s.SessionID, s.StepCount, s.SizeMB)
	}

	// Try loading history for the most recent session
	topSession := sessions[0]
	events, err := LoadSessionHistory(topSession.SessionID, core.NewPayloadAnalyzer())
	if err != nil {
		t.Logf("LoadSessionHistory info/warning: %v", err)
	} else {
		t.Logf("✅ Successfully loaded %d historical steps for session %s", len(events), topSession.SessionID)
		agg := core.ComputeSessionAggregateMetrics(events)
		t.Logf("AGGREGATE: Processed=%d, Cached=%d, HitRate=%.2f%%, New=%d, Saved=%d (%.2f%%), Turns=%d",
			agg.TotalStats.TotalProcessed, agg.TotalStats.TotalCached, agg.TotalStats.CacheHitRate,
			agg.TotalStats.TotalNew, agg.TotalStats.TokensSaved, agg.TotalStats.SavingsPercentage, agg.TotalStats.TurnCount)
		for _, m := range agg.ModelStats {
			t.Logf("  • Model %s: Turns=%d, Processed=%d, Cached=%d (%.2f%%), Saved=%d",
				m.ModelName, m.TurnCount, m.TotalProcessed, m.TotalCached, m.CacheHitRate, m.TokensSaved)
		}
	}
}

func TestCleanModelNameAndExtraction(t *testing.T) {
	sampleLine := "The user changed setting `Model Selection` from None to Gemini 3.7 Flash (High). No need to comment on this change if the user doesn't ask about it."
	matches := modelSelectRegex.FindStringSubmatch(sampleLine)
	if len(matches) < 2 {
		t.Fatalf("Expected regex to match Model Selection in sample line, got: %v", matches)
	}
	model := CleanModelName(matches[1])
	if model != "Gemini 3.7 Flash (High)" {
		t.Fatalf("Expected 'Gemini 3.7 Flash (High)', got '%s'", model)
	}
}

func TestGetLatestActiveSession(t *testing.T) {
	latest, err := GetLatestActiveSession()
	if err != nil {
		t.Logf("GetLatestActiveSession info (no sessions on disk): %v", err)
		return
	}
	if latest.SessionID == "" {
		t.Fatalf("Expected non-empty SessionID from GetLatestActiveSession, got empty")
	}
	if latest.LogPath == "" {
		t.Fatalf("Expected non-empty LogPath from GetLatestActiveSession, got empty")
	}
	t.Logf("✅ Latest session detected: ID=%s, LogPath=%s, Steps=%d", latest.SessionID, latest.LogPath, latest.StepCount)
}

