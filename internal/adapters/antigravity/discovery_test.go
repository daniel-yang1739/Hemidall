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
		return
	}

	// Try loading history for the most recent session
	topSession := sessions[0]
	events, err := LoadSessionHistory(topSession.SessionID, core.NewPayloadAnalyzer(nil))
	if err == nil && len(events) > 0 {
		agg := core.ComputeSessionAggregateMetrics(events)
		if agg.TotalStats.TotalProcessed < 0 {
			t.Errorf("TotalProcessed should be non-negative, got %d", agg.TotalStats.TotalProcessed)
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
		return
	}
	if latest.SessionID == "" {
		t.Fatalf("Expected non-empty SessionID from GetLatestActiveSession, got empty")
	}
	if latest.LogPath == "" {
		t.Fatalf("Expected non-empty LogPath from GetLatestActiveSession, got empty")
	}
}

