package ui

import (
	"fmt"
	"strings"

	"heimdall/internal/core"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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
	if n >= 1000000 {
		return fmt.Sprintf("%.2fM", float64(n)/1000000.0)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}

func renderBorderlessKpiStrip(tot core.ModelTokenStats, width int) string {
	if width < 100 {
		// Responsive 2-Column Grid for Narrow / Half Width Terminals
		colW := (width - 4) / 2
		if colW < 18 {
			colW = 18
		}
		hStyle := lipgloss.NewStyle().Foreground(ColorMuted).Bold(true)

		h1 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "TOTAL PROCESSED"), colW))
		h2 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "CACHE HIT VOLUME"), colW))
		v1 := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%s Tok (%d Turns)", formatTokShort(tot.TotalProcessed), tot.TurnCount)), colW))
		v2 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%s Tok (%.1f%% Hit)", formatTokShort(tot.TotalCached), tot.CacheHitRate)), colW))

		h3 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "UNCACHED INBOUND"), colW))
		h4 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "CACHED SAVED (%)"), colW))
		v3 := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%s Tok (%.1f%% Cold)", formatTokShort(tot.TotalNew), 100.0-tot.CacheHitRate)), colW))
		v4 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(
			truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%s Tok (%.1f%% Saved)", formatTokShort(tot.TokensSaved), tot.SavingsPercentage)), colW))

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

	h1 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "TOTAL PROCESSED"), colW))
	h2 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "CACHE HIT VOLUME"), colW))
	h3 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "UNCACHED INBOUND"), colW))
	h4 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "EFFECTIVE TOKENS"), colW))
	h5 := hStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, "CACHED SAVED (%)"), colW))

	v1 := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatTokShort(tot.TotalProcessed)+" Tok"), colW))
	v2 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatTokShort(tot.TotalCached)+" Tok"), colW))
	v3 := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatTokShort(tot.TotalNew)+" Tok"), colW))
	v4 := lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatTokShort(tot.EffectiveTokens)+" Tok"), colW))
	v5 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, formatTokShort(tot.TokensSaved)+" Tok"), colW))

	s1 := subStyle.Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%d Cloud Turns", tot.TurnCount)), colW))
	s2 := lipgloss.NewStyle().Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%.1f%% Hit Rate", tot.CacheHitRate)), colW))
	s3 := lipgloss.NewStyle().Foreground(ColorHighlight).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%.1f%% Cold In", 100.0-tot.CacheHitRate)), colW))
	effPct := 0.0
	if tot.TotalProcessed > 0 {
		effPct = float64(tot.EffectiveTokens) / float64(tot.TotalProcessed) * 100.0
	}
	s4 := lipgloss.NewStyle().Foreground(ColorSecondary).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%.1f%% of Raw", effPct)), colW))
	s5 := lipgloss.NewStyle().Foreground(ColorSuccess).Render(truncateVisualWidth(fmt.Sprintf("%-*s", colW, fmt.Sprintf("%.1f%% Net Saved", tot.SavingsPercentage)), colW))

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
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("  MULTI-MODEL TOKEN & SAVINGS BREAKDOWN:") + "\n")

	targetWidth := width - 4
	if targetWidth < 30 {
		targetWidth = 30
	}

	if width < 75 {
		// Tier 1: 4-Column Ultra-Compact Table for Narrow Laptop Splits (< 75 cols)
		minWidths := []int{14, 5, 10, 14}
		leftAligns := []bool{true, false, false, false}

		headers := []string{"Model Name", "Turns", "Processed", "Saved (%)"}
		headerStr := "  " + formatSpaceBetweenRow(headers, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(headerStr) + "\n")

		for _, m := range models {
			savStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(m.TokensSaved), m.SavingsPercentage)
			rowCols := []string{
				truncateVisualWidth(m.ModelName, 14),
				fmt.Sprintf("%d", m.TurnCount),
				formatTokShort(m.TotalProcessed),
				savStr,
			}
			rowStr := "  " + formatSpaceBetweenRow(rowCols, minWidths, leftAligns, targetWidth)
			sb.WriteString(rowStr + "\n")
		}

		sep := "  " + strings.Repeat("─", targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(sep) + "\n")

		totSavStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(total.TokensSaved), total.SavingsPercentage)
		totCols := []string{
			"TOTAL",
			fmt.Sprintf("%d", total.TurnCount),
			formatTokShort(total.TotalProcessed),
			totSavStr,
		}
		totStr := "  " + formatSpaceBetweenRow(totCols, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(totStr))

		return sb.String()
	} else if width < 110 {
		// Tier 2: 5-Column Compact Table for Medium Terminals (75 <= width < 110)
		minWidths := []int{16, 5, 9, 14, 14}
		leftAligns := []bool{true, false, false, false, false}

		headers := []string{"Model Name", "Turns", "Processed", "Cached (Hit %)", "Cached Saved (%)"}
		headerStr := "  " + formatSpaceBetweenRow(headers, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(headerStr) + "\n")

		for _, m := range models {
			hitStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(m.TotalCached), m.CacheHitRate)
			savStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(m.TokensSaved), m.SavingsPercentage)
			rowCols := []string{
				truncateVisualWidth(m.ModelName, 16),
				fmt.Sprintf("%d", m.TurnCount),
				formatTokShort(m.TotalProcessed),
				hitStr,
				savStr,
			}
			rowStr := "  " + formatSpaceBetweenRow(rowCols, minWidths, leftAligns, targetWidth)
			sb.WriteString(rowStr + "\n")
		}

		sep := "  " + strings.Repeat("─", targetWidth)
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(sep) + "\n")

		totHitStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(total.TotalCached), total.CacheHitRate)
		totSavStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(total.TokensSaved), total.SavingsPercentage)
		totCols := []string{
			"TOTAL SUMMARY",
			fmt.Sprintf("%d", total.TurnCount),
			formatTokShort(total.TotalProcessed),
			totHitStr,
			totSavStr,
		}
		totStr := "  " + formatSpaceBetweenRow(totCols, minWidths, leftAligns, targetWidth)
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(totStr))

		return sb.String()
	}

	// Tier 3: 7-Column Full Table for Wide Terminals (>= 110 cols)
	minWidths := []int{22, 6, 11, 16, 11, 16, 16}
	leftAligns := []bool{true, false, false, false, false, false, false}

	headers := []string{"Model Name", "Turns", "Processed", "Cached (Hit %)", "Uncached", "Effective (Factor)", "Cached Saved (%)"}
	headerStr := "  " + formatSpaceBetweenRow(headers, minWidths, leftAligns, targetWidth)
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(headerStr) + "\n")

	for _, m := range models {
		hitStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(m.TotalCached), m.CacheHitRate)
		effStr := fmt.Sprintf("%s (%s)", formatTokShort(m.EffectiveTokens), m.DiscountLabel)
		savStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(m.TokensSaved), m.SavingsPercentage)
		rowCols := []string{
			truncateVisualWidth(m.ModelName, 22),
			fmt.Sprintf("%d", m.TurnCount),
			formatTokShort(m.TotalProcessed),
			hitStr,
			formatTokShort(m.TotalNew),
			effStr,
			savStr,
		}
		rowStr := "  " + formatSpaceBetweenRow(rowCols, minWidths, leftAligns, targetWidth)
		sb.WriteString(rowStr + "\n")
	}

	sep := "  " + strings.Repeat("─", targetWidth)
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(sep) + "\n")

	totHitStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(total.TotalCached), total.CacheHitRate)
	totEffStr := fmt.Sprintf("%s (%s)", formatTokShort(total.EffectiveTokens), total.DiscountLabel)
	totSavStr := fmt.Sprintf("%s (%4.1f%%)", formatTokShort(total.TokensSaved), total.SavingsPercentage)
	totCols := []string{
		"TOTAL SUMMARY",
		fmt.Sprintf("%d", total.TurnCount),
		formatTokShort(total.TotalProcessed),
		totHitStr,
		formatTokShort(total.TotalNew),
		totEffStr,
		totSavStr,
	}
	totStr := "  " + formatSpaceBetweenRow(totCols, minWidths, leftAligns, targetWidth)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(totStr))

	return sb.String()
}

