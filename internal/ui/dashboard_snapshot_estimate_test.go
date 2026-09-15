package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"heimdall/internal/core"
)

const (
	dashboardTestWidth          = 160
	dashboardTestHalfWidth      = 90
	dashboardSystemTokens       = 10
	dashboardToolsTokens        = 20
	dashboardBufferTokens       = 30
	dashboardHistoryTokens      = 40
	dashboardInboundTokens      = 50
	dashboardTotalTokens        = dashboardSystemTokens + dashboardToolsTokens + dashboardBufferTokens + dashboardHistoryTokens + dashboardInboundTokens
	dashboardSnapshotStep       = 2280
	dashboardPlaybackRevision   = 1
	dashboardPlaybackFirstStep  = 1
	dashboardPlaybackSecondStep = 2
)

func TestDashboardCloudStepLabelsObservedAndMissingValues(t *testing.T) {
	usage := core.PersistedUsageObservation{
		Available: true, HasObservedContextTokens: true, ObservedContextTokens: 171100,
		HasUncachedInputTokens: true, UncachedInputTokens: 15651,
		HasCachedInputTokens: true, CachedInputTokens: 155449,
		HasThinkingOutputTokens: true, ThinkingOutputTokens: 1124,
		HasOutputContentTokens: true, OutputContentTokens: 162,
	}
	view := renderCloudStepInspection(core.StepInspectionReadModel{Event: core.UnifiedAgentEvent{StepIndex: dashboardSnapshotStep, Type: core.StepTypeModelResponse, Usage: usage}}, dashboardTestWidth)
	expected := []string{"TURN TOKEN TELEMETRY", "Context Window Load    unavailable", "Context Tokens", "  - Cached Content", "  - Uncached Prompt", "Output Tokens", "  - Thinking Output", "  - Content Output", "171.1k"}
	for _, text := range expected {
		t.Run(text, func(t *testing.T) { requireViewContains(t, view, text) })
	}
	if strings.Contains(view, "[observed]") || strings.Contains(view, "100% EMPIRICAL") || strings.Contains(view, "90% OFF") || strings.Contains(view, "256k") {
		t.Fatal("unjustified telemetry or limit claim")
	}
}

func TestDashboardLocalStep_Pos_RendersEventTimeAndTriggeredCloudStep(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.height = 40
	model.history = []core.UnifiedAgentEvent{
		{StepIndex: 1, Type: core.StepTypeToolCall, Scope: core.ScopeCloudInference},
		{StepIndex: 2, Type: core.StepTypeGeneric, Scope: core.ScopeLocalExecution, ParentStepIdx: 1, RawContent: "command result output"},
	}
	model.dashboardIdx = 1
	model.dashboardReadModel = core.DashboardReadModel{
		SessionID: "session-a",
		Inspections: map[int]core.StepInspectionReadModel{
			2: {
				Kind:               core.DashboardStepLocal,
				LocalAction:        "run_cmd",
				TriggeredCloudStep: 1,
				ResultPreview:      "command result output",
			},
		},
	}
	model.dashboardReadModelHistoryCount = len(model.history)

	view := model.renderDashboardView()

	requireViewContains(t, view, "STEP #2 · LOCAL EXECUTION")
	requireViewContains(t, view, "Event / Time")
	requireViewContains(t, view, "GENERIC")
	requireViewContains(t, view, "Tool / Action")
	requireViewContains(t, view, "run_cmd")
	requireViewContains(t, view, "Triggered Cloud Step")
	requireViewContains(t, view, "#1")
	requireViewContains(t, view, "RESULT")
	requireViewContains(t, view, "command result output")
}

func TestDashboardCloudStepLeavesMissingTelemetryUnavailable(t *testing.T) {
	view := renderCloudStepInspection(core.StepInspectionReadModel{Event: core.UnifiedAgentEvent{StepIndex: dashboardSnapshotStep, RawContent: "locally visible response", Thinking: "local reasoning"}}, dashboardTestWidth)
	requireViewContains(t, view, "TURN TOKEN TELEMETRY")
	requireViewContains(t, view, "- Content Output       unavailable")
	requireViewContains(t, view, "- Thinking Output      unavailable")
	requireViewContains(t, view, "unknown")
	if strings.Contains(view, "[observed]") {
		t.Fatal("local text must not become observed telemetry")
	}
}

func TestDashboardReadModel_Neg_ShowsPreparationWithoutCachedReadModel(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.height = 40

	view := model.renderDashboardView()

	requireViewContains(t, view, "Preparing cached session metrics")
}

func TestFormatEstimateDimension_Pos_AlignsColumnsForShortAndLongLabels(t *testing.T) {
	shortLabel := formatRequestContextDimension("Tool Schemas", core.RequestContextSection{Tokens: dashboardSystemTokens, Evidence: core.ContextSectionReconstructed}, dashboardTotalTokens, dashboardTestWidth, ColorSecondary)
	longLabel := formatRequestContextDimension("Conversation Context", core.RequestContextSection{Tokens: dashboardHistoryTokens, Evidence: core.ContextSectionReconstructed}, dashboardTotalTokens, dashboardTestWidth, ColorWarning)

	requireEstimateColumnsAligned(t, shortLabel, longLabel)
}

