package runtime

import (
	"context"
	"heimdall/internal/core"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	supervisorTestTimeout = 5 * time.Second
	supervisorTestMode    = 0o600
	supervisorTestLine    = "{\"step_index\":1,\"source\":\"USER\",\"type\":\"USER_INPUT\",\"created_at\":\"2026-09-01T01:02:03Z\",\"content\":\"original\"}\n"
)

func TestSupervisorFailedSwitchPreservesActiveSession(t *testing.T) {
	s, path, _ := fixtureSupervisor(t)
	query := s.Query()
	initial := awaitUpdate(t, s, core.MonitorHealthy)
	err := s.StartSession("missing", path+".missing", "")
	update := awaitUpdate(t, s, core.MonitorHealthy)
	if err == nil || update.SwitchError == "" {
		t.Fatal("failed load must be reported")
	}
	if s.Query() != query || update.Session.Ref.SessionID != "original" || update.Epoch != initial.Epoch {
		t.Fatal("failed switch replaced active data")
	}
	if len(update.Session.Steps) != 1 {
		t.Fatal("failed switch cleared history")
	}
}

func TestSupervisorRecoversAndDistinguishesIdleFromFailure(t *testing.T) {
	s, path, cancel := fixtureSupervisor(t)
	initial := awaitUpdate(t, s, core.MonitorHealthy)
	idle := awaitUpdate(t, s, core.MonitorHealthy)
	if !idle.Health.LastSuccess.After(initial.Health.LastSuccess) || idle.Session.Revision != initial.Session.Revision {
		t.Fatal("idle source must remain healthy without changing revision")
	}
	moveSource(t, path, path+".saved")
	failed := awaitUpdate(t, s, core.MonitorDegraded)
	if failed.Health.Error == "" || len(failed.Session.Steps) != 1 {
		t.Fatal("read failure must preserve facts and explain degradation")
	}
	moveSource(t, path+".saved", path)
	recovered := awaitUpdate(t, s, core.MonitorHealthy)
	if recovered.Health.Error != "" || !recovered.Health.LastSuccess.After(failed.Health.LastSuccess) {
		t.Fatal("source did not recover")
	}
	cancel()
	stopped := awaitUpdate(t, s, core.MonitorStopped)
	if stopped.Epoch != initial.Epoch {
		t.Fatal("shutdown changed activation identity")
	}
}

func TestSupervisorReopeningSameSessionGetsNewEpoch(t *testing.T) {
	s, path, _ := fixtureSupervisor(t)
	first := awaitUpdate(t, s, core.MonitorHealthy)
	if err := s.StartSession("original", path, ""); err != nil {
		t.Fatal(err)
	}
	second := awaitUpdate(t, s, core.MonitorHealthy)
	if second.Epoch <= first.Epoch {
		t.Fatal("same-session reload must invalidate old work")
	}
}

func fixtureSupervisor(t *testing.T) (*SessionSupervisor, string, context.CancelFunc) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(supervisorTestLine), supervisorTestMode); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s := newSessionSupervisor(ctx, dir)
	if err := s.StartSession("original", path, ""); err != nil {
		t.Fatal(err)
	}
	return s, path, cancel
}

func awaitUpdate(t *testing.T, s *SessionSupervisor, state core.MonitorState) core.SessionUpdate {
	t.Helper()
	timer := time.NewTimer(supervisorTestTimeout)
	defer timer.Stop()
	for {
		select {
		case update := <-s.Updates():
			if update.Health.State == state {
				return update
			}
		case <-timer.C:
			t.Fatalf("no %s update", state)
			return core.SessionUpdate{}
		}
	}
}

func moveSource(t *testing.T, from, to string) {
	t.Helper()
	if err := os.Rename(from, to); err != nil {
		t.Fatal(err)
	}
}
