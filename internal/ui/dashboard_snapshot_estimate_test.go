package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"heimdall/internal/core"
)

const (
	dashboardTestWidth          = 160
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

func TestDashboardSnapshotEstimate_Pos_RendersFiveVisibleDimensions(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.history = []core.UnifiedAgentEvent{{StepIndex: dashboardSnapshotStep, Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference}}
	model.dashboardIdx = 0
	model.playbackEstimateHistoryCount = len(model.history)
	model.playbackEstimateRevision = dashboardPlaybackRevision
	model.playbackEstimateCachedRevision = dashboardPlaybackRevision
	model.playbackContextEstimates = []core.VisibleContextEvidenceEstimate{{
		Available:         true,
		SourceKind:        core.ContextEvidenceTranscriptFallback,
		SelectedStepIndex: dashboardSnapshotStep,
		ToolBufferTokens:  dashboardBufferTokens,
		InboundTokens:     dashboardInboundTokens,
		TotalTokens:       dashboardBufferTokens + dashboardInboundTokens,
	}}
	model.contextEstimate = core.VisibleContextEvidenceEstimate{
		Available:                  true,
		SourceKind:                 core.ContextEvidencePersistedSnapshot,
		SnapshotGenIndex:           1104,
		SnapshotGeneratedStepIndex: dashboardSnapshotStep,
		HasSnapshotGeneratedStep:   true,
		SystemTokens:               dashboardSystemTokens,
		ToolsTokens:                dashboardToolsTokens,
		ToolBufferTokens:           dashboardBufferTokens,
		HistoryTokens:              dashboardHistoryTokens,
		InboundTokens:              dashboardInboundTokens,
		TotalTokens:                dashboardTotalTokens,
	}

	view := model.renderDashboardView()

	requireViewContains(t, view, "PLAYBACK CONTEXT EVIDENCE")
	requireViewContains(t, view, "selected Step #2280; snapshot idx=1104 exactly matches this generated step")
	requireViewContains(t, view, "1. System [snapshot]")
	requireViewContains(t, view, "2. Tools [snapshot]")
	requireViewContains(t, view, "3. Tool buffers [transcript]")
	requireViewContains(t, view, "4. History [snapshot]")
	requireViewContains(t, view, "5. Inbound [transcript]")
	requireViewContains(t, view, "Visible evidence counted locally")
	requireViewContains(t, view, "████")
}

func TestDashboardSnapshotEstimate_Neg_LabelsUnavailableSnapshot(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth

	view := model.renderDashboardView()

	requireViewContains(t, view, "Playback estimate: preparing cached transcript timeline")
	requireViewContains(t, view, "no SQLite query runs when moving the cursor")
}

func TestFormatEstimateDimension_Pos_AlignsColumnsForShortAndLongLabels(t *testing.T) {
	shortLabel := formatEstimateDimension("1. System [transcript]", dashboardSystemTokens, dashboardTotalTokens, dashboardDefaultProgressBarBlocks, ColorSecondary)
	longLabel := formatEstimateDimension("3. Tool buffers [transcript]", dashboardBufferTokens, dashboardTotalTokens, dashboardDefaultProgressBarBlocks, ColorWarning)

	requireEstimateColumnsAligned(t, shortLabel, longLabel)
}

func TestDashboardPlaybackEstimate_Pos_ChangesScopeWithSelectedEvent(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.history = []core.UnifiedAgentEvent{
		{StepIndex: dashboardPlaybackFirstStep, Type: core.StepTypeUserInput, RawContent: "first prompt"},
		{StepIndex: dashboardPlaybackSecondStep, Type: core.StepTypeRunCommand, RawContent: "tool result"},
	}
	model.playbackEstimateHistoryCount = len(model.history)
	model.playbackEstimateRevision = dashboardPlaybackRevision
	model.playbackEstimateCachedRevision = dashboardPlaybackRevision
	model.playbackContextEstimates = core.BuildPlaybackContextEvidence(model.history)
	model.dashboardIdx = 0

	firstView := model.renderDashboardView()

	model.dashboardIdx = 1
	secondView := model.renderDashboardView()

	requireViewContains(t, firstView, "selected Step #1; transcript evidence observed through this event")
	requireViewContains(t, secondView, "selected Step #2; transcript evidence observed through this event")
	requireViewDoesNotContain(t, secondView, "selected Step #1; transcript evidence observed through this event")
}

func TestDashboardAggregate_Neg_RendersUnavailableMetricsWithoutUsage(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth

	view := model.renderDashboardView()

	requireViewContains(t, view, "TOTAL PROCESSED")
	requireViewContains(t, view, "EFFECTIVE TOKENS")
	requireViewContains(t, view, "unavailable")
}

func TestDashboardAggregate_Boundary_UsesProtoDefaultForOmittedCacheField(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.history = []core.UnifiedAgentEvent{
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, HasMeteredInputTokens: true, MeteredInputTokens: dashboardComparableInputTokens, HasCachedContentTokens: true, CachedContentTokens: dashboardComparableCachedTokens}},
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, HasMeteredInputTokens: true, MeteredInputTokens: dashboardDefaultZeroInputTokens}},
	}

	view := model.renderDashboardView()

	requireViewContains(t, view, "CACHE HIT VOLUME")
	requireViewContains(t, view, "1/2 explicit")
	requireViewContains(t, view, "1 default zero")
	requireViewContains(t, view, "160 Tok")
}

