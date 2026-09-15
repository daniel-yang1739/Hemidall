package ui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

const barSubdivisions = 8

// Fractional cells preserve small nonzero observations instead of rounding them
// to an empty bar. Exact values remain beside each chart.
func renderProportionBar(ratio float64, width int, color lipgloss.TerminalColor) string {
	if width <= 0 {
		return ""
	}
	ratio = max(0, min(1, ratio))
	cells := ratio * float64(width)
	full := int(cells)
	fraction := int((cells - float64(full)) * barSubdivisions)
	pieces := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	filled := strings.Repeat("█", full)
	used := full
	if full < width && (fraction > 0 || ratio > 0 && full == 0) {
		filled += pieces[max(1, fraction)]
		used++
	}
	return lipgloss.NewStyle().Foreground(color).Render(filled) + lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("░", width-used))
}
