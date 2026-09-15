package ui

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"heimdall/internal/core"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	insightsTestEpoch    = 1
	insightsTestRevision = 1
	insightsTestStep     = 1
)

func TestSessionSnapshotRebuildRemovesOldHistoryAndUpdatesReport(t *testing.T) {
	model := attachTestSnapshot(t, NewModel("session", false), insightsSnapshot(insightsTestEpoch, insightsTestRevision, "first"))
	if len(model.history) != 1 || !model.reportReady {
		t.Fatal("snapshot was not prepared")
	}
	reset := insightsSnapshot(insightsTestEpoch, insightsTestRevision+1, "")
	reset.Session.Steps = nil
	model = attachTestSnapshot(t, model, reset)
	if len(model.history) != 0 || len(model.analysisReport.ToolOutputs) != 0 || model.analysisReport.Revision != reset.Session.Revision {
		t.Fatal("reset retained ghost events or stale report")
	}
	if !reflect.DeepEqual(model.analysisReport.Metrics, model.dashboardReadModel.Metrics) {
		t.Fatal("dashboard and report must share aggregates")
	}
}

func TestSessionAnalysisRejectsOldActivationEvenForSameSessionID(t *testing.T) {
	model := NewModel("session", false)
	updated, oldCommand := model.Update(SessionUpdateMsg(insightsSnapshot(insightsTestEpoch, insightsTestRevision, "old")))
	model = updated.(Model)
	updated, queued := model.Update(SessionUpdateMsg(insightsSnapshot(insightsTestEpoch+1, insightsTestRevision, "new")))
	model = updated.(Model)
	if queued != nil {
		t.Fatal("background work must coalesce")
	}
	updated, newCommand := model.Update(oldCommand())
	model = updated.(Model)
	if model.reportReady || newCommand == nil {
		t.Fatal("old activation was accepted")
	}
	updated, _ = model.Update(newCommand())
	model = updated.(Model)
	if len(model.history) != 1 || model.history[0].RawContent != "new" || model.sessionEpoch != insightsTestEpoch+1 {
		t.Fatal("new activation missing")
	}
	updated, _ = model.Update(SessionResetMsg{SessionID: "old"})
	if updated.(Model).sessionID != "session" {
		t.Fatal("legacy reset bypassed epoch isolation")
	}
}

func TestHealthHeartbeatDoesNotRecomputeAnalysis(t *testing.T) {
	update := insightsSnapshot(insightsTestEpoch, insightsTestRevision, "first")
	model := attachTestSnapshot(t, NewModel("session", false), update)
	update.Health = core.MonitorHealth{State: core.MonitorDegraded, Error: "source unavailable"}
	updated, cmd := model.Update(SessionUpdateMsg(update))
	model = updated.(Model)
	if cmd != nil || len(model.history) != 1 || model.analysisReport.Health.State != core.MonitorDegraded {
		t.Fatal("heartbeat invalidated facts or failed to update health")
	}
	model.width = 100
	if !strings.Contains(model.renderFooter(), "source unavailable") {
		t.Fatal("source failure must remain visible")
	}
}

func TestInsightsNavigationOpensRankedEvidence(t *testing.T) {
	update := insightsSnapshot(insightsTestEpoch, insightsTestRevision, "output")
	update.Session.Steps = append(update.Session.Steps, core.Step{Index: insightsTestStep + 1, Kind: string(core.StepTypeUserInput), Content: "next question"})
	model := attachTestSnapshot(t, NewModel("session", false), update)
	model.activeView = ViewInsights
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	selected, found := model.getSelectedEvent()
	if model.activeView != ViewHistory || model.focusPane != FocusDetail || !found || selected.StepIndex != insightsTestStep {
		t.Fatal("ranking did not open its evidence")
	}
	if model.historyOffset != model.selectedIdx {
		t.Fatal("evidence was selected outside the visible history list")
	}
}

func TestInsightsSelectionFollowsVisibleRowAfterRankingShrinks(t *testing.T) {
	model := attachTestSnapshot(t, NewModel("session", false), insightsSnapshot(insightsTestEpoch, insightsTestRevision, "old"))
	model.activeView, model.insightsIndex = ViewInsights, core.InsightsRankingLimit-1
	model = attachTestSnapshot(t, model, insightsSnapshot(insightsTestEpoch, insightsTestRevision+1, "new"))
	if model.insightsIndex != 0 {
		t.Fatal("selection remained outside refreshed ranking")
	}
	model.updateInsightsKey("enter")
	if model.activeView != ViewHistory {
		t.Fatal("visible selected evidence did not open")
	}
}