func TestDashboardAggregate_Pos_ProjectsEffectiveInputOnlyForPricedModel(t *testing.T) {
	model := NewModel("session-a", false)
	model.width = dashboardTestWidth
	model.history = []core.UnifiedAgentEvent{
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardKnownModel, HasMeteredInputTokens: true, MeteredInputTokens: dashboardComparableInputTokens, HasCachedContentTokens: true, CachedContentTokens: dashboardComparableCachedTokens}},
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardUnknownModel, HasMeteredInputTokens: true, MeteredInputTokens: dashboardComparableInputTokens, HasCachedContentTokens: true, CachedContentTokens: dashboardComparableCachedTokens}},
	}

	view := model.renderDashboardView()

	requireViewContains(t, view, "Effective Input")
	requireViewContains(t, view, "@0.10x")
	requireViewContains(t, view, "weighted equivalent input")
	requireRenderedLinesWithinWidth(t, view, dashboardTestWidth)
}

func TestDashboardAggregate_Pos_ShowsCompleteEffectiveAndSavedTotals(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardKnownModel, HasMeteredInputTokens: true, MeteredInputTokens: dashboardGeminiMeteredInputTokens, HasCachedContentTokens: true, CachedContentTokens: dashboardGeminiCachedContentTokens}},
		{SessionID: "session-a", Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Usage: core.PersistedUsageObservation{Available: true, ModelName: dashboardClaudeModel, HasMeteredInputTokens: true, MeteredInputTokens: dashboardClaudeMeteredInputTokens, HasCachedContentTokens: true, CachedContentTokens: dashboardClaudeCachedContentTokens}},
	}
	metrics := core.ComputeSessionAggregateMetrics(history)

	table := renderModelBreakdownTable(metrics.ModelStats, metrics.TotalStats, dashboardTestWidth)

	requireViewContains(t, table, "Effective Input")
	requireViewContains(t, table, "Cached Saved (%)")
	requireViewContains(t, table, "TOTAL SUMMARY")
	requireViewContains(t, table, "300 (15.0%)")
	requireViewContains(t, table, "470")
	requireViewContains(t, table, "1.5k (76.5%)")
	requireViewDoesNotContain(t, table, "Cache Data")
	requireViewDoesNotContain(t, table, "Source")
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

const (
	dashboardComparableInputTokens     = 50
	dashboardComparableCachedTokens    = 60
	dashboardDefaultZeroInputTokens    = 50
	dashboardGeminiMeteredInputTokens  = 100
	dashboardGeminiCachedContentTokens = 900
	dashboardClaudeMeteredInputTokens  = 200
	dashboardClaudeCachedContentTokens = 800
	dashboardKnownModel                = "gemini-3.7-flash"
	dashboardClaudeModel               = "claude-sonnet-4-6"
	dashboardUnknownModel              = "unknown"
)