func TestDashboardPlayback_Pos_ChangesSelectedStepPanel(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.height = 40
	model.history = []core.UnifiedAgentEvent{
		{StepIndex: dashboardPlaybackFirstStep, Type: core.StepTypeUserInput, RawContent: "first prompt"},
		{StepIndex: dashboardPlaybackSecondStep, Type: core.StepTypeRunCommand, RawContent: "tool result"},
	}
	attachDashboardReadModel(&model)
	model.dashboardIdx = 0

	firstView := model.renderDashboardView()

	model.dashboardIdx = 1
	secondView := model.renderDashboardView()

	requireViewContains(t, firstView, "USER INPUT")
	requireViewContains(t, secondView, "LOCAL EXECUTION")
	requireViewDoesNotContain(t, secondView, "USER INPUT")
}

func TestDashboardLayout_Pos_KeepsTwoFixedHeightPanels(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.height = 40
	model.history = []core.UnifiedAgentEvent{{StepIndex: dashboardPlaybackFirstStep, Type: core.StepTypeUserInput, RawContent: strings.Repeat("long input ", dashboardResultPreviewRows)}}
	attachDashboardReadModel(&model)

	view := model.renderDashboardView()

	requireDashboardPanelCount(t, view, 2)
	requireDashboardHeight(t, view, dashboardPanelOuterHeight+selectedStepPanelOuterHeight)
}

func TestDashboardAggregate_Neg_RendersUnavailableMetricsWithoutUsage(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.height = 40
	attachDashboardReadModel(&model)

	view := model.renderDashboardView()

	requireViewContains(t, view, "TOTAL INPUT")
	requireViewContains(t, view, "TOTAL OUTPUT")
	requireViewContains(t, view, "CACHE HIT RATE")
	requireViewContains(t, view, "ESTIMATED COST")
	requireViewContains(t, view, "unavailable")
}

func TestDashboardAggregate_Boundary_UsesProtoDefaultForOmittedCacheField(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.height = 40
	model.history = []core.UnifiedAgentEvent{
		{SessionID: "session-a", StepIndex: dashboardPlaybackFirstStep, Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, HasUncachedInputTokens: true, UncachedInputTokens: dashboardComparableInputTokens, HasCachedInputTokens: true, CachedInputTokens: dashboardComparableCachedTokens}},
		{SessionID: "session-a", StepIndex: dashboardPlaybackSecondStep, Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, HasUncachedInputTokens: true, UncachedInputTokens: dashboardDefaultZeroInputTokens}},
	}
	attachDashboardReadModel(&model)

	view := model.renderDashboardView()

	requireViewContains(t, view, "TOTAL INPUT")
	requireViewContains(t, view, "CACHE HIT RATE")
	requireViewContains(t, view, "160 Tok")
	requireViewContains(t, view, "37.5%")
	requireViewContains(t, view, "60 cached tokens")
}

func TestDashboardAggregate_Pos_ProjectsCostOnlyForPricedModel(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.height = 40
	model.history = []core.UnifiedAgentEvent{
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardKnownModel, HasUncachedInputTokens: true, UncachedInputTokens: dashboardComparableInputTokens, HasCachedInputTokens: true, CachedInputTokens: dashboardComparableCachedTokens}},
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardUnknownModel, HasUncachedInputTokens: true, UncachedInputTokens: dashboardComparableInputTokens, HasCachedInputTokens: true, CachedInputTokens: dashboardComparableCachedTokens}},
	}
	attachDashboardReadModel(&model)

	view := model.renderDashboardView()

	requireViewContains(t, view, "ESTIMATED COST")
	requireViewContains(t, view, "standard API rate")
	requireViewContains(t, view, "unavailable")
	requireRenderedLinesWithinWidth(t, view, dashboardTestWidth)
}

