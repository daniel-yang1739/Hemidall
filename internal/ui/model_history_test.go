package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"heimdall/internal/core"
)

func TestHistoryContentPayload_Pos_ShowsStepOutputArtifactSource(t *testing.T) {
	model := NewModel("session-a", false)
	event := core.UnifiedAgentEvent{
		Type: core.StepTypeGeneric, RawContent: "persisted output", ContentSource: core.SourceKindArtifacts,
	}

	rendered := strings.Join(model.buildContentPayloadLines(event, 80), "\n")

	requireViewContains(t, rendered, "Source: STEP OUTPUT ARTIFACT")
}

const (
	historyBatchFirstStep       = 1
	historyBatchPackagedStep    = 2
	historyBatchCloudStep       = 3
	historyBatchLatestStep      = 4
	historyBatchExpectedEntries = 4
)

func TestHistoryBatch_Pos_PreservesInspectionAndBackfillsPackaging(t *testing.T) {
	model := NewModel("session-a", false)
	model.activeView = ViewHistory
	model.history = []core.UnifiedAgentEvent{
		{SessionID: "session-a", StepIndex: historyBatchFirstStep, Summary: "first"},
		{SessionID: "session-a", StepIndex: historyBatchPackagedStep, Summary: "second"},
	}
	model.selectedIdx = historyBatchFirstStep

	updated, _ := model.Update(HistoryBatchMsg{Events: []core.UnifiedAgentEvent{
		{SessionID: "session-a", StepIndex: historyBatchCloudStep, Summary: "cloud", ConsumedStepIndices: []int{historyBatchPackagedStep}},
		{SessionID: "session-a", StepIndex: historyBatchLatestStep, Summary: "latest"},
	}})
	result := updated.(Model)

	requireHistoryEventCount(t, result, historyBatchExpectedEntries)
	requireSelectedHistoryStep(t, result, historyBatchFirstStep)
	requireHistoryPackaging(t, result, historyBatchCloudStep)
	requireHistoryLatestStep(t, result, historyBatchLatestStep)
}

func TestHistoryBatch_Neg_DropsEventsFromAnotherSession(t *testing.T) {
	model := NewModel("session-a", false)

	updated, _ := model.Update(HistoryBatchMsg{Events: []core.UnifiedAgentEvent{
		{SessionID: "session-b", StepIndex: historyBatchFirstStep, Summary: "stale"},
	}})
	result := updated.(Model)

	requireHistoryEventCount(t, result, 0)
}

func TestHistoryBatch_Pos_RefreshesVisibleSnapshotEstimateAfterHydration(t *testing.T) {
	model := NewModelWithData("session-a", false, ModelData{
		ContextPayloadBuilder: hydratedEstimatePayload,
	})

	updated, command := model.Update(HistoryBatchMsg{Events: []core.UnifiedAgentEvent{{
		SessionID:  "session-a",
		StepIndex:  historyBatchFirstStep,
		RawContent: "latest user input",
		Type:       core.StepTypeUserInput,
	}}})
	afterBatch := updated.(Model)

	result := applyScheduledHistoryBatchCommands(t, afterBatch, command)

	requireContextEstimateHistoryCount(t, result, historyBatchFirstStep)
	requireContextEstimateAvailable(t, result, true)
	requireContextEstimatePositiveInbound(t, result)
	requireDashboardReadModelCache(t, result, historyBatchFirstStep)
}

func TestDashboardReadModel_Boundary_DropsStaleRevision(t *testing.T) {
	model := NewModel("session-a", false)
	model.history = []core.UnifiedAgentEvent{{SessionID: "session-a", StepIndex: historyBatchFirstStep, Type: core.StepTypeUserInput, RawContent: "current"}}
	model.dashboardReadModelRevision = historyBatchPackagedStep
	stale := DashboardReadModelMsg{
		SessionID:    "session-a",
		HistoryCount: len(model.history),
		Revision:     historyBatchFirstStep,
		ReadModel:    core.BuildDashboardReadModelFromEvents("session-a", model.history),
	}

	updated, _ := model.Update(stale)
	result := updated.(Model)

	requireDashboardReadModelCache(t, result, 0)
}