func renderTrendPanel(series core.TurnTrendSeries, width int) string {
	var sb strings.Builder
	pTitle := fmt.Sprintf("MULTI-TURN CONTEXT & CACHE HIT TREND (Last %d Cloud Turns)", len(series.Points))
	sb.WriteString(TitleStyle.Render(pTitle) + "\n")

	if len(series.Points) == 0 {
		sb.WriteString("  No cloud turns recorded yet...")
		return sb.String()
	}

	labelW := 25
	badgeW := 16
	sparkW := width - labelW - badgeW - 4
	if sparkW < 10 {
		sparkW = 10
	}

	var ctxVals, cacheVals, newVals, hitVals []float64
	for _, p := range series.Points {
		ctxVals = append(ctxVals, float64(p.TotalTokens))
		cacheVals = append(cacheVals, float64(p.CachedTokens))
		newVals = append(newVals, float64(p.NewTokens))
		hitVals = append(hitVals, p.CacheHitRate)
	}

	maxCtx := float64(series.MaxContext)
	if maxCtx <= 0 {
		maxCtx = 1.0
	}
	peakNew := float64(series.PeakNew)
	if peakNew <= 0 {
		peakNew = 1.0
	}

	// Line 1: Context Total
	s1 := RenderSparkline(ctxVals, maxCtx, sparkW, lipgloss.NewStyle().Foreground(ColorSecondary))
	b1 := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("[Peak: %s]", formatTokShort(series.MaxContext)))
	sb.WriteString(fmt.Sprintf("  Context Total (Cyan) : %s %s\n", s1, b1))

	// Line 2: Cached Volume
	s2 := RenderSparkline(cacheVals, maxCtx, sparkW, lipgloss.NewStyle().Foreground(ColorSuccess))
	b2 := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("[Curr: %s]", formatTokShort(series.LatestCached)))
	sb.WriteString(fmt.Sprintf("  Cached Volume (Green): %s %s\n", s2, b2))

	// Line 3: New Input
	s3 := RenderSparkline(newVals, peakNew, sparkW, lipgloss.NewStyle().Foreground(ColorHighlight))
	b3 := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("[Peak: %s]", formatTokShort(series.PeakNew)))
	sb.WriteString(fmt.Sprintf("  New Input     (Orange): %s %s\n", s3, b3))

	// Line 4: Hit Rate %
	s4 := RenderSparkline(hitVals, 100.0, sparkW, lipgloss.NewStyle().Foreground(ColorSuccess))
	b4 := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("[Avg: %4.1f%%]", series.AvgHitRate))
	sb.WriteString(fmt.Sprintf("  Hit Rate %%    (Lime) : %s %s", s4, b4))

	return sb.String()
}

