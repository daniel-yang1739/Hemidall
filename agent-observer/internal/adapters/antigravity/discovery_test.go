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
	}
}