func TestHistoryBatch_Pos_SortsPreviewBeforeChronologicalBackfill(t *testing.T) {
	model := NewModel("session-a", false)
	model.history = []core.UnifiedAgentEvent{{SessionID: "session-a", StepIndex: historyBatchLatestStep, Summary: "preview"}}

	updated, _ := model.Update(HistoryBatchMsg{Events: []core.UnifiedAgentEvent{
		{SessionID: "session-a", StepIndex: historyBatchFirstStep, Summary: "first"},
		{SessionID: "session-a", StepIndex: historyBatchPackagedStep, Summary: "second"},
	}})
	result := updated.(Model)

	requireHistoryStepAt(t, result, 0, historyBatchFirstStep)
	requireHistoryStepAt(t, result, historyBatchFirstStep, historyBatchPackagedStep)
	requireHistoryStepAt(t, result, historyBatchPackagedStep, historyBatchLatestStep)
	requireHistoryLatestStep(t, result, historyBatchLatestStep)
}

func TestHistoryBatch_Boundary_ReconcilesPackagingAfterPreviewBackfill(t *testing.T) {
	model := NewModel("session-a", false)
	model.history = []core.UnifiedAgentEvent{{
		SessionID:           "session-a",
		StepIndex:           historyBatchLatestStep,
		Summary:             "preview cloud",
		ConsumedStepIndices: []int{historyBatchPackagedStep},
	}}

	updated, _ := model.Update(HistoryBatchMsg{Events: []core.UnifiedAgentEvent{{
		SessionID: "session-a",
		StepIndex: historyBatchPackagedStep,
		Summary:   "backfilled local",
	}}})
	result := updated.(Model)

	requireHistoryStepPackaging(t, result, historyBatchPackagedStep, historyBatchLatestStep)
}

func TestHistoryBatch_Pos_PreservesEnrichmentOnSameStepStatusUpdate(t *testing.T) {
	model := NewModel("session-a", false)
	model.history = []core.UnifiedAgentEvent{{
		SessionID:           "session-a",
		StepIndex:           historyBatchLatestStep,
		Status:              "RUNNING",
		Scope:               core.ScopeCloudInference,
		PackagedInStepIdx:   historyBatchCloudStep,
		ConsumedStepIndices: []int{historyBatchPackagedStep},
		Tokens:              core.TokenBreakdown{StepDelta: historyBatchLatestStep},
		Usage:               core.PersistedUsageObservation{Available: true, HasObservedContextTokens: true, ObservedContextTokens: historyBatchLatestStep},
	}}

	updated, _ := model.Update(HistoryBatchMsg{Events: []core.UnifiedAgentEvent{{
		SessionID:  "session-a",
		StepIndex:  historyBatchLatestStep,
		Status:     "DONE",
		RawContent: "completed response",
	}}})
	result := updated.(Model)

	requireUpdatedHistoryStatus(t, result, "DONE")
	requireUpdatedHistoryScope(t, result, core.ScopeCloudInference)
	requireUpdatedHistoryTokenDelta(t, result, historyBatchLatestStep)
	requireUpdatedHistoryUsage(t, result, historyBatchLatestStep)
	requireUpdatedHistoryContent(t, result, "completed response")
}

func TestStepModelName_Neg_DoesNotInventSessionModel(t *testing.T) {
	model := NewModel("session-a", false)
	model.availableSessions = []core.SessionInfo{{SessionID: "session-a", ModelName: "Gemini 3.7 Flash (High)"}}
	event := core.UnifiedAgentEvent{SessionID: "session-a", Type: core.StepTypeModelResponse}

	requireStepModelName(t, model, event, "")
}

func TestStepModelName_Pos_UsesExactPersistedClaudeModel(t *testing.T) {
	model := NewModel("session-a", false)
	event := core.UnifiedAgentEvent{
		SessionID: "session-a",
		Type:      core.StepTypeModelResponse,
		Usage:     core.PersistedUsageObservation{ModelName: "claude-sonnet-4-6"},
	}

	requireStepModelName(t, model, event, "claude-sonnet-4-6")
}

func hydratedEstimatePayload(history []core.UnifiedAgentEvent, sessionID string) core.AgentContextPayload {
	return core.AgentContextPayload{
		SnapshotAvailable: true,
		SystemPrompt:      "system prompt",
		PersistedRecords:  []core.PersistedContextRecord{{PrimaryText: "persisted history"}},
		LatestPrompt:      history[0].RawContent,
	}
}

func requireHistoryEventCount(t *testing.T, model Model, expected int) {
	t.Helper()
	if len(model.history) != expected {
		t.Fatalf("history count: got %d, want %d", len(model.history), expected)
	}
}

func requireSelectedHistoryStep(t *testing.T, model Model, expected int) {
	t.Helper()
	event, ok := model.getSelectedEvent()
	if !ok {
		t.Fatal("selected history event is unavailable")
	}
	if event.StepIndex != expected {
		t.Fatalf("selected step: got %d, want %d", event.StepIndex, expected)
	}
}