func TestSwitchFailureKeepsCurrentHistory(t *testing.T) {
	model := NewModel("original", false, failingSwitcher{})
	model.history = []core.UnifiedAgentEvent{{SessionID: "original", StepIndex: insightsTestStep}}
	updated, cmd := model.Update(SwitchSessionReqMsg{SessionID: "missing"})
	updated, _ = updated.(Model).Update(cmd())
	model = updated.(Model)
	if model.sessionID != "original" || len(model.history) != 1 || !strings.Contains(model.statusMessage, "load failed") {
		t.Fatal("failed switch destroyed current state or hid error")
	}
}

type failingSwitcher struct{}

func (failingSwitcher) SwitchSession(string) error { return errors.New("load failed") }

func TestReportExportIsExclusiveAndPreservesRevision(t *testing.T) {
	model := attachTestSnapshot(t, NewModel("session", false), insightsSnapshot(insightsTestEpoch, insightsTestRevision, "private output"))
	dir := t.TempDir()
	first, err := exportAnalysisReport(model.analysisReport, dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := exportAnalysisReport(model.analysisReport, dir)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || strings.Contains(string(content), "private output") || !strings.Contains(string(content), "\"revision\": 1") {
		t.Fatal("export overwrote data, leaked payload or changed revision")
	}
	_, err = exportAnalysisReport(model.analysisReport, filepath.Join(dir, "missing"))
	if err == nil {
		t.Fatal("export failure was swallowed")
	}
}

func insightsSnapshot(epoch, revision uint64, content string) core.SessionUpdate {
	return core.SessionUpdate{Epoch: epoch, Ready: true, Health: core.MonitorHealth{State: core.MonitorHealthy}, Session: core.Session{Ref: core.SessionRef{SessionID: "session"}, Revision: revision, Steps: []core.Step{{Index: insightsTestStep, Kind: string(core.StepTypeRunCommand), Content: content}}}}
}

func attachTestSnapshot(t *testing.T, model Model, update core.SessionUpdate) Model {
	t.Helper()
	updated, cmd := model.Update(SessionUpdateMsg(update))
	if cmd == nil {
		t.Fatal("expected asynchronous analysis")
	}
	updated, _ = updated.(Model).Update(cmd())
	return updated.(Model)
}

func TestInsightsResponsiveLayoutPreservesEvidenceAndActions(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
	}{
		{"wide", 120, 36},
		{"standard", 80, 28},
		{"compact", 60, 24},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := attachTestSnapshot(t, NewModel("session", false), insightsSnapshot(insightsTestEpoch, insightsTestRevision, "output"))
			model.width, model.height, model.activeView = tc.width, tc.height, ViewInsights
			body := model.renderInsightsView()
			if lipgloss.Width(body) != tc.width || lipgloss.Height(body) != tc.height-insightsFrameRows {
				t.Fatalf("layout does not fit %dx%d: got %dx%d", tc.width, tc.height, lipgloss.Width(body), lipgloss.Height(body))
			}
			if !strings.Contains(body, "SELECTED EVIDENCE") || !strings.Contains(body, "#1") {
				t.Fatal("ranked step or selected evidence was clipped")
			}
			if !strings.Contains(model.renderHeader(), "[5] Insights") || lipgloss.Width(model.renderHeader()) > tc.width {
				t.Fatal("active navigation was clipped")
			}
			if !strings.Contains(model.renderFooter(), ":report Export") || lipgloss.Width(model.renderFooter()) > tc.width {
				t.Fatal("export action was clipped")
			}
		})
	}
}

func TestInsightsOutputSummaryDistinguishesMissingFromObservedZero(t *testing.T) {
	cases := []struct {
		name      string
		available bool
		want      string
	}{
		{"missing", false, "—"},
		{"observed-zero", true, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel("session", false)
			model.analysisReport = core.SessionAnalysisReport{
				Coverage:    core.ReportCoverage{CompleteInput: 1, Generations: 1},
				Generations: []core.GenerationAnalysis{{ContentOutput: core.Measurement{Available: tc.available}}},
			}
			lines := strings.Split(model.insightSummary(120), "\n")
			// The middle summary column is the output metric, independently of input coverage.
			values := strings.Fields(lines[4])
			if len(values) < 4 || values[2] != tc.want {
				t.Fatalf("output summary: %q; want %s", lines[4], tc.want)
			}
		})
	}
}
