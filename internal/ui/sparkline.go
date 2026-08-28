package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var sparklineBlocks = []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline renders a single-line 8-level Unicode Block sparkline
func RenderSparkline(values []float64, maxVal float64, width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return strings.Repeat(" ", width)
	}

	// Auto-compute maxVal if non-positive
	if maxVal <= 0 {
		for _, v := range values {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal <= 0 {
		maxVal = 1.0
	}

	// Right-align: take the latest `width` values, or pad with leading spaces
	var targetVals []float64
	if len(values) > width {
		targetVals = values[len(values)-width:]
	} else {
		targetVals = values
	}

	var sb strings.Builder
	padCount := width - len(targetVals)
	for i := 0; i < padCount; i++ {
		sb.WriteRune(' ')
	}

	for _, v := range targetVals {
		if v <= 0 {
			sb.WriteRune(' ')
			continue
		}
		ratio := v / maxVal
		if ratio > 1.0 {
			ratio = 1.0
		}
		idx := int(ratio * 8)
		if idx < 1 {
			idx = 1
		}
		if idx > 8 {
			idx = 8
		}
		sb.WriteRune(sparklineBlocks[idx])
	}

	return style.Render(sb.String())
}