func requireHistoryPackaging(t *testing.T, model Model, expectedPackageStep int) {
	t.Helper()
	packagedEvent := model.history[historyBatchFirstStep]
	if packagedEvent.PackagedInStepIdx != expectedPackageStep {
		t.Fatalf("step %d package: got %d, want %d", packagedEvent.StepIndex, packagedEvent.PackagedInStepIdx, expectedPackageStep)
	}
}

func requireHistoryStepAt(t *testing.T, model Model, index, expectedStep int) {
	t.Helper()
	if got := model.history[index].StepIndex; got != expectedStep {
		t.Fatalf("history step at %d: got %d, want %d", index, got, expectedStep)
	}
}

func requireHistoryStepPackaging(t *testing.T, model Model, targetStep, expectedPackageStep int) {
	t.Helper()
	for _, event := range model.history {
		if event.StepIndex == targetStep {
			if event.PackagedInStepIdx != expectedPackageStep {
				t.Fatalf("step %d package: got %d, want %d", targetStep, event.PackagedInStepIdx, expectedPackageStep)
			}
			return
		}
	}
	t.Fatalf("step %d is unavailable", targetStep)
}

func requireUpdatedHistoryStatus(t *testing.T, model Model, expected string) {
	t.Helper()
	if got := model.history[0].Status; got != expected {
		t.Fatalf("updated status: got %q, want %q", got, expected)
	}
}

func requireUpdatedHistoryScope(t *testing.T, model Model, expected core.StepScope) {
	t.Helper()
	if got := model.history[0].Scope; got != expected {
		t.Fatalf("updated scope: got %q, want %q", got, expected)
	}
}

func requireUpdatedHistoryTokenDelta(t *testing.T, model Model, expected int) {
	t.Helper()
	if got := model.history[0].Tokens.StepDelta; got != expected {
		t.Fatalf("updated token delta: got %d, want %d", got, expected)
	}
}

func requireUpdatedHistoryUsage(t *testing.T, model Model, expectedTotal int) {
	t.Helper()
	if got := model.history[0].Usage.ObservedContextTokens; got != expectedTotal {
		t.Fatalf("updated observed context: got %d, want %d", got, expectedTotal)
	}
}

func requireUpdatedHistoryContent(t *testing.T, model Model, expected string) {
	t.Helper()
	if got := model.history[0].RawContent; got != expected {
		t.Fatalf("updated content: got %q, want %q", got, expected)
	}
}

func requireStepModelName(t *testing.T, model Model, event core.UnifiedAgentEvent, expected string) {
	t.Helper()
	if got := model.getStepModelName(event); got != expected {
		t.Fatalf("step model name: got %q, want %q", got, expected)
	}
}

func requireHistoryLatestStep(t *testing.T, model Model, expected int) {
	t.Helper()
	if model.latestEvent.StepIndex != expected {
		t.Fatalf("latest step: got %d, want %d", model.latestEvent.StepIndex, expected)
	}
}

func requireContextEstimateHistoryCount(t *testing.T, model Model, expected int) {
	t.Helper()
	if model.contextEstimateHistoryCount != expected {
		t.Fatalf("estimate history count: got %d, want %d", model.contextEstimateHistoryCount, expected)
	}
}

func requireContextEstimateAvailable(t *testing.T, model Model, expected bool) {
	t.Helper()
	if model.contextEstimate.Available != expected {
		t.Fatalf("estimate availability: got %t, want %t", model.contextEstimate.Available, expected)
	}
}

func requireContextEstimatePositiveInbound(t *testing.T, model Model) {
	t.Helper()
	if model.contextEstimate.InboundTokens <= 0 {
		t.Fatalf("expected inbound token estimate, got %d", model.contextEstimate.InboundTokens)
	}
}

func requireDashboardReadModelCache(t *testing.T, model Model, wantCount int) {
	t.Helper()
	if model.dashboardReadModelHistoryCount != wantCount {
		t.Fatalf("dashboard read model history count: got %d, want %d", model.dashboardReadModelHistoryCount, wantCount)
	}
}

func applyScheduledHistoryBatchCommands(t *testing.T, model Model, command tea.Cmd) Model {
	t.Helper()
	batch, ok := command().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected a batch command")
	}
	result := model
	for _, scheduled := range batch {
		if scheduled == nil {
			continue
		}
		message := scheduled()
		updated, _ := result.Update(message)
		result = updated.(Model)
	}
	return result
}