func TestDashboardAggregate_Pos_ShowsUsageAndCostTotals(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardKnownModel, HasUncachedInputTokens: true, UncachedInputTokens: dashboardGeminiUncachedInputTokens, HasCachedInputTokens: true, CachedInputTokens: dashboardGeminiCachedInputTokens, ThinkingOutputTokens: 50, OutputContentTokens: 50}},
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardClaudeModel, HasUncachedInputTokens: true, UncachedInputTokens: dashboardClaudeUncachedInputTokens, HasCachedInputTokens: true, CachedInputTokens: dashboardClaudeCachedInputTokens, ThinkingOutputTokens: 100, OutputContentTokens: 100}},
	}
	metrics := core.ComputeSessionAggregateMetrics(history)

	table := renderModelBreakdownTable(metrics.ModelStats, metrics.TotalStats, dashboardTestWidth)

	requireViewContains(t, table, "MULTI-MODEL USAGE & COST BREAKDOWN")
	requireViewDoesNotContain(t, table, "MULTI-MODEL USAGE & COST BREAKDOWN:")
	requireViewContains(t, table, "Model Name")
	requireViewContains(t, table, "Turns")
	requireViewContains(t, table, "Input")
	requireViewContains(t, table, "Cached")
	requireViewContains(t, table, "Output")
	requireViewContains(t, table, "Est. Cost")
	requireViewContains(t, table, "TOTAL SUMMARY")
	requireViewContains(t, table, "USD")
	requireViewDoesNotContain(t, table, "Effective Input")
	requireViewDoesNotContain(t, table, "Cached Saved (%)")
	requireViewDoesNotContain(t, table, "NT$")
}

func TestDashboardAggregate_Pos_HalfWidthColumns(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardKnownModel, HasUncachedInputTokens: true, UncachedInputTokens: dashboardGeminiUncachedInputTokens, HasCachedInputTokens: true, CachedInputTokens: dashboardGeminiCachedInputTokens, ThinkingOutputTokens: 50, OutputContentTokens: 50}},
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardClaudeModel, HasUncachedInputTokens: true, UncachedInputTokens: dashboardClaudeUncachedInputTokens, HasCachedInputTokens: true, CachedInputTokens: dashboardClaudeCachedInputTokens, ThinkingOutputTokens: 100, OutputContentTokens: 100}},
	}
	metrics := core.ComputeSessionAggregateMetrics(history)

	table := renderModelBreakdownTable(metrics.ModelStats, metrics.TotalStats, dashboardTestHalfWidth)

	requireViewContains(t, table, "MULTI-MODEL USAGE & COST BREAKDOWN")
	requireViewDoesNotContain(t, table, "MULTI-MODEL USAGE & COST BREAKDOWN:")
	requireViewContains(t, table, "Model Name")
	requireViewContains(t, table, "Input")
	requireViewContains(t, table, "Cached")
	requireViewContains(t, table, "Output")
	requireViewContains(t, table, "Est. Cost")
	requireViewDoesNotContain(t, table, "Turns")
	requireViewContains(t, table, "TOTAL SUMMARY")
	requireViewContains(t, table, "USD")
	requireRenderedLinesWithinWidth(t, table, dashboardTestHalfWidth)
}

func requireViewDoesNotContain(t *testing.T, view, unexpected string) {
	t.Helper()
	if strings.Contains(view, unexpected) {
		t.Fatalf("expected view not to contain %q, got: %s", unexpected, view)
	}
}

func requireRenderedLinesWithinWidth(t *testing.T, view string, width int) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if lipgloss.Width(line) > width {
			t.Fatalf("rendered line width %d exceeds limit %d: %s", lipgloss.Width(line), width, line)
		}
	}
}

func requireEstimateColumnsAligned(t *testing.T, shortLabel, longLabel string) {
	t.Helper()
	shortTokensColumn := strings.Index(shortLabel, " Tokens")
	longTokensColumn := strings.Index(longLabel, " Tokens")
	shortPercentageColumn := strings.Index(shortLabel, " (")
	longPercentageColumn := strings.Index(longLabel, " (")
	shortBarColumn := strings.Index(shortLabel, "%) [")
	longBarColumn := strings.Index(longLabel, "%) [")
	if shortTokensColumn != longTokensColumn || shortPercentageColumn != longPercentageColumn || shortBarColumn != longBarColumn {
		t.Fatalf("misaligned estimate columns: short=%q long=%q", shortLabel, longLabel)
	}
}

func requireDashboardPanelCount(t *testing.T, view string, want int) {
	t.Helper()
	if got := strings.Count(view, "╭"); got != want {
		t.Fatalf("dashboard panel count: got %d, want %d", got, want)
	}
}

func requireDashboardHeight(t *testing.T, view string, want int) {
	t.Helper()
	if got := lipgloss.Height(view); got != want {
		t.Fatalf("dashboard height: got %d, want %d", got, want)
	}
}

func attachDashboardReadModel(model *Model) {
	model.dashboardReadModel = core.BuildDashboardReadModelFromEvents(model.sessionID, model.history)
	model.dashboardReadModelHistoryCount = len(model.history)
}

const (
	dashboardComparableInputTokens     = 50
	dashboardComparableCachedTokens    = 60
	dashboardDefaultZeroInputTokens    = 50
	dashboardGeminiUncachedInputTokens = 100
	dashboardGeminiCachedInputTokens   = 900
	dashboardClaudeUncachedInputTokens = 200
	dashboardClaudeCachedInputTokens   = 800
	dashboardKnownModel                = "gemini-3.7-flash"
	dashboardClaudeModel               = "claude-sonnet-4-6"
	dashboardUnknownModel              = "unknown"
)
