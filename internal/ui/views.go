package ui

import (
	"fmt"
	"strings"

	"heimdall/internal/core"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	percentageScale                   = 100.0
	zeroPercentage                    = 0.0
	tokensPerThousand                 = 1_000
	tokensPerMillion                  = 1_000_000
	dashboardNarrowContentWidth       = 70
	dashboardNarrowProgressBarBlocks  = 8
	dashboardDefaultProgressBarBlocks = 15
	dashboardContextLabelWidth        = 30
	dashboardContextTokenWidth        = 8
	dashboardContextPercentageWidth   = 5
)

func formatCommas(n int) string {
	if n < 0 {
		return "-" + formatCommas(-n)
	}
	in := fmt.Sprintf("%d", n)
	if len(in) <= 3 {
		return in
	}
	var out strings.Builder
	rem := len(in) % 3
	if rem > 0 {
		out.WriteString(in[:rem])
		if len(in) > rem {
			out.WriteString(",")
		}
	}
	for i := rem; i < len(in); i += 3 {
		out.WriteString(in[i : i+3])
		if i+3 < len(in) {
			out.WriteString(",")
		}
	}
	return out.String()
}

func formatTokShort(n int) string {
	if n >= tokensPerMillion {
		return fmt.Sprintf("%.2fM", float64(n)/float64(tokensPerMillion))
	}
	if n >= tokensPerThousand {
		return fmt.Sprintf("%.1fk", float64(n)/float64(tokensPerThousand))
	}
	return fmt.Sprintf("%d", n)
}

func formatTokFloatShort(n float64) string {
	if n >= float64(tokensPerMillion) {
		return fmt.Sprintf("%.2fM", n/float64(tokensPerMillion))
	}
	if n >= float64(tokensPerThousand) {
		return fmt.Sprintf("%.1fk", n/float64(tokensPerThousand))
	}
	return fmt.Sprintf("%.0f", n)
}

func formatCacheCoverage(stats core.ModelTokenStats) string {
	if stats.TurnCount == 0 {
		return "unavailable"
	}
	return fmt.Sprintf("%d/%d comparable", stats.CachedTurnCount, stats.TurnCount)
}

func hasComparableCacheMetrics(stats core.ModelTokenStats) bool {
	return stats.CachedTurnCount > 0 && stats.ComparableTokens > 0
}

func formatCacheHitCell(stats core.ModelTokenStats) string {
	if !hasComparableCacheMetrics(stats) {
		return "unavailable"
	}
	return fmt.Sprintf("%s (%.1f%%)", formatTokShort(stats.TotalCached), stats.CacheHitRate)
}

func formatUncachedCell(stats core.ModelTokenStats) string {
	if !hasComparableCacheMetrics(stats) {
		return "unavailable"
	}
	return formatTokShort(stats.TotalNew)
}

func formatEffectiveInputCell(stats core.ModelTokenStats) string {
	projection, available := core.ProjectCacheAdjustedInput(stats)
	if !available {
		return "unavailable"
	}
	return fmt.Sprintf("%s @%.2fx", formatTokFloatShort(projection.EffectiveInputTokens), projection.CacheInputMultiplier)
}

func formatObservedContextSum(stats core.ModelTokenStats) string {
	if stats.TurnCount == 0 {
		return "unavailable"
	}
	return formatTokShort(stats.TotalProcessed) + " Tok"
}

func cacheMetricSubLabel(stats core.ModelTokenStats, hitRate bool) string {
	if !hasComparableCacheMetrics(stats) {
		return "fields unavailable"
	}
	if hitRate {
		return fmt.Sprintf("%.1f%% comparable hit", stats.CacheHitRate)
	}
	return fmt.Sprintf("%.1f%% comparable new", percentageScale-stats.CacheHitRate)
}

func cacheMetricLabel(base string, stats core.ModelTokenStats) string {
	if !hasComparableCacheMetrics(stats) {
		return base
	}
	if stats.CompleteUsage {
		return base
	}
	return base + " (PARTIAL)"
}

func formatCacheFields(usage core.PersistedUsageObservation) string {
	if !usage.HasCachedTokens {
		return "cached token field unavailable"
	}
	uncached, hasUncached := usage.UncachedTokens()
	hitRate, hasHitRate := usage.CacheHitRate()
	if !hasUncached || !hasHitRate {
		return "invalid relative to observed total"
	}
	return fmt.Sprintf("%s cached (%.1f%% hit), %s uncached", formatTokShort(usage.CachedTokens), hitRate, formatTokShort(uncached))
}

func formatEstimateDimension(name string, tokens, total, barBlocks int, color lipgloss.TerminalColor) string {
	percentage := zeroPercentage
	if total > 0 {
		percentage = float64(tokens) / float64(total) * percentageScale
	}
	return fmt.Sprintf("  %-*s %*s Tokens (%*.1f%%) [%s]", dashboardContextLabelWidth, name, dashboardContextTokenWidth, formatTokShort(tokens), dashboardContextPercentageWidth, percentage, renderVisibleContentBar(percentage, barBlocks, color))
}

func renderVisibleContentBar(percentage float64, totalBlocks int, color lipgloss.TerminalColor) string {
	filled := int((percentage / percentageScale) * float64(totalBlocks))
	if filled > totalBlocks {
		filled = totalBlocks
	}
	if filled < 0 {
		filled = 0
	}
	filledBar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	emptyBar := lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("░", totalBlocks-filled))
	return filledBar + emptyBar
}

func renderBorderlessKpiStrip(tot core.ModelTokenStats, width int) string {
	if width < 100 {
		// Responsive 2-Column Grid for Narrow / Half Width Terminals
		colW := (width - 4) / 2
		if colW < 18 {
			colW = 18
		}
		hStyle := lipgloss.NewStyle().Foreground(ColorMuted).Bold(true)

		h1 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "OBSERVED CONTEXT SUM"), colW))
		h2 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, cacheMetricLabel("CACHE HIT VOLUME", tot)), colW))
		v1 := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%s (%d Turns)", formatObservedContextSum(tot), tot.TurnCount)), colW))
		v2 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatCacheHitCell(tot)), colW))

		h3 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, cacheMetricLabel("UNCACHED INBOUND", tot)), colW))
		h4 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "CACHE DATA"), colW))
		v3 := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatUncachedCell(tot)), colW))
		v4 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatCacheCoverage(tot)), colW))

		row1 := fmt.Sprintf("  %s %s", h1, h2)
		row2 := fmt.Sprintf("  %s %s", v1, v2)
		row3 := fmt.Sprintf("  %s %s", h3, h4)
		row4 := fmt.Sprintf("  %s %s", v3, v4)

		return fmt.Sprintf("%s\n%s\n%s\n%s", truncateVisualWidth(row1, width), truncateVisualWidth(row2, width), truncateVisualWidth(row3, width), truncateVisualWidth(row4, width))
	}

	// 5-Column Grid for Wide Terminals (>= 100 cols)
	colW := (width - 4) / 5
	if colW < 18 {
		colW = 18
	}

	hStyle := lipgloss.NewStyle().Foreground(ColorMuted).Bold(true)
	subStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	h1 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "OBSERVED CONTEXT SUM"), colW))
	h2 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, cacheMetricLabel("CACHE HIT VOLUME", tot)), colW))
	h3 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, cacheMetricLabel("UNCACHED INBOUND", tot)), colW))
	h4 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "CACHE DATA"), colW))
	h5 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "OBSERVED TURNS"), colW))

	v1 := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatObservedContextSum(tot)), colW))
	v2 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatCacheHitCell(tot)), colW))
	v3 := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatUncachedCell(tot)), colW))
	v4 := lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatCacheCoverage(tot)), colW))
	v5 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%d turns", tot.TurnCount)), colW))

	s1 := subStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%d observed turns", tot.TurnCount)), colW))
	s2 := lipgloss.NewStyle().Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, cacheMetricSubLabel(tot, true)), colW))
	s3 := lipgloss.NewStyle().Foreground(ColorHighlight).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, cacheMetricSubLabel(tot, false)), colW))
	s4 := lipgloss.NewStyle().Foreground(ColorSecondary).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "schema-inferred"), colW))
	s5 := lipgloss.NewStyle().Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "persisted records"), colW))

	row1 := fmt.Sprintf("  %s %s %s %s %s", h1, h2, h3, h4, h5)
	row2 := fmt.Sprintf("  %s %s %s %s %s", v1, v2, v3, v4, v5)
	row3 := fmt.Sprintf("  %s %s %s %s %s", s1, s2, s3, s4, s5)

	return fmt.Sprintf("%s\n%s\n%s", truncateVisualWidth(row1, width), truncateVisualWidth(row2, width), truncateVisualWidth(row3, width))
}

