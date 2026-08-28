package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// ==============================================================================
// 6. SPARKLINE MATH & SCALING ALGORITHMS (3 Positive + 3 Negative)
// ==============================================================================

func TestSparkline_Pos_EightLevelUnicodeBlocksMapping(t *testing.T) {
	// Values from 0 to 100
	vals := []float64{12.5, 25.0, 37.5, 50.0, 62.5, 75.0, 87.5, 100.0}
	res := RenderSparkline(vals, 100.0, 8, lipgloss.NewStyle())

	if len([]rune(res)) != 8 {
		t.Fatalf("Expected 8 runes output, got %d (raw: %q)", len([]rune(res)), res)
	}

	// Verify that height increases monotonically
	runes := []rune(res)
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] > runes[i+1] {
			t.Errorf("Expected monotonic increase in sparkline block: %c <= %c", runes[i], runes[i+1])
		}
	}
}

func TestSparkline_Pos_RightAlignmentWithPadding(t *testing.T) {
	vals := []float64{50.0, 100.0}
	width := 6
	res := RenderSparkline(vals, 100.0, width, lipgloss.NewStyle())

	runes := []rune(res)
	if len(runes) != width {
		t.Fatalf("Expected width %d, got %d", width, len(runes))
	}
	// First 4 runes must be leading spaces
	for i := 0; i < 4; i++ {
		if runes[i] != ' ' {
			t.Errorf("Expected leading space at index %d, got %c", i, runes[i])
		}
	}
}

func TestSparkline_Pos_LongArrayTruncation(t *testing.T) {
	vals := make([]float64, 100)
	for i := 0; i < 100; i++ {
		vals[i] = float64(i)
	}
	width := 10
	res := RenderSparkline(vals, 100.0, width, lipgloss.NewStyle())

	runes := []rune(res)
	if len(runes) != width {
		t.Fatalf("Expected width %d, got %d", width, len(runes))
	}
}

func TestSparkline_Neg_EmptyArraySpacesOutput(t *testing.T) {
	res := RenderSparkline([]float64{}, 100.0, 10, lipgloss.NewStyle())
	if res != strings.Repeat(" ", 10) {
		t.Errorf("Expected 10 spaces on empty array, got %q", res)
	}
}

func TestSparkline_Neg_AllZeroOrFlatlineNoDivisionByZero(t *testing.T) {
	// All zeros
	resZero := RenderSparkline([]float64{0, 0, 0}, 0.0, 5, lipgloss.NewStyle())
	if len([]rune(resZero)) != 5 {
		t.Errorf("Expected width 5, got %d", len([]rune(resZero)))
	}

	// Flatline non-zero
	resFlat := RenderSparkline([]float64{50, 50, 50}, 50.0, 5, lipgloss.NewStyle())
	if len([]rune(resFlat)) != 5 {
		t.Errorf("Expected width 5, got %d", len([]rune(resFlat)))
	}
}

func TestSparkline_Neg_NegativeValuesClampedToZero(t *testing.T) {
	res := RenderSparkline([]float64{-10.0, -5.0}, 100.0, 4, lipgloss.NewStyle())
	runes := []rune(res)
	if len(runes) != 4 {
		t.Fatalf("Expected width 4, got %d", len(runes))
	}
	// Negative values must be rendered as empty space
	if runes[2] != ' ' || runes[3] != ' ' {
		t.Errorf("Negative values should render as spaces, got %c, %c", runes[2], runes[3])
	}
}