func (m Model) renderDashboardView() string {
	e := m.latestEvent
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
		if idx < len(m.history)-1 {
			isPlayback = true
		}
	}

	t := e.Tokens
	total := t.TotalTokens

	dTotal := total
	if dTotal == 0 {
		dTotal = t.SystemTokens + t.ToolsDefTokens + t.ToolResultTokens + t.HistoryTokens + t.ActiveTurnTokens + t.ThinkingTokens
	}

	var sysPct, toolsPct, resPct, histPct, activePct float64
	if dTotal > 0 {
		sysPct = float64(t.SystemTokens) / float64(dTotal) * 100.0
		toolsPct = float64(t.ToolsDefTokens) / float64(dTotal) * 100.0
		resPct = float64(t.ToolResultTokens) / float64(dTotal) * 100.0
		histPct = float64(t.HistoryTokens) / float64(dTotal) * 100.0
		activePct = float64(t.ActiveTurnTokens+t.ThinkingTokens) / float64(dTotal) * 100.0
	}

	ctxLimit := t.ContextLimit
	if ctxLimit == 0 {
		ctxLimit = t.OfficialContextLimit
	}
	if ctxLimit == 0 {
		ctxLimit = core.DefaultFallbackAgentWindow
	}
	var ctxUsagePct float64
	if total > 0 {
		ctxUsagePct = float64(total) / float64(ctxLimit) * 100.0
	}

	var cacheBadge string
	if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
		cacheBadge = lipgloss.NewStyle().Foreground(ColorMuted).Render("[LOCAL OFFLINE / 0 GPU TOK]")
	} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
		cacheBadge = lipgloss.NewStyle().Foreground(ColorHighlight).Render("[USER INBOUND / STAGED]")
	} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
		cacheBadge = TitleStyle.Render("[CONTEXT RE-ANCHORED]")
	} else {
		switch e.CacheStatus {
		case "HIT":
			cacheBadge = BadgeSuccess.Render(fmt.Sprintf("[CACHE HIT %.1f%%]", t.CacheHitRate))
		case "PARTIAL":
			cacheBadge = BadgeWarning.Render(fmt.Sprintf("[PARTIAL HIT %.1f%%]", t.CacheHitRate))
		case "WRITE":
			cacheBadge = TitleStyle.Render("[CACHE WRITE / INITIAL]")
		case "EXPIRED":
			cacheBadge = BadgeDanger.Render("[TTL EXPIRED / COLD START]")
		default:
			cacheBadge = BadgeDanger.Render("[CACHE MISS / BROKEN]")
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
		invokedModel := m.getStepModelName(e)
		if invokedModel == "" {
			invokedModel = "Gemini 3.7 Flash"
		}
		toolName := m.getLocalToolName(e)
		if toolName == "" {
			toolName = string(e.Type)
		}
		packagedInfo := "Staged for Next Cloud Turn"
		if e.PackagedInStepIdx > 0 {
			packagedInfo = fmt.Sprintf("Billed in Cloud Turn #%04d ☁️", e.PackagedInStepIdx)
		}

		p1.WriteString(TitleStyle.Render(p1Title) + "\n")
		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Origin / Role  : %s (%s | Step #%03d)\n", e.GetAgentRole(), toolName, e.StepIndex))
			p1.WriteString(fmt.Sprintf("  • Total Context  : Nil (Local Subprocess)\n"))
			p1.WriteString(fmt.Sprintf("  • Step Delta     : +%s Tok (Local Buffer)\n", formatTokShort(t.StepDelta)))
			p1.WriteString(fmt.Sprintf("  • Status & Bill  : %s | %s", e.Status, packagedInfo))
		} else {
			p1.WriteString(fmt.Sprintf("  • Origin / Role        : %s (Tool Action: %s | Step #%04d | Status: %s | %s)\n", e.GetAgentRole(), lipgloss.NewStyle().Bold(true).Render(toolName), e.StepIndex, e.Status, timeStr))
			p1.WriteString(fmt.Sprintf("  • Total Context Window : Nil (Local Machine Subprocess / 0 GPU Inbound)\n"))
			p1.WriteString(fmt.Sprintf("  • Step Delta (Buffer)  : +%d Tokens (Local Output Buffer ➔ Staged for Next Turn)\n", t.StepDelta))
			if e.Status == "BLOCKED" {
				p1.WriteString("  • Billing Attribution  : Blocked by System Permission Guard (0 GPU Tokens Billed) 🛡️")
			} else {
				p1.WriteString(fmt.Sprintf("  • Billing Attribution  : Local Machine Subprocess (0 GPU Tokens) ➔ %s", packagedInfo))
			}
			if !isPlayback && e.ParentStepIdx > 0 {
				p1.WriteString(fmt.Sprintf("\n  • Invoking Parent      : Dispatched by Cloud Step #%04d (%s)", e.ParentStepIdx, invokedModel))
			}
		}
	} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
		p1Title := "TRACK 1: USER INTERACTION (CLIENT PROMPT)"
		p1.WriteString(TitleStyle.Render(p1Title) + "\n")
		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Origin / Role  : HUMAN CLIENT (Step #%03d)\n", e.StepIndex))
			p1.WriteString(fmt.Sprintf("  • Total Context  : Nil (Client Intent)\n"))
			p1.WriteString(fmt.Sprintf("  • Step Delta     : +%s Tok (Prompt Buffer)\n", formatTokShort(t.StepDelta)))
			p1.WriteString(fmt.Sprintf("  • Status & Time  : %s | %s", e.Status, timeStr))
		} else {
			p1.WriteString(fmt.Sprintf("  • Origin / Role        : HUMAN CLIENT Intent (Step #%03d | Status: %s | %s)\n", e.StepIndex, e.Status, timeStr))
			p1.WriteString(fmt.Sprintf("  • Total Context Window : Nil (Client Inbound Intent / Pre-Inference Staged)\n"))
			p1.WriteString(fmt.Sprintf("  • Step Delta (Prompt)  : +%d Tokens (Natural Language Inbound Buffer)\n", t.StepDelta))
			if e.PackagedInStepIdx > 0 {
				p1.WriteString(fmt.Sprintf("  • Inference Settlement : Ingested ➔ Settled & Billed in Cloud Step #%04d ☁️", e.PackagedInStepIdx))
			} else {
				p1.WriteString("  • Billing Status       : Inbound Intent ➔ Staged locally (Awaiting Next Cloud Inference Turn ⏳)")
			}
		}
	} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
		p1Title := "TRACK 1: SYSTEM COMPACTION (CHECKPOINT)"
		p1.WriteString(TitleStyle.Render(p1Title) + "\n")
		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Event / Role   : %s Context Compaction (Step #%03d)\n", e.GetAgentRole(), e.StepIndex))
			p1.WriteString(fmt.Sprintf("  • Summary Size   : %d Tokens  %s\n", total, cacheBadge))
			p1.WriteString(fmt.Sprintf("  • Step Delta     : +%s Tok (GC Summary)\n", formatTokShort(t.StepDelta)))
			p1.WriteString(fmt.Sprintf("  • Status & Time  : %s | %s", e.Status, timeStr))
		} else {
			p1.WriteString(fmt.Sprintf("  • Origin / Role        : %s Middleware (Sidecar Context GC | Step #%03d | Status: %s | %s)\n", e.GetAgentRole(), e.StepIndex, e.Status, timeStr))
			p1.WriteString(fmt.Sprintf("  • Summary Size         : %d Tokens (Replaces ~200k+ Old Historical Tokens)  %s\n", total, cacheBadge))
			p1.WriteString(fmt.Sprintf("  • Step Delta (Summary) : +%d Tokens (Compacted Summary Payload)\n", t.StepDelta))
			p1.WriteString("  • Window Action        : Prepend Checkpoint ➔ Re-anchors Active Window Base for Next Turn 🔄")
		}
	} else {
		p1Title := "TRACK 1: OFFICIAL CLOUD TELEMETRY"
		if e.IsSubagent || e.GetAgentRole() == "SUBAGENT" {
			p1Title = "TRACK 1: SUBAGENT CLOUD TELEMETRY"
		}
		p1.WriteString(TitleStyle.Render(p1Title) + "\n")

		// Dynamic financial & capacity calculations
		modelName := t.OfficialModel
		if modelName == "" {
			modelName = "gemini-3.7-flash"
		}
		cachedRate := t.PricingCachedUSD
		uncachedRate := t.PricingUncachedUSD
		if uncachedRate == 0 {
			spec := core.ResolveModelSpec(modelName, nil)
			cachedRate = spec.Pricing.CachedPerMillionUSD
			uncachedRate = spec.Pricing.UncachedPerMillionUSD
		}
		costUSD := float64(t.CachedTokens)*cachedRate/core.TokensPerMillion + float64(t.NewTokens)*uncachedRate/core.TokensPerMillion
		discountRate := t.CacheDiscount
		if discountRate == 0 {
			discountRate = 0.90
		}
		savedTok := int(float64(t.CachedTokens) * discountRate)
		exchangeRate := t.ExchangeRate
		if exchangeRate == 0 {
			exchangeRate = core.DefaultUSDtoTWDExchangeRate
		}
		costTWD := costUSD * exchangeRate

		if contentWidth < 80 {
			p1.WriteString(fmt.Sprintf("  • Agent / Model  : [%s] %s (Step #%03d)\n",
				e.GetAgentRole(), lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(truncateVisualWidth(modelName, contentWidth-28)), e.StepIndex))
			if total > 0 {
				p1.WriteString(fmt.Sprintf("  • Total Context  : %s Tok (%4.1f%% of %dk Window) %s\n",
					lipgloss.NewStyle().Bold(true).Render(formatTokShort(total)), ctxUsagePct, ctxLimit/1000, cacheBadge))
				p1.WriteString(fmt.Sprintf("  • Step Delta     : +%s Tok | %s Cached (%.1f%%)\n",
					formatTokShort(t.StepDelta), formatTokShort(t.CachedTokens), t.CacheHitRate))
			} else {
				p1.WriteString(fmt.Sprintf("  • Total Context  : Nil (Awaiting Telemetry ⏳)\n"))
				p1.WriteString(fmt.Sprintf("  • Step Delta     : +%s Tok (Estimated)\n", formatTokShort(t.StepDelta)))
			}
			p1.WriteString(fmt.Sprintf("  • Status & Cost  : ~$%.4f USD | Status: %s", costUSD, e.Status))
		} else {
			p1.WriteString(fmt.Sprintf("  • Agent / Model        : [%s] %s  (Step #%03d | Status: %s | %s)\n",
				e.GetAgentRole(), lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(modelName), e.StepIndex, e.Status, timeStr))
			if total > 0 {
				p1.WriteString(fmt.Sprintf("  • Total Context Window : %s Tokens (%5.1f%% of %dk Window)  %s\n",
					lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", total)), ctxUsagePct, ctxLimit/1000, cacheBadge))
				p1.WriteString(fmt.Sprintf("  • Step Delta (New In)  : +%s Tokens (Uncached Prefill) | Turn Cost: ~$%.4f USD (NT$ %.2f)\n",
					formatTokShort(t.StepDelta), costUSD, costTWD))
				p1.WriteString(fmt.Sprintf("  • Prefix Cache Savings : %s Tokens Cached (%.1f%% Hit) ➔ Net Saved ~%s Tok (%.1f%% Discount)",
					formatTokShort(t.CachedTokens), t.CacheHitRate, formatTokShort(savedTok), discountRate*100.0))
			} else {
				p1.WriteString(fmt.Sprintf("  • Total Context Window : Nil (Awaiting SQLite Telemetry Record ⏳)\n"))
				p1.WriteString(fmt.Sprintf("  • Step Delta (Local)   : +%d Tokens (Estimated Local BPE)", t.StepDelta))
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

	// ==================== PANEL 2: TRACK 2: LOCAL 5-DIMENSION CONTEXT ANATOMY ====================
	var p2 strings.Builder
	p2Title := "TRACK 2: LOCAL 5-DIMENSION CONTEXT ANATOMY (PAYLOAD ANALYSIS)"
	p2.WriteString(TitleStyle.Render(p2Title) + "\n")

	barBlocks := 12
	if contentWidth < 70 {
		barBlocks = 8
	}

	if contentWidth < 80 {
		p2.WriteString(fmt.Sprintf("  1. System Prompt : %-6s (%4.1f%%) [%s]\n",
			formatTokShort(t.SystemTokens), sysPct, renderColorBar(sysPct, barBlocks, ColorSecondary)))
		p2.WriteString(fmt.Sprintf("  2. Tools Schema  : %-6s (%4.1f%%) [%s]\n",
			formatTokShort(t.ToolsDefTokens), toolsPct, renderColorBar(toolsPct, barBlocks, ColorSecondary)))
		p2.WriteString(fmt.Sprintf("  3. Tool Results  : %-6s (%4.1f%%) [%s]\n",
			formatTokShort(t.ToolResultTokens), resPct, renderColorBar(resPct, barBlocks, ColorHighlight)))
		p2.WriteString(fmt.Sprintf("  4. History Turns : %-6s (%4.1f%%) [%s]\n",
			formatTokShort(t.HistoryTokens), histPct, renderColorBar(histPct, barBlocks, ColorPrimary)))
		p2.WriteString(fmt.Sprintf("  5. Active / CoT  : %-6s (%4.1f%%) [%s]",
			formatTokShort(t.ActiveTurnTokens+t.ThinkingTokens), activePct, renderColorBar(activePct, barBlocks, ColorWarning)))
	} else {
		p2.WriteString(fmt.Sprintf("  1. System Instruction : %-8s Tokens (%5.1f%%)  [%s]\n",
			formatTokShort(t.SystemTokens), sysPct, renderColorBar(sysPct, 15, ColorSecondary)))
		p2.WriteString(fmt.Sprintf("  2. MCP Tools Schema   : %-8s Tokens (%5.1f%%)  [%s]\n",
			formatTokShort(t.ToolsDefTokens), toolsPct, renderColorBar(toolsPct, 15, ColorSecondary)))
		p2.WriteString(fmt.Sprintf("  3. Tool Results / Diff: %-8s Tokens (%5.1f%%)  [%s]\n",
			formatTokShort(t.ToolResultTokens), resPct, renderColorBar(resPct, 15, ColorHighlight)))
		p2.WriteString(fmt.Sprintf("  4. Conversation Hist  : %-8s Tokens (%5.1f%%)  [%s]\n",
			formatTokShort(t.HistoryTokens), histPct, renderColorBar(histPct, 15, ColorPrimary)))
		p2.WriteString(fmt.Sprintf("  5. Active Turn / CoT  : %-8s Tokens (%5.1f%%)  [%s]",
			formatTokShort(t.ActiveTurnTokens+t.ThinkingTokens), activePct, renderColorBar(activePct, 15, ColorWarning)))
	}

	panel2Box := PanelStyle.Width(panelInnerWidth).Render(p2.String())

	return lipgloss.JoinVertical(lipgloss.Left, panel0Box, panel1Box, panel2Box)
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
		leftLines = append(leftLines, DimRowStyle.Render(truncateVisualWidth("  No matching steps...", topLeftContentWidth)))
		leftLines = append(leftLines, DimRowStyle.Render(truncateVisualWidth("  Press [Esc] to reset", topLeftContentWidth)))
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
		rightLines = append(rightLines, DimRowStyle.Render(truncateVisualWidth("  No step selected", topRightContentWidth)))
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
		bottomLines = append(bottomLines, DimRowStyle.Render(truncateVisualWidth("  Select a step above to inspect details.", bottomContentWidth)))
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
		leftLines = append(leftLines, DimRowStyle.Render(truncateVisualWidth("  No matching steps...", listContentWidth)))
		leftLines = append(leftLines, DimRowStyle.Render(truncateVisualWidth("  Press [Esc] to reset", listContentWidth)))
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
		telemetryLines = append(telemetryLines, DimRowStyle.Render(truncateVisualWidth("  No step selected", rightContentWidth)))
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
		payloadLines = append(payloadLines, DimRowStyle.Render(truncateVisualWidth("  Select a step on the left to inspect details.", rightContentWidth)))
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

func renderColorBar(pct float64, totalBlocks int, color lipgloss.TerminalColor) string {
	filled := int((pct / 100.0) * float64(totalBlocks))
	if filled > totalBlocks {
		filled = totalBlocks
	}
	if filled < 0 {
		filled = 0
	}
	filledStr := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	emptyStr := lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("░", totalBlocks-filled))
	return filledStr + emptyStr
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
	// Only cloud inference steps have GPU KV-Cache telemetry
	if !e.IsCloudStep() {
		return ""
	}
	status := e.CacheStatus
	if status == "" {
		status = core.ClassifyCacheStatus(e.Tokens.CacheHitRate, e.Tokens.CachedTokens, e.Tokens.TotalTokens, false)
	}
	switch status {
	case "HIT":
		return lipgloss.NewStyle().Foreground(ColorSuccess).Render(fmt.Sprintf("[HIT %.0f%%]", e.Tokens.CacheHitRate))
	case "PARTIAL":
		return lipgloss.NewStyle().Foreground(ColorWarning).Render(fmt.Sprintf("[PART %.0f%%]", e.Tokens.CacheHitRate))
	case "WRITE":
		return lipgloss.NewStyle().Foreground(ColorSecondary).Render("[WRITE]")
	case "EXPIRED", "TTL_EXPIRED":
		return lipgloss.NewStyle().Foreground(ColorHighlight).Render("[EXPIRED]")
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
	model := e.Tokens.OfficialModel
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

