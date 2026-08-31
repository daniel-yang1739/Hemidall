package adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"heimdall/internal/core"
)

const fixtureTranscriptContent = "{\"step_index\":0,\"type\":\"USER_INPUT\"}\n"

func TestWatcherHub_Pos_StartSessionWithExplicitTranscript(t *testing.T) {
	hub, cancel := newFixtureWatcherHub(t)
	defer cancel()

	startErr := hub.StartSession("session-a", createFixtureTranscript(t, "session-a.jsonl"), "")

	requireHubNoError(t, startErr)
	requireHubEqual(t, "session-a", hub.ActiveSessionID())
	hub.Stop()
}

func TestWatcherHub_Pos_ReplacesActiveSession(t *testing.T) {
	hub, cancel := newFixtureWatcherHub(t)
	defer cancel()
	firstStartErr := hub.StartSession("session-a", createFixtureTranscript(t, "session-a.jsonl"), "")
	secondStartErr := hub.StartSession("session-b", createFixtureTranscript(t, "session-b.jsonl"), "")

	requireHubNoError(t, firstStartErr)
	requireHubNoError(t, secondStartErr)
	requireHubEqual(t, "session-b", hub.ActiveSessionID())
	hub.Stop()
}

func TestWatcherHub_Neg_RejectsMissingTranscript(t *testing.T) {
	hub, cancel := newFixtureWatcherHub(t)
	defer cancel()

	startErr := hub.StartSession("session-a", filepath.Join(t.TempDir(), "missing.jsonl"), "")

	requireHubError(t, startErr)
	requireHubEqual(t, "", hub.ActiveSessionID())
}

func newFixtureWatcherHub(t *testing.T) (*WatcherHub, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	eventChan := make(chan core.UnifiedAgentEvent, 1)
	return NewWatcherHub(ctx, eventChan, core.NewPayloadAnalyzer()), cancel
}

func createFixtureTranscript(t *testing.T, filename string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	writeErr := os.WriteFile(path, []byte(fixtureTranscriptContent), 0o600)
	requireHubNoError(t, writeErr)
	return path
}

func requireHubEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func requireHubNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireHubError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