func formatSpaceBetweenRow(cols []string, minWidths []int, leftAlign []bool, targetWidth int) string {
	k := len(cols)
	if k == 0 {
		return ""
	}
	if k == 1 {
		return cols[0]
	}

	sumMin := 0
	for _, w := range minWidths {
		sumMin += w
	}

	extra := targetWidth - sumMin
	if extra < k-1 {
		extra = k - 1 // At least 1 space per gap
	}

	baseGap := extra / (k - 1)
	remGap := extra % (k - 1)

	var sb strings.Builder
	for i := 0; i < k; i++ {
		w := minWidths[i]
		txt := cols[i]
		if leftAlign[i] {
			sb.WriteString(fmt.Sprintf("%-*s", w, truncateVisualWidth(txt, w)))
		} else {
			sb.WriteString(fmt.Sprintf("%*s", w, truncateVisualWidth(txt, w)))
		}

		if i < k-1 {
			gap := baseGap
			if i < remGap {
				gap++
			}
			sb.WriteString(strings.Repeat(" ", gap))
		}
	}
	return sb.String()
}

func renderModelBreakdownTable(models []core.ModelTokenStats, total core.ModelTokenStats, width int) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("  MULTI-MODEL PERSISTED USAGE & PRICE-EQUIVALENT PROJECTIONS:") + "\n")

	targetWidth := width - 4
	if targetWidth < 30 {
		targetWidth = 30
	}

	if width < 75 {
		// Tier 1: 4-Column Ultra-Compact Table for Narrow Laptop Splits (< 75 cols)
		minWidths := []int{14, 5, 10, 14}
		leftAligns := []bool{true, false, false, false}

		headers := []string{"Model Name", "Turns", "Observed", "Cache Data"}
		headerStr := "  " + formatSpaceBetweenRow(headers, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(headerStr) + "\n")

		for _, m := range models {
			rowCols := []string{
				truncateVisualWidth(m.ModelName, 14),
				fmt.Sprintf("%d", m.TurnCount),
				formatTokShort(m.TotalProcessed),
				formatCacheCoverage(m),
			}
			rowStr := "  " + formatSpaceBetweenRow(rowCols, minWidths, leftAligns, targetWidth)
			sb.WriteString(rowStr + "\n")
		}

		sep := "  " + strings.Repeat("─", targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(sep) + "\n")

		totCols := []string{
			"TOTAL",
			fmt.Sprintf("%d", total.TurnCount),
			formatTokShort(total.TotalProcessed),
			formatCacheCoverage(total),
		}
		totStr := "  " + formatSpaceBetweenRow(totCols, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(totStr))

		return sb.String()
	} else if width < 110 {
		// Tier 2: 5-Column Compact Table for Medium Terminals (75 <= width < 110)
		minWidths := []int{16, 5, 9, 14, 14}
		leftAligns := []bool{true, false, false, false, false}

		headers := []string{"Model Name", "Turns", "Observed", "Effective", "Coverage"}
		headerStr := "  " + formatSpaceBetweenRow(headers, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(headerStr) + "\n")

		for _, m := range models {
			rowCols := []string{
				truncateVisualWidth(m.ModelName, 16),
				fmt.Sprintf("%d", m.TurnCount),
				formatTokShort(m.TotalProcessed),
				formatEffectiveInputCell(m),
				formatCacheCoverage(m),
			}
			rowStr := "  " + formatSpaceBetweenRow(rowCols, minWidths, leftAligns, targetWidth)
			sb.WriteString(rowStr + "\n")
		}

		sep := "  " + strings.Repeat("─", targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(sep) + "\n")

		totCols := []string{
			"TOTAL SUMMARY",
			fmt.Sprintf("%d", total.TurnCount),
			formatTokShort(total.TotalProcessed),
			"per-model only",
			formatCacheCoverage(total),
		}
		totStr := "  " + formatSpaceBetweenRow(totCols, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(totStr))

		return sb.String()
	}

	// Tier 3: 8-Column Full Table for Wide Terminals (>= 110 cols)
	minWidths := []int{18, 5, 10, 15, 10, 16, 15, 8}
	leftAligns := []bool{true, false, false, false, false, false, false, false}

	headers := []string{"Model Name", "Turns", "Observed", "Cached", "Uncached", "Effective Input", "Cache Coverage", "Source"}
	headerStr := "  " + formatSpaceBetweenRow(headers, minWidths, leftAligns, targetWidth)
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(headerStr) + "\n")

	for _, m := range models {
		rowCols := []string{
			truncateVisualWidth(m.ModelName, 18),
			fmt.Sprintf("%d", m.TurnCount),
			formatTokShort(m.TotalProcessed),
			formatCacheHitCell(m),
			formatUncachedCell(m),
			formatEffectiveInputCell(m),
			formatCacheCoverage(m),
			"metadata",
		}
		rowStr := "  " + formatSpaceBetweenRow(rowCols, minWidths, leftAligns, targetWidth)
		sb.WriteString(rowStr + "\n")
	}

	sep := "  " + strings.Repeat("─", targetWidth)
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(sep) + "\n")

	totCols := []string{
		"TOTAL SUMMARY",
		fmt.Sprintf("%d", total.TurnCount),
		formatTokShort(total.TotalProcessed),
		formatCacheHitCell(total),
		formatUncachedCell(total),
		"per-model only",
		formatCacheCoverage(total),
		"metadata",
	}
	totStr := "  " + formatSpaceBetweenRow(totCols, minWidths, leftAligns, targetWidth)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(totStr))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("  Effective Input = uncached + cached × official model cache-price ratio; per-model projection, not an Antigravity invoice."))

	return sb.String()
}

