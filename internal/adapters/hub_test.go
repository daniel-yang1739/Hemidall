package adapters

import (
	"context"
	"testing"

	"heimdall/internal/core"
)

func TestWatcherHub_Pos_InitializeAndStartSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventChan := make(chan core.UnifiedAgentEvent, 100)
	analyzer := core.NewPayloadAnalyzer()
	hub := NewWatcherHub(ctx, eventChan, analyzer)

	_ = hub.StartSession("test-session-1", core.AgentTypeAntigravity, "", "")
	activeSID := hub.ActiveSessionID()
	if activeSID != "test-session-1" {
		t.Fatalf("Expected active session 'test-session-1', got '%s'", activeSID)
	}
}

func TestWatcherHub_Pos_SwitchSessionHotReload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventChan := make(chan core.UnifiedAgentEvent, 100)
	analyzer := core.NewPayloadAnalyzer()
	hub := NewWatcherHub(ctx, eventChan, analyzer)

	_ = hub.StartSession("session-A", core.AgentTypeAntigravity, "", "")
	_ = hub.SwitchSession("session-B", core.AgentTypeAntigravity)

	activeSID := hub.ActiveSessionID()
	if activeSID != "session-B" {
		t.Fatalf("Expected active session after switch to be 'session-B', got '%s'", activeSID)
	}

	hub.Stop()
	activeAfterStop := hub.ActiveSessionID()
	if activeAfterStop != "" {
		t.Fatalf("Expected empty active session after Stop(), got '%s'", activeAfterStop)
	}
}

func TestWatcherHub_Neg_UnsupportedAgentType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventChan := make(chan core.UnifiedAgentEvent, 100)
	analyzer := core.NewPayloadAnalyzer()
	hub := NewWatcherHub(ctx, eventChan, analyzer)

	err := hub.StartSession("session-unknown", core.AgentType("unknown_engine"), "", "")
	if err == nil {
		t.Fatalf("Expected error when starting unsupported agent type, got nil")
	}
}

func TestWatcherHub_Pos_IdempotentStartSameSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventChan := make(chan core.UnifiedAgentEvent, 100)
	analyzer := core.NewPayloadAnalyzer()
	hub := NewWatcherHub(ctx, eventChan, analyzer)

	_ = hub.StartSession("session-idempotent", core.AgentTypeAntigravity, "", "")
	err := hub.StartSession("session-idempotent", core.AgentTypeAntigravity, "", "")
	if err != nil {
		t.Fatalf("Expected idempotent start on same session to succeed with nil error, got: %v", err)
	}
}
