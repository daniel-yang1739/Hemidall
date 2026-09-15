package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"heimdall/internal/core"
)

func TestTelemetryBarsRespectAvailabilityAndBounds(t *testing.T) {
	cases := []struct {
		name          string
		amount, total int
		available     bool
		want          string
		bars          bool
	}{
		{"half", 50, 100, true, "50.0%", true},
		{"observed-zero", 0, 100, true, "0.0%", true},
		{"unknown", 50, 100, false, "[observed]", false},
		{"zero-denominator", 0, 0, true, "[observed]", false},
		{"over-capacity", 120, 100, true, "120.0%", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := renderTelemetryBarRow("Context Tokens", "50 [observed]", tc.amount, tc.total, tc.available, ColorPrimary, 100)
			if !strings.Contains(view, tc.want) {
				t.Fatalf("missing %s: %s", tc.want, view)
			}
			if strings.ContainsAny(view, "█░") != tc.bars {
				t.Fatalf("incorrect bar availability: %s", view)
			}
			if lipgloss.Width(view) > 100 {
				t.Fatal("bar exceeds viewport")
			}
		})
	}
}

func TestCloudTelemetryBarsUseIndependentDenominators(t *testing.T) {
	usage := core.PersistedUsageObservation{
		HasObservedContextTokens: true, ObservedContextTokens: 150000,
		HasContextLimit: true, ContextLimit: 250000,
		HasUncachedInputTokens: true, UncachedInputTokens: 10000,
		HasCachedInputTokens: true, CachedInputTokens: 90000,
		HasThinkingOutputTokens: true, ThinkingOutputTokens: 3000,
		HasOutputContentTokens: true, OutputContentTokens: 1000,
	}
	view := renderCloudStepInspection(core.StepInspectionReadModel{Event: core.UnifiedAgentEvent{Usage: usage}}, 120)
	cases := []struct{ label, percentage string }{
		{"Context Window Load", "60.0%"},
		{"Cached Content", "90.0%"},
		{"Uncached Prompt", "10.0%"},
		{"Output Tokens", "3.8%"},
		{"Thinking Output", "75.0%"},
		{"Content Output", "25.0%"},
	}
	barColumn := strings.Index(telemetryLine(view, "Context Tokens"), "[")
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			line := telemetryLine(view, tc.label)
			if !strings.Contains(line, tc.percentage) || !strings.ContainsAny(line, "█▏▎▍▌▋▊▉") {
				t.Fatalf("wrong denominator or missing bar: %s", line)
			}
			if strings.Index(line, "[") != barColumn {
				t.Fatalf("parent and child charts are misaligned: %s", line)
			}
		})
	}
	children := []struct{ parent, label string }{
		{"Context Tokens", "Cached Content"},
		{"Context Tokens", "Uncached Prompt"},
		{"Output Tokens", "Thinking Output"},
		{"Output Tokens", "Content Output"},
	}
	for _, child := range children {
		t.Run(child.label+" hierarchy", func(t *testing.T) {
			if !strings.HasPrefix(telemetryLine(view, child.label), "  - ") || strings.Index(view, child.parent) >= strings.Index(view, child.label) {
				t.Fatalf("child must follow its parent with a bullet and no extra indentation: %s", child.label)
			}
		})
	}
	if strings.Contains(view, "[observed]") || strings.Contains(view, "SOURCE-LABELED") {
		t.Fatal("redundant source labels returned")
	}
}

func TestTelemetryBarsFitWideAndNarrowColumns(t *testing.T) {
	cases := []struct {
		name  string
		width int
	}{{"wide", 150}, {"standard", 76}, {"compact", 56}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := renderTelemetryBarRow("Context Tokens", "151.7k [observed]", 151700, 256000, true, ColorPrimary, tc.width)
			if lipgloss.Width(view) > tc.width || !strings.Contains(view, "█") || !strings.Contains(view, "[observed]") {
				t.Fatalf("metric, bar or source label does not fit: %s", view)
			}
		})
	}
}

func telemetryLine(view, label string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, label) {
			return line
		}
	}
	return ""
}

func TestProportionBarPreservesSmallValuesAndClampsVisualExtent(t *testing.T) {
	cases := []struct {
		name  string
		ratio float64
		want  string
	}{
		{"zero", 0, "░░░░░░░░"},
		{"small-positive", 0.001, "▏░░░░░░░"},
		{"fraction", 0.1875, "█▌░░░░░░"},
		{"half", 0.5, "████░░░░"},
		{"full", 1, "████████"},
		{"overflow", 1.5, "████████"},
		{"negative", -1, "░░░░░░░░"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bar := renderProportionBar(tc.ratio, insightsBarWidth, ColorSecondary)
			if !strings.Contains(bar, tc.want) || lipgloss.Width(bar) != insightsBarWidth {
				t.Fatalf("bar=%q, want %q", bar, tc.want)
			}
		})
	}
}

func TestInsightsChartsShareLargestValueAcrossVisibleRows(t *testing.T) {
	model := NewModel("session", false)
	items := []core.RankedObservation{
		{Label: "largest", Amount: core.Measurement{Available: true, Value: 100}},
		{Label: "half", Amount: core.Measurement{Available: true, Value: 50}},
	}
	lines := model.insightTable(items, 1, 76, 4)
	if !strings.Contains(lines[2], "████████") || !strings.Contains(lines[3], "████") || !strings.Contains(lines[3], "░░░░") {
		t.Fatalf("ranking bars do not share the maximum: %v", lines)
	}
	if !strings.Contains(lines[3], "50") {
		t.Fatal("chart replaced the precise amount")
	}
}