func (m Model) renderDashboardView() string {
	e := m.latestEvent
	dashboardHistoryIndex := -1
	isPlayback := false
	if len(m.history) > 0 {
		idx := m.dashboardIdx
		if idx < 0 {
			idx = 0
		}
		if idx >= len(m.history) {
			idx = len(m.history) - 1
		}
		e = m.history[idx]
		dashboardHistoryIndex = idx
		if idx < len(m.history)-1 {
			isPlayback = true
		}
	}

	t := e.Tokens
	usage := e.Usage
	total := t.TotalTokens
	if usage.HasTotalTokens {
		total = usage.TotalTokens
	}

	ctxLimit := 0
	if usage.HasContextLimit {
		ctxLimit = usage.ContextLimit
	}
	var ctxUsagePct float64
	if total > 0 && ctxLimit > 0 {
		ctxUsagePct = float64(total) / float64(ctxLimit) * 100.0
	}

	var cacheBadge string
	if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
		cacheBadge = lipgloss.NewStyle().Foreground(ColorMuted).Render("[LOCAL EXECUTION]")
	} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
		cacheBadge = lipgloss.NewStyle().Foreground(ColorHighlight).Render("[USER INPUT]")
	} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
		cacheBadge = TitleStyle.Render("[COMPACTION RECORD]")
	} else {
		switch e.CacheStatus {
		case "HIT", "PARTIAL":
			hitRate, _ := usage.CacheHitRate()
			cacheBadge = BadgeSuccess.Render(fmt.Sprintf("[DECODED CACHE %.1f%%]", hitRate))
		default:
			cacheBadge = BadgeDanger.Render("[CACHE DATA UNAVAILABLE]")
		}
	}

	panelInnerWidth := m.width - 2
	if panelInnerWidth < 40 {
		panelInnerWidth = 40
	}
	contentWidth := panelInnerWidth - 2 // Account for Padding(0, 1)

	// ==================== PANEL 0: SESSION AGGREGATE & MULTI-MODEL EFFICIENCY ====================
	agg := core.ComputeSessionAggregateMetrics(m.history)
	tot := agg.TotalStats

	var p0 strings.Builder
	p0.WriteString(TitleStyle.Render("SESSION TOKEN AGGREGATES & MULTI-MODEL EFFICIENCY") + "\n")
	p0.WriteString(renderBorderlessKpiStrip(tot, contentWidth) + "\n")

	// Render Multi-Model Breakdown Table (if models exist)
	if len(agg.ModelStats) > 0 {
		p0.WriteString(renderModelBreakdownTable(agg.ModelStats, tot, contentWidth))
	}

	panel0Box := PanelStyle.Width(panelInnerWidth).Render(p0.String())

	// ==================== PANEL 1: TRACK 1: STEP TELEMETRY & CACHE STATUS ====================
	var p1 strings.Builder
	timeStr := e.Timestamp.Local().Format("2006-01-02 15:04:05")
	if e.Timestamp.IsZero() {
		timeStr = "N/A"
	}

	playbackTag := ""
	if isPlayback {
		playbackTag = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf("(PLAYBACK: Step #%d | %d of %d)", e.StepIndex, m.dashboardIdx+1, len(m.history)))
	}

	if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
		p1Title := "TRACK 1: LOCAL EXECUTION STEP (OFFLINE)"
		if e.Status == "BLOCKED" {
			p1Title = "TRACK 1: PERMISSION BOUNDARY (BLOCKED)"
		} else if e.Source == "SYSTEM" {
			p1Title = "TRACK 1: INTERNAL HARNESS BACKGROUND TASK"
		}
		toolName := m.getLocalToolName(e)
		if toolName == "" {
			toolName = string(e.Type)
		}
		packagedInfo := "No linked cloud step observed"
		if e.PackagedInStepIdx > 0 {
			packagedInfo = fmt.Sprintf("Linked to Cloud Step #%04d", e.PackagedInStepIdx)
		}

		p1.WriteString(TitleStyle.Render(p1Title) + "\n")
		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Origin / Role  : %s (%s | Step #%03d)\n", e.GetAgentRole(), toolName, e.StepIndex))
			p1.WriteString("  • Persisted Usage: unavailable for local event\n")
			p1.WriteString(fmt.Sprintf("  • Local Text     : +%s Tok (cl100k_base)\n", formatTokShort(t.StepDelta)))
			p1.WriteString(fmt.Sprintf("  • Status & Link  : %s | %s", e.Status, packagedInfo))
		} else {
			p1.WriteString(fmt.Sprintf("  • Origin / Role        : %s (Tool Action: %s | Step #%04d | Status: %s | %s)\n", e.GetAgentRole(), lipgloss.NewStyle().Bold(true).Render(toolName), e.StepIndex, e.Status, timeStr))
			p1.WriteString("  • Persisted Usage      : unavailable for local event\n")
			p1.WriteString(fmt.Sprintf("  • Local Text Estimate  : +%d Tokens (cl100k_base)\n", t.StepDelta))
			p1.WriteString(fmt.Sprintf("  • Observed Step Link   : %s", packagedInfo))
			if !isPlayback && e.ParentStepIdx > 0 {
				p1.WriteString(fmt.Sprintf("\n  • Parent Relationship  : Triggered by Step #%04d", e.ParentStepIdx))
			}
		}
	} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
		p1Title := "TRACK 1: USER INTERACTION (CLIENT PROMPT)"
		p1.WriteString(TitleStyle.Render(p1Title) + "\n")
		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Origin / Role  : HUMAN CLIENT (Step #%03d)\n", e.StepIndex))
			p1.WriteString("  • Persisted Usage: unavailable for client input\n")
			p1.WriteString(fmt.Sprintf("  • Local Text     : +%s Tok (cl100k_base)\n", formatTokShort(t.StepDelta)))
			p1.WriteString(fmt.Sprintf("  • Status & Time  : %s | %s", e.Status, timeStr))
		} else {
			p1.WriteString(fmt.Sprintf("  • Origin / Role        : HUMAN CLIENT Intent (Step #%03d | Status: %s | %s)\n", e.StepIndex, e.Status, timeStr))
			p1.WriteString("  • Persisted Usage      : unavailable for client input\n")
			p1.WriteString(fmt.Sprintf("  • Local Text Estimate  : +%d Tokens (cl100k_base)\n", t.StepDelta))
			if e.PackagedInStepIdx > 0 {
				p1.WriteString(fmt.Sprintf("  • Observed Step Link   : Linked to Cloud Step #%04d", e.PackagedInStepIdx))
			} else {
				p1.WriteString("  • Observed Step Link   : No linked cloud step observed")
			}
		}
	} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
		p1Title := "TRACK 1: SYSTEM COMPACTION (CHECKPOINT)"
		p1.WriteString(TitleStyle.Render(p1Title) + "\n")
		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Event / Role   : %s Context Compaction (Step #%03d)\n", e.GetAgentRole(), e.StepIndex))
			p1.WriteString(fmt.Sprintf("  • Summary Size   : %d Tokens (local estimate)  %s\n", t.StepDelta, cacheBadge))
			p1.WriteString(fmt.Sprintf("  • Step Delta     : +%s Tok (GC Summary)\n", formatTokShort(t.StepDelta)))
			p1.WriteString(fmt.Sprintf("  • Status & Time  : %s | %s", e.Status, timeStr))
		} else {
			p1.WriteString(fmt.Sprintf("  • Origin / Role        : %s Middleware (Sidecar Context GC | Step #%03d | Status: %s | %s)\n", e.GetAgentRole(), e.StepIndex, e.Status, timeStr))
			p1.WriteString(fmt.Sprintf("  • Summary Size         : %d Tokens (local text estimate)  %s\n", t.StepDelta, cacheBadge))
			p1.WriteString(fmt.Sprintf("  • Step Delta (Summary) : +%d Tokens (Compacted Summary Payload)\n", t.StepDelta))
			p1.WriteString("  • Interpretation       : A compaction checkpoint was observed; the omitted history is not reconstructed")
		}
	} else {
		p1Title := "TRACK 1: PERSISTED CLOUD USAGE OBSERVATION"
		if e.IsSubagent || e.GetAgentRole() == "SUBAGENT" {
			p1Title = "TRACK 1: SUBAGENT PERSISTED USAGE OBSERVATION"
		}
		p1.WriteString(TitleStyle.Render(p1Title) + "\n")

		modelName := usage.ModelName
		if modelName == "" {
			modelName = "unknown"
		}

		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Agent / Model  : [%s] %s (Step #%03d)\n",
				e.GetAgentRole(), lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(truncateVisualWidth(modelName, contentWidth-28)), e.StepIndex))
			if usage.HasTotalTokens {
				p1.WriteString(fmt.Sprintf("  • Observed Total : %s Tok %s\n", lipgloss.NewStyle().Bold(true).Render(formatTokShort(total)), cacheBadge))
				p1.WriteString(fmt.Sprintf("  • Cache Fields   : %s\n", formatCacheFields(usage)))
			} else {
				p1.WriteString("  • Observed Total : unavailable\n")
				p1.WriteString(fmt.Sprintf("  • Local Delta    : ~%s Tok (cl100k_base estimate)\n", formatTokShort(t.StepDelta)))
			}
			p1.WriteString(fmt.Sprintf("  • Status         : %s", e.Status))
		} else {
			p1.WriteString(fmt.Sprintf("  • Agent / Model        : [%s] %s  (Step #%03d | Status: %s | %s)\n",
				e.GetAgentRole(), lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(modelName), e.StepIndex, e.Status, timeStr))
			if usage.HasTotalTokens {
				if usage.HasContextLimit {
					p1.WriteString(fmt.Sprintf("  • Observed Context      : %s Tokens (%5.1f%% of observed %dk limit)  %s\n", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", total)), ctxUsagePct, ctxLimit/1_000, cacheBadge))
				} else {
					p1.WriteString(fmt.Sprintf("  • Observed Context      : %s Tokens (limit unavailable)  %s\n", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", total)), cacheBadge))
				}
				p1.WriteString(fmt.Sprintf("  • Decoded Cache Fields  : %s\n", formatCacheFields(usage)))
				p1.WriteString(fmt.Sprintf("  • Local Delta Estimate  : +%s Tokens (cl100k_base)", formatTokShort(t.StepDelta)))
			} else {
				p1.WriteString("  • Observed Context      : unavailable\n")
				p1.WriteString(fmt.Sprintf("  • Local Delta Estimate  : +%d Tokens (cl100k_base)", t.StepDelta))
			}
			if !isPlayback && len(e.ConsumedStepIndices) > 0 {
				var childStrs []string
				for _, c := range e.ConsumedStepIndices {
					childStrs = append(childStrs, fmt.Sprintf("#%04d", c))
				}
				p1.WriteString(fmt.Sprintf("\n  • Packaged Tool Inputs : Consumed Local Step %s", strings.Join(childStrs, ", ")))
			}
		}
	}

	if isPlayback {
		p1.WriteString("\n" + playbackTag)
	}

	panel1Box := PanelStyle.Width(panelInnerWidth).Render(p1.String())

	// ==================== PANEL 2: PLAYBACK CONTEXT EVIDENCE ====================
	var p2 strings.Builder
	p2Title := "TRACK 2: PLAYBACK CONTEXT EVIDENCE (LOCAL ESTIMATE; NOT REQUEST ANATOMY)"
	p2.WriteString(TitleStyle.Render(p2Title) + "\n")
	estimate := m.playbackContextEstimateFor(dashboardHistoryIndex, e)
	if !estimate.Available {
		p2.WriteString("  • Playback estimate: preparing cached transcript timeline\n")
		p2.WriteString("  • Navigation remains local: no SQLite query runs when moving the cursor")
	} else {
		p2.WriteString(formatContextEstimateScope(estimate) + "\n")
		barBlocks := dashboardDefaultProgressBarBlocks
		if contentWidth < dashboardNarrowContentWidth {
			barBlocks = dashboardNarrowProgressBarBlocks
		}
		p2.WriteString(formatEstimateDimension(contextEstimateDimensionLabel("1. System", estimate.SourceKind), estimate.SystemTokens, estimate.TotalTokens, barBlocks, ColorSecondary) + "\n")
		p2.WriteString(formatEstimateDimension(contextEstimateDimensionLabel("2. Tools", estimate.SourceKind), estimate.ToolsTokens, estimate.TotalTokens, barBlocks, ColorHighlight) + "\n")
		p2.WriteString(formatEstimateDimension("3. Tool buffers [transcript]", estimate.ToolBufferTokens, estimate.TotalTokens, barBlocks, ColorWarning) + "\n")
		p2.WriteString(formatEstimateDimension(contextEstimateDimensionLabel("4. History", estimate.SourceKind), estimate.HistoryTokens, estimate.TotalTokens, barBlocks, ColorPrimary) + "\n")
		p2.WriteString(formatEstimateDimension("5. Inbound [transcript]", estimate.InboundTokens, estimate.TotalTokens, barBlocks, ColorSuccess) + "\n")
		p2.WriteString(fmt.Sprintf("  • Visible evidence counted locally: %s Tokens (the five bars add only to this number)\n", formatTokShort(estimate.TotalTokens)))
		p2.WriteString("  • Private, non-text, and unclassified fields are not measured; Track 1 is separate stored telemetry")
	}

	panel2Box := PanelStyle.Width(panelInnerWidth).Render(p2.String())

	return lipgloss.JoinVertical(lipgloss.Left, panel0Box, panel1Box, panel2Box)
}

func formatContextEstimateScope(estimate core.VisibleContextEvidenceEstimate) string {
	if estimate.HasSnapshotGeneratedStep {
		return fmt.Sprintf("  • Scope: selected Step #%d; snapshot idx=%d exactly matches this generated step", estimate.SelectedStepIndex, estimate.SnapshotGenIndex)
	}
	if estimate.IsGenerationInputEstimate {
		return fmt.Sprintf("  • Scope: selected cloud Step #%d; transcript evidence observed before this generation", estimate.SelectedStepIndex)
	}
	return fmt.Sprintf("  • Scope: selected Step #%d; transcript evidence observed through this event", estimate.SelectedStepIndex)
}

func contextEstimateDimensionLabel(dimension string, source core.ContextEvidenceKind) string {
	if source == core.ContextEvidencePersistedSnapshot {
		return dimension + " [snapshot]"
	}
	return dimension + " [transcript]"
}

func (m Model) renderHistoryView() string {
	if m.width < 100 {
		return m.renderHistoryViewVertical()
	}
	return m.renderHistoryViewHorizontal()
}

func (m Model) renderHistoryViewVertical() string {
	bodyHeight := m.height - 2
	if bodyHeight < 8 {
		bodyHeight = 8
	}

	// Top Section: Horizontal Split (Left: Step List 38 cols, Right: Telemetry width - 38 cols)
	topContentRows := (bodyHeight - 4) * 45 / 100
	if topContentRows < 11 {
		topContentRows = 11
	}
	topBoxHeight := topContentRows + 2

	leftOuterWidth := 38
	rightOuterWidth := m.width - leftOuterWidth
	if rightOuterWidth < 20 {
		rightOuterWidth = 20
		leftOuterWidth = m.width - rightOuterWidth
	}

	topLeftInnerWidth := leftOuterWidth - 2
	topRightInnerWidth := rightOuterWidth - 2
	topLeftContentWidth := topLeftInnerWidth - 2
	topRightContentWidth := topRightInnerWidth - 2

	// 1. Top-Left Box: Step List
	filtered := m.getFilteredHistory()
	var leftLines []string

	countStr := fmt.Sprintf("%d/%d", len(filtered), len(m.history))
	if len(filtered) == len(m.history) {
		countStr = fmt.Sprintf("%d", len(m.history))
	}
	leftTitle := fmt.Sprintf("STEPS (%s)", countStr)
	if m.focusPane == FocusList {
		leftTitle = fmt.Sprintf("STEPS (%s) <", countStr)
	}

	// Line 1: Title
	leftLines = append(leftLines, truncateVisualWidth(TitleStyle.Render(leftTitle), topLeftContentWidth))

	// Line 2: Filters line
	var typeBadgeStr string
	if m.historyTypeFilter == TypeFilterAll {
		typeBadgeStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("[T:All]")
	} else {
		typeBadgeStr = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf("[T:%s]", m.historyTypeFilter))
	}

	var cacheBadgeStr string
	if m.historyCacheFilter == CacheFilterAll {
		cacheBadgeStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("[C:All]")
	} else {
		cacheBadgeStr = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf("[C:%s]", m.historyCacheFilter))
	}

	filterPrefix := lipgloss.NewStyle().Foreground(ColorLightText).Render("Filters:")
	filterLine := fmt.Sprintf("%s %s %s", filterPrefix, typeBadgeStr, cacheBadgeStr)
	leftLines = append(leftLines, truncateVisualWidth(filterLine, topLeftContentWidth))

	if m.isHistorySearching || m.historyStepQuery != "" {
		cursorChar := ""
		if m.isHistorySearching {
			cursorChar = "█"
		}
		jumpBox := fmt.Sprintf("Jump: [#%s%s]", m.historyStepQuery, lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(jumpBox, topLeftContentWidth)))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", topLeftContentWidth)))
	} else if m.historySearchErr != "" {
		errBox := lipgloss.NewStyle().Foreground(ColorDanger).Render(fmt.Sprintf("❌ %s", m.historySearchErr))
		leftLines = append(leftLines, truncateVisualWidth(errBox, topLeftContentWidth))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", topLeftContentWidth)))
	}

	if len(filtered) == 0 {
		leftLines = append(leftLines, truncateVisualWidth("  No matching steps...", topLeftContentWidth))
		leftLines = append(leftLines, truncateVisualWidth("  Press [Esc] to reset", topLeftContentWidth))
	} else {
		maxCards := m.getHistoryVisibleCards()
		endIdx := m.historyOffset + maxCards
		if endIdx > len(filtered) {
			endIdx = len(filtered)
		}

		if m.historyOffset > 0 {
			leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  ..."))
		}

		for i := m.historyOffset; i < endIdx; i++ {
			realIdx := len(filtered) - 1 - i
			cardLine1, cardLine2 := m.formatHistoryCard(filtered, realIdx, i == m.selectedIdx, topLeftContentWidth)

			leftLines = append(leftLines, cardLine1)
			if cardLine2 != "" {
				leftLines = append(leftLines, cardLine2)
			}
		}

		if endIdx < len(filtered) && len(leftLines) < topContentRows {
			leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  ..."))
		}
	}
	for len(leftLines) < topContentRows {
		leftLines = append(leftLines, "")
	}
	if len(leftLines) > topContentRows {
		leftLines = leftLines[:topContentRows]
	}

	var topLeftBox string
	if m.focusPane == FocusList {
		topLeftBox = ActivePanelStyle.Width(topLeftInnerWidth).Render(strings.Join(leftLines, "\n"))
	} else {
		topLeftBox = PanelStyle.Width(topLeftInnerWidth).Render(strings.Join(leftLines, "\n"))
	}

	// 2. Top-Right Box: Step Telemetry (Compact 4-core metrics)
	var rightLines []string
	rightLines = append(rightLines, TitleStyle.Render(truncateVisualWidth("STEP TELEMETRY", topRightContentWidth)))

	selectedEvent, hasEvent := m.getSelectedEvent()
	if hasEvent {
		telemetryLines := m.buildTelemetryPanelLines(selectedEvent, topRightContentWidth, true)
		for _, line := range telemetryLines {
			rightLines = append(rightLines, truncateVisualWidth(line, topRightContentWidth))
		}
	} else {
		rightLines = append(rightLines, truncateVisualWidth("  No step selected", topRightContentWidth))
	}
	for len(rightLines) < topContentRows {
		rightLines = append(rightLines, "")
	}
	if len(rightLines) > topContentRows {
		rightLines = rightLines[:topContentRows]
	}
	topRightBox := PanelStyle.Width(topRightInnerWidth).Render(strings.Join(rightLines, "\n"))

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topLeftBox, topRightBox)

	// 3. Bottom Box: Content Payload (Full Width)
	bottomBoxInnerWidth := m.width - 2
	bottomContentWidth := bottomBoxInnerWidth - 2
	bottomContentRows := bodyHeight - topBoxHeight - 2
	if bottomContentRows < 4 {
		bottomContentRows = 4
	}

	var bottomLines []string
	bottomTitle := "CONTENT PAYLOAD"
	if m.isVisualMode {
		start := m.visualStart
		end := m.visualCursor
		if start > end {
			start, end = end, start
		}
		bottomTitle = fmt.Sprintf("CONTENT PAYLOAD (VISUAL: %d lines | [y] Copy)", end-start+1)
	} else if m.focusPane == FocusDetail {
		bottomTitle = "CONTENT PAYLOAD < [Scroll: j/k, Ctrl+u/d, g/G]"
	}
	bottomLines = append(bottomLines, TitleStyle.Render(truncateVisualWidth(bottomTitle, bottomContentWidth)))

	if hasEvent {
		payloadLines := m.buildContentPayloadLines(selectedEvent, bottomContentWidth)
		totalInspectorLines := len(payloadLines)

		availableLines := bottomContentRows - 1
		if availableLines < 1 {
			availableLines = 1
		}

		maxScroll := totalInspectorLines - availableLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		currentScroll := m.detailScroll
		if currentScroll > maxScroll {
			currentScroll = maxScroll
		}

		endLine := currentScroll + availableLines
		if endLine > totalInspectorLines {
			endLine = totalInspectorLines
		}

		vStart := m.visualStart
		vEnd := m.visualCursor
		if vStart > vEnd {
			vStart, vEnd = vEnd, vStart
		}

		for i := currentScroll; i < endLine; i++ {
			rawLine := payloadLines[i]
			lineText := truncateVisualWidth(rawLine, bottomContentWidth)
			if m.isVisualMode && i >= vStart && i <= vEnd {
				bottomLines = append(bottomLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary).Render(lineText))
			} else {
				bottomLines = append(bottomLines, lineText)
			}
		}
	} else {
		bottomLines = append(bottomLines, truncateVisualWidth("  Select a step above to inspect details.", bottomContentWidth))
	}

	for len(bottomLines) < bottomContentRows {
		bottomLines = append(bottomLines, "")
	}
	if len(bottomLines) > bottomContentRows {
		bottomLines = bottomLines[:bottomContentRows]
	}

	var bottomBox string
	if m.isVisualMode {
		bottomBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorHighlight).Padding(0, 1).Width(bottomBoxInnerWidth).Render(strings.Join(bottomLines, "\n"))
	} else if m.focusPane == FocusDetail {
		bottomBox = ActivePanelStyle.Width(bottomBoxInnerWidth).Render(strings.Join(bottomLines, "\n"))
	} else {
		bottomBox = PanelStyle.Width(bottomBoxInnerWidth).Render(strings.Join(bottomLines, "\n"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomBox)
}

func (m Model) renderHistoryViewHorizontal() string {
	leftOuterWidth := 38
	rightOuterWidth := m.width - leftOuterWidth
	if rightOuterWidth < 30 {
		rightOuterWidth = 30
		leftOuterWidth = m.width - rightOuterWidth
	}

	listInnerWidth := leftOuterWidth - 2
	rightInnerWidth := rightOuterWidth - 2

	listContentWidth := listInnerWidth - 2
	rightContentWidth := rightInnerWidth - 2

	innerRowsLimit := m.height - 4
	if innerRowsLimit < 4 {
		innerRowsLimit = 4
	}

	// 1. Left Pane: Step List
	filtered := m.getFilteredHistory()
	var leftLines []string

	countStr := fmt.Sprintf("%d/%d", len(filtered), len(m.history))
	if len(filtered) == len(m.history) {
		countStr = fmt.Sprintf("%d", len(m.history))
	}
	leftTitle := fmt.Sprintf("STEPS (%s)", countStr)
	if m.focusPane == FocusList {
		leftTitle = fmt.Sprintf("STEPS (%s) <", countStr)
	}

	// Line 1: Title
	leftLines = append(leftLines, truncateVisualWidth(TitleStyle.Render(leftTitle), listContentWidth))

	// Line 2: Filters line
	var typeBadgeStr string
	if m.historyTypeFilter == TypeFilterAll {
		typeBadgeStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("[T:All]")
	} else {
		typeBadgeStr = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf("[T:%s]", m.historyTypeFilter))
	}

	var cacheBadgeStr string
	if m.historyCacheFilter == CacheFilterAll {
		cacheBadgeStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("[C:All]")
	} else {
		cacheBadgeStr = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf("[C:%s]", m.historyCacheFilter))
	}

	filterPrefix := lipgloss.NewStyle().Foreground(ColorLightText).Render("Filters:")
	filterLine := fmt.Sprintf("%s %s %s", filterPrefix, typeBadgeStr, cacheBadgeStr)
	leftLines = append(leftLines, truncateVisualWidth(filterLine, listContentWidth))

	if m.isHistorySearching || m.historyStepQuery != "" {
		cursorChar := ""
		if m.isHistorySearching {
			cursorChar = "█"
		}
		jumpBox := fmt.Sprintf("Jump: [#%s%s]", m.historyStepQuery, lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(jumpBox, listContentWidth)))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", listContentWidth)))
	} else if m.historySearchErr != "" {
		errBox := lipgloss.NewStyle().Foreground(ColorDanger).Render(fmt.Sprintf("❌ %s", m.historySearchErr))
		leftLines = append(leftLines, truncateVisualWidth(errBox, listContentWidth))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", listContentWidth)))
	}

	if len(filtered) == 0 {
		leftLines = append(leftLines, truncateVisualWidth("  No matching steps...", listContentWidth))
		leftLines = append(leftLines, truncateVisualWidth("  Press [Esc] to reset", listContentWidth))
	} else {
		maxCards := m.getHistoryVisibleCards()
		endIdx := m.historyOffset + maxCards
		if endIdx > len(filtered) {
			endIdx = len(filtered)
		}

		if m.historyOffset > 0 {
			leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  ..."))
		}

		for i := m.historyOffset; i < endIdx; i++ {
			realIdx := len(filtered) - 1 - i
			cardLine1, cardLine2 := m.formatHistoryCard(filtered, realIdx, i == m.selectedIdx, listContentWidth)

			leftLines = append(leftLines, cardLine1)
			if cardLine2 != "" {
				leftLines = append(leftLines, cardLine2)
			}
		}

		if endIdx < len(filtered) && len(leftLines) < innerRowsLimit {
			leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  ..."))
		}
	}
	for len(leftLines) < innerRowsLimit {
		leftLines = append(leftLines, "")
	}
	if len(leftLines) > innerRowsLimit {
		leftLines = leftLines[:innerRowsLimit]
	}

	var leftBox string
	if m.focusPane == FocusList {
		leftBox = ActivePanelStyle.Width(listInnerWidth).Render(strings.Join(leftLines, "\n"))
	} else {
		leftBox = PanelStyle.Width(listInnerWidth).Render(strings.Join(leftLines, "\n"))
	}

	// 2. Right Column (Top: Telemetry Panel, Bottom: Content Payload Panel)
	topContentRows := 6
	bottomContentRows := innerRowsLimit - topContentRows - 2
	if bottomContentRows < 4 {
		bottomContentRows = 4
	}

	selectedEvent, hasEvent := m.getSelectedEvent()

	// 2a. Right-Top: Telemetry & Metrics Panel
	var telemetryLines []string
	telemetryLines = append(telemetryLines, TitleStyle.Render(truncateVisualWidth("STEP TELEMETRY & METRICS", rightContentWidth)))
	if hasEvent {
		tLines := m.buildTelemetryPanelLines(selectedEvent, rightContentWidth, false)
		for _, line := range tLines {
			telemetryLines = append(telemetryLines, truncateVisualWidth(line, rightContentWidth))
		}
	} else {
		telemetryLines = append(telemetryLines, truncateVisualWidth("  No step selected", rightContentWidth))
	}
	for len(telemetryLines) < topContentRows {
		telemetryLines = append(telemetryLines, "")
	}
	if len(telemetryLines) > topContentRows {
		telemetryLines = telemetryLines[:topContentRows]
	}
	topTelemetryBox := PanelStyle.Width(rightInnerWidth).Render(strings.Join(telemetryLines, "\n"))

	// 2b. Right-Bottom: Content Payload Panel
	var payloadLines []string
	payloadTitle := "CONTENT PAYLOAD"
	if m.isVisualMode {
		start := m.visualStart
		end := m.visualCursor
		if start > end {
			start, end = end, start
		}
		payloadTitle = fmt.Sprintf("CONTENT PAYLOAD (VISUAL: %d lines | [y] Copy)", end-start+1)
	} else if m.focusPane == FocusDetail {
		payloadTitle = "CONTENT PAYLOAD < [Scroll: j/k, Ctrl+u/d, g/G]"
	}
	payloadLines = append(payloadLines, TitleStyle.Render(truncateVisualWidth(payloadTitle, rightContentWidth)))

	if hasEvent {
		allPayloadLines := m.buildContentPayloadLines(selectedEvent, rightContentWidth)
		totalInspectorLines := len(allPayloadLines)

		availableLines := bottomContentRows - 1
		if availableLines < 1 {
			availableLines = 1
		}

		maxScroll := totalInspectorLines - availableLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		currentScroll := m.detailScroll
		if currentScroll > maxScroll {
			currentScroll = maxScroll
		}

		endLine := currentScroll + availableLines
		if endLine > totalInspectorLines {
			endLine = totalInspectorLines
		}

		vStart := m.visualStart
		vEnd := m.visualCursor
		if vStart > vEnd {
			vStart, vEnd = vEnd, vStart
		}

		for i := currentScroll; i < endLine; i++ {
			rawLine := allPayloadLines[i]
			lineText := truncateVisualWidth(rawLine, rightContentWidth)
			if m.isVisualMode && i >= vStart && i <= vEnd {
				payloadLines = append(payloadLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary).Render(lineText))
			} else {
				payloadLines = append(payloadLines, lineText)
			}
		}
	} else {
		payloadLines = append(payloadLines, truncateVisualWidth("  Select a step on the left to inspect details.", rightContentWidth))
	}

	for len(payloadLines) < bottomContentRows {
		payloadLines = append(payloadLines, "")
	}
	if len(payloadLines) > bottomContentRows {
		payloadLines = payloadLines[:bottomContentRows]
	}

	var bottomPayloadBox string
	if m.isVisualMode {
		bottomPayloadBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorHighlight).Padding(0, 1).Width(rightInnerWidth).Render(strings.Join(payloadLines, "\n"))
	} else if m.focusPane == FocusDetail {
		bottomPayloadBox = ActivePanelStyle.Width(rightInnerWidth).Render(strings.Join(payloadLines, "\n"))
	} else {
		bottomPayloadBox = PanelStyle.Width(rightInnerWidth).Render(strings.Join(payloadLines, "\n"))
	}

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topTelemetryBox, bottomPayloadBox)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightCol)
}

