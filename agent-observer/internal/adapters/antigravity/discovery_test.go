package antigravity

import (
	"testing"
	"agent-observer/internal/core"
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
		for _, idx := range []int{100, 500, 1000, 1500, 2000, 2291, 2330, len(events) - 1} {
			if idx >= 0 && idx < len(events) {
				ev := events[idx]
				t.Logf("Step %d (Index %d): Type=%v, Total=%d, Cached=%d, New=%d, HitRate=%.2f%%, Hist=%d, Official=%v",
					ev.StepIndex, idx, ev.Type, ev.Tokens.TotalTokens, ev.Tokens.CachedTokens, ev.Tokens.NewTokens, ev.Tokens.CacheHitRate, ev.Tokens.HistoryTokens, ev.Tokens.IsOfficialData)
			}
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