// truncateVisualWidth truncates a string by its visual terminal column width (CJK, Emoji & ANSI escape aware)
func truncateVisualWidth(s string, maxVisualWidth int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "    ")

	if maxVisualWidth <= 0 {
		return ""
	}

	w := 0
	runes := []rune(s)
	var res []rune
	inAnsi := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == 0x1b {
			inAnsi = true
			res = append(res, r)
			continue
		}
		if inAnsi {
			res = append(res, r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inAnsi = false
			}
			continue
		}

		rw := runewidth.RuneWidth(r)
		if w+rw > maxVisualWidth {
			break
		}
		res = append(res, r)
		w += rw
	}
	return string(res)
}

func shortenType(t string) string {
	switch t {
	case "USER_INPUT":
		return "USER"
	case "MODEL_RESPONSE":
		return "MODEL"
	case "TOOL_CALL":
		return "TOOL"
	case "RUN_COMMAND":
		return "CMD"
	case "VIEW_FILE":
		return "FILE"
	case "CODE_ACTION":
		return "CODE"
	case "LIST_DIRECTORY":
		return "DIR"
	case "GENERIC":
		return "OUT"
	case "ERROR_MESSAGE":
		return "ERR"
	default:
		if len(t) > 5 {
			return t[:5]
		}
		return t
	}
}

func formatShortCache(e core.UnifiedAgentEvent) string {
	if !e.IsCloudStep() || !e.Usage.Available {
		return ""
	}
	status := e.CacheStatus
	if status == "" {
		hitRate, ok := e.Usage.CacheHitRate()
		if !ok {
			return ""
		}
		status = core.ClassifyCacheStatus(hitRate, e.Usage.CachedTokens, e.Usage.TotalTokens)
	}
	hitRate, hasHitRate := e.Usage.CacheHitRate()
	switch status {
	case "HIT":
		if hasHitRate {
			return lipgloss.NewStyle().Foreground(ColorSuccess).Render(fmt.Sprintf("[HIT %.0f%%]", hitRate))
		}
		return ""
	case "PARTIAL":
		if hasHitRate {
			return lipgloss.NewStyle().Foreground(ColorWarning).Render(fmt.Sprintf("[PART %.0f%%]", hitRate))
		}
		return ""
	case "MISS":
		return lipgloss.NewStyle().Foreground(ColorDanger).Render("[MISS]")
	default:
		return ""
	}
}

func (m Model) formatHistoryCard(
	filtered []core.UnifiedAgentEvent,
	realIdx int,
	isSelected bool,
	maxWidth int,
) (string, string) {
	e := filtered[realIdx]

	isCloud := e.IsCloudStep() || e.Scope == core.ScopeCloudInference || e.Type == core.StepTypeModelResponse || e.Type == core.StepTypeToolCall
	isLocal := e.IsLocalStep() || e.Scope == core.ScopeLocalExecution
	isUser := e.Type == core.StepTypeUserInput || e.Scope == core.ScopeUserInteraction

	isCloudTop := false
	if isCloud && realIdx > 0 {
		eBelow := filtered[realIdx-1]
		if eBelow.IsLocalStep() || eBelow.Scope == core.ScopeLocalExecution || eBelow.Type == core.StepTypeUserInput || eBelow.Scope == core.ScopeUserInteraction {
			isCloudTop = true
		}
	}

	isLocalInside := false
	if (isLocal || isUser) && realIdx+1 < len(filtered) {
		eAbove := filtered[realIdx+1]
		if eAbove.IsCloudStep() || eAbove.Scope == core.ScopeCloudInference || eAbove.Type == core.StepTypeModelResponse || eAbove.Type == core.StepTypeToolCall {
			isLocalInside = true
		}
	}

	modelName := m.getStepModelName(e)
	toolName := m.getLocalToolName(e)

	connStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	headerStyle := DimRowStyle
	summaryStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	prefix := " "
	if isSelected {
		if m.focusPane == FocusList {
			headerStyle = SelectedRowStyle
			summaryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary)
			connStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary)
		} else {
			headerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
			summaryStyle = lipgloss.NewStyle().Foreground(ColorLightText)
			connStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
		}
		prefix = ">"
	}

	var cardLine1 string
	var cardLine2 string

	if isCloudTop {
		typeStr := formatStepType(string(e.Type))
		if e.IsSubagent || e.GetAgentRole() == "SUBAGENT" {
			typeStr = "👥 SUBAGENT"
		}
		cacheTag := formatShortCache(e)
		var text1 string
		if cacheTag != "" {
			text1 = fmt.Sprintf("[%04d] %s %s", e.StepIndex, typeStr, cacheTag)
		} else {
			text1 = fmt.Sprintf("[%04d] %s", e.StepIndex, typeStr)
		}

		if isSelected && m.focusPane == FocusList {
			cardLine1 = headerStyle.Render(truncateVisualWidth(prefix+"┌"+text1, maxWidth))
			if modelName != "" {
				cardLine2 = summaryStyle.Render(truncateVisualWidth(" │  Model: "+modelName, maxWidth))
			}
		} else {
			cardLine1 = prefix + connStyle.Render("┌") + headerStyle.Render(truncateVisualWidth(text1, maxWidth-2))
			if modelName != "" {
				cardLine2 = " " + connStyle.Render("│") + summaryStyle.Render(truncateVisualWidth("  Model: "+modelName, maxWidth-2))
			}
		}
	} else if isLocal {
		tag := "💻 OUTPUT"
		if e.Status == "BLOCKED" {
			tag = lipgloss.NewStyle().Foreground(ColorDanger).Render("🛡️ BLOCKED")
		} else if e.IsSubagent || e.GetAgentRole() == "SUBAGENT" {
			tag = lipgloss.NewStyle().Foreground(ColorSecondary).Render("👥 SUBAGENT")
		} else if e.Source == "SYSTEM" || e.GetAgentRole() == "INTERNAL" {
			tag = lipgloss.NewStyle().Foreground(ColorMuted).Render("⚙️ INTERNAL")
		}

		if toolName != "" && toolName != "OUTPUT" {
			text1 := fmt.Sprintf("[%04d] %s", e.StepIndex, tag)
			text2 := "  Tool: " + toolName

			if isSelected && m.focusPane == FocusList {
				cardLine1 = headerStyle.Render(truncateVisualWidth(prefix+"│"+text1, maxWidth))
				cardLine2 = summaryStyle.Render(truncateVisualWidth(" └"+text2, maxWidth))
			} else {
				cardLine1 = prefix + connStyle.Render("│") + headerStyle.Render(truncateVisualWidth(text1, maxWidth-2))
				cardLine2 = " " + connStyle.Render("└") + summaryStyle.Render(truncateVisualWidth(text2, maxWidth-2))
			}
		} else {
			connChar := " "
			if isLocalInside {
				connChar = "└"
			}
			text1 := fmt.Sprintf("[%04d] %s", e.StepIndex, tag)

			if isSelected && m.focusPane == FocusList {
				cardLine1 = headerStyle.Render(truncateVisualWidth(prefix+connChar+text1, maxWidth))
			} else {
				cardLine1 = prefix + connStyle.Render(connChar) + headerStyle.Render(truncateVisualWidth(text1, maxWidth-2))
			}
		}
	} else if isUser {
		connChar := " "
		if isLocalInside {
			connChar = "└"
		}
		text1 := fmt.Sprintf("[%04d] 👤 USER", e.StepIndex)

		if isSelected && m.focusPane == FocusList {
			cardLine1 = headerStyle.Render(truncateVisualWidth(prefix+connChar+text1, maxWidth))
		} else {
			cardLine1 = prefix + connStyle.Render(connChar) + headerStyle.Render(truncateVisualWidth(text1, maxWidth-2))
		}
	} else {
		// Standalone step (CHECKPOINT, SYSTEM_INIT, etc.)
		typeStr := formatStepType(string(e.Type))
		cacheTag := formatShortCache(e)
		var text1 string
		if cacheTag != "" {
			text1 = fmt.Sprintf("[%04d] %s %s", e.StepIndex, typeStr, cacheTag)
		} else {
			text1 = fmt.Sprintf("[%04d] %s", e.StepIndex, typeStr)
		}

		if isSelected && m.focusPane == FocusList {
			cardLine1 = headerStyle.Render(truncateVisualWidth(prefix+" "+text1, maxWidth))
			if modelName != "" {
				cardLine2 = summaryStyle.Render(truncateVisualWidth("    Model: "+modelName, maxWidth))
			}
		} else {
			cardLine1 = prefix + " " + headerStyle.Render(truncateVisualWidth(text1, maxWidth-2))
			if modelName != "" {
				cardLine2 = "   " + summaryStyle.Render(truncateVisualWidth("Model: "+modelName, maxWidth-3))
			}
		}
	}

	return cardLine1, cardLine2
}

func formatStepType(t string) string {
	switch t {
	case "USER_INPUT":
		return "👤 USER"
	case "MODEL_RESPONSE":
		return "🤖 MODEL"
	case "TOOL_CALL":
		return "🛠️ TOOL"
	case "RUN_COMMAND":
		return "💻 RUN_CMD"
	case "VIEW_FILE":
		return "📄 VIEW_FILE"
	case "CODE_ACTION":
		return "📝 CODE_DIFF"
	case "LIST_DIRECTORY":
		return "📁 LIST_DIR"
	case "GENERIC":
		return "💻 OUTPUT"
	case "ERROR_MESSAGE":
		return "⚠️ ERROR"
	case "CHECKPOINT":
		return "⚙️ CHECKPOINT"
	case "SYSTEM_INIT":
		return "⚙️ SYSTEM"
	default:
		return t
	}
}

func (m Model) getStepModelName(e core.UnifiedAgentEvent) string {
	if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution || e.Type == core.StepTypeUserInput {
		return ""
	}
	model := e.Usage.ModelName
	if model == "" && (e.IsCloudStep() || e.Type == core.StepTypeToolCall || e.Type == core.StepTypeModelResponse) {
		model = m.getSessionModelName()
	}
	if model == "" {
		return ""
	}
	model = strings.TrimPrefix(model, "models/")
	return model
}

func (m Model) getLocalToolName(e core.UnifiedAgentEvent) string {
	switch e.Type {
	case core.StepTypeRunCommand:
		return "run_cmd"
	case core.StepTypeViewFile:
		return "view_file"
	case core.StepTypeCodeAction:
		return "edit_file"
	case core.StepTypeListDirectory:
		return "list_dir"
	case core.StepTypeAskQuestion:
		return "ask_user"
	}

	if e.ParentStepIdx > 0 {
		for _, p := range m.history {
			if p.StepIndex == e.ParentStepIdx && len(p.ToolCalls) > 0 {
				return cleanToolDisplay(p.ToolCalls[0].ToolName)
			}
		}
	}

	for i := len(m.history) - 1; i >= 0; i-- {
		p := m.history[i]
		if p.StepIndex < e.StepIndex && (p.Type == core.StepTypeToolCall || len(p.ToolCalls) > 0) {
			if len(p.ToolCalls) > 0 {
				return cleanToolDisplay(p.ToolCalls[0].ToolName)
			}
			break
		}
	}

	return "OUTPUT"
}

func cleanToolDisplay(t string) string {
	switch t {
	case "run_command":
		return "run_cmd"
	case "replace_file_content", "write_to_file":
		return "edit_file"
	case "view_file":
		return "view_file"
	case "grep_search", "find_by_name":
		return "search"
	case "list_dir":
		return "list_dir"
	case "ask_question":
		return "ask_user"
	default:
		if len(t) > 9 {
			return t[:9]
		}
		return t
	}
}
