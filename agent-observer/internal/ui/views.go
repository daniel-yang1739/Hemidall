package ui

import (
	"fmt"
	"strings"

	"agent-observer/internal/core"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

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
	if total == 0 {
		total = 1
	}

	sysPct := float64(t.SystemTokens) / float64(total) * 100.0
	toolsPct := float64(t.ToolsDefTokens) / float64(total) * 100.0
	resPct := float64(t.ToolResultTokens) / float64(total) * 100.0
	histPct := float64(t.HistoryTokens) / float64(total) * 100.0
	activePct := float64(t.ActiveTurnTokens+t.ThinkingTokens) / float64(total) * 100.0

	ctxLimit := t.OfficialContextLimit
	if ctxLimit == 0 {
		ctxLimit = 256000
	}
	ctxUsagePct := float64(total) / float64(ctxLimit) * 100.0

	var cacheBadge string
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

	panelInnerWidth := m.width - 2
	if panelInnerWidth < 40 {
		panelInnerWidth = 40
	}
	contentWidth := panelInnerWidth - 2 // Account for Padding(0, 1)

	// ==================== PANEL 1: OFFICIAL TELEMETRY & CACHE (TRACK 1) ====================
	var p1 strings.Builder
	p1Title := "TRACK 1: OFFICIAL GEMINI TELEMETRY (BILLING GROUND TRUTH)"
	if isPlayback {
		p1Title = fmt.Sprintf("TRACK 1: OFFICIAL TELEMETRY %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf("(PLAYBACK: Step #%d | %d of %d)", e.StepIndex, m.dashboardIdx+1, len(m.history))))
	}
	p1.WriteString(TitleStyle.Render(p1Title) + "\n")

	modelName := t.OfficialModel
	if modelName == "" {
		modelName = "gemini-3.7-flash-high"
	}

	timeStr := e.Timestamp.Format("2006-01-02 15:04:05")
	if e.Timestamp.IsZero() {
		timeStr = "N/A"
	}

	// 5 Comprehensive Points for Track 1
	p1.WriteString(fmt.Sprintf("  • Backend Model         : %s\n", lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(modelName)))
	p1.WriteString(fmt.Sprintf("  • Total Active Context  : %s Tokens (%5.1f%% of %dk Window)\n",
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", total)), ctxUsagePct, ctxLimit/1000))
	p1.WriteString(fmt.Sprintf("  • Prefix Cache Hit      : %s Tokens (%5.1f%%)  %s\n",
		BadgeSuccess.Render(fmt.Sprintf("%d", t.CachedTokens)), t.CacheHitRate, cacheBadge))
	p1.WriteString(fmt.Sprintf("  • New Billable Tokens   : %s Tokens (%5.1f%%)\n",
		lipgloss.NewStyle().Foreground(ColorHighlight).Render(fmt.Sprintf("%d", t.NewTokens)), 100.0-t.CacheHitRate))
	p1.WriteString(fmt.Sprintf("  • Response / Event Time : %s  (Step #%03d | Status: %s)",
		lipgloss.NewStyle().Foreground(ColorLightText).Render(timeStr), e.StepIndex, e.Status))

	panel1Box := PanelStyle.Width(panelInnerWidth).Render(p1.String())

	// ==================== PANEL 2: LOCAL 5-DIMENSION CONTEXT ANATOMY (TRACK 2) ====================
	var p2 strings.Builder
	p2Title := TitleStyle.Render("TRACK 2: LOCAL 5-DIMENSION CONTEXT ANATOMY (PAYLOAD ANALYSIS)")
	p2.WriteString(p2Title + "\n")

	p2.WriteString(fmt.Sprintf("  1. System Instruction : %-8d Tokens (%5.1f%%)  [%s]\n",
		t.SystemTokens, sysPct, renderColorBar(sysPct, 15, ColorSecondary)))
	p2.WriteString(fmt.Sprintf("  2. MCP Tools Schema   : %-8d Tokens (%5.1f%%)  [%s]\n",
		t.ToolsDefTokens, toolsPct, renderColorBar(toolsPct, 15, ColorSecondary)))
	p2.WriteString(fmt.Sprintf("  3. Tool Results / Diff: %-8d Tokens (%5.1f%%)  [%s]\n",
		t.ToolResultTokens, resPct, renderColorBar(resPct, 15, ColorHighlight)))
	p2.WriteString(fmt.Sprintf("  4. Conversation Hist  : %-8d Tokens (%5.1f%%)  [%s]\n",
		t.HistoryTokens, histPct, renderColorBar(histPct, 15, ColorPrimary)))
	p2.WriteString(fmt.Sprintf("  5. Active Turn / CoT  : %-8d Tokens (%5.1f%%)  [%s]",
		t.ActiveTurnTokens+t.ThinkingTokens, activePct, renderColorBar(activePct, 15, ColorWarning)))

	if t.RawLocalAccumulated > total && m.height >= 34 {
		truncated := t.RawLocalAccumulated - total
		p2.WriteString(fmt.Sprintf("\n  Raw Log Accumulated   : %d Tokens (%d Tokens Truncated by Cloud Window)",
			t.RawLocalAccumulated, truncated))
	}

	panel2Box := PanelStyle.Width(panelInnerWidth).Render(p2.String())

	// ==================== PANEL 3: RECENT LIVE EVENTS (CONTENT-HUGGING ROUNDED BOX) ====================
	if len(m.history) > 0 || m.height >= 26 {
		var p3Lines []string
		p3Lines = append(p3Lines, TitleStyle.Render("RECENT LIVE EVENTS (Press [Enter] or [2] to inspect history)"))

		maxEventLines := 6
		if len(m.history) == 0 {
			p3Lines = append(p3Lines, "  No events recorded yet...")
		} else {
			startIdx := len(m.history) - maxEventLines
			if startIdx < 0 {
				startIdx = 0
			}
			for i := startIdx; i < len(m.history); i++ {
				ev := m.history[i]
				typeBadge := fmt.Sprintf("[%03d|%-5s]", ev.StepIndex, shortenType(string(ev.Type)))
				timeStr := ev.Timestamp.Format("15:04:05")
				eventLine := fmt.Sprintf("  %s %s  %s", typeBadge, timeStr, ev.Summary)
				p3Lines = append(p3Lines, truncateVisualWidth(eventLine, contentWidth))
			}
		}

		panel3Box := PanelStyle.Width(panelInnerWidth).Render(strings.Join(p3Lines, "\n"))
		return lipgloss.JoinVertical(lipgloss.Left, panel1Box, panel2Box, panel3Box)
	}

	return lipgloss.JoinVertical(lipgloss.Left, panel1Box, panel2Box)
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
		filterBox := fmt.Sprintf("Filter: [#%s%s]", m.historyStepQuery, lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(filterBox, topLeftContentWidth)))
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
			e := filtered[realIdx]

			prefix := "  "
			headerStyle := DimRowStyle
			summaryStyle := lipgloss.NewStyle().Foreground(ColorMuted)

			if i == m.selectedIdx {
				prefix = "> "
				if m.focusPane == FocusList {
					headerStyle = SelectedRowStyle
					summaryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary)
				} else {
					headerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
					summaryStyle = lipgloss.NewStyle().Foreground(ColorLightText)
				}
			}

			isLocal := e.IsLocalStep() || e.Scope == core.ScopeLocalExecution

			var cardLine1 string
			var cardLine2 string

			if isLocal {
				cardLine1 = fmt.Sprintf("%s└── [%04d] OUTPUT %s", prefix, e.StepIndex, lipgloss.NewStyle().Foreground(ColorSuccess).Render("(Local)"))
				toolName := m.getLocalToolName(e)
				if toolName != "" && toolName != "OUTPUT" {
					cardLine2 = "        Tool: " + toolName
				}
			} else {
				typeStr := formatStepType(string(e.Type))
				cacheTag := formatShortCache(e)
				if cacheTag != "" {
					cardLine1 = fmt.Sprintf("%s[%04d] %s %s", prefix, e.StepIndex, typeStr, cacheTag)
				} else {
					cardLine1 = fmt.Sprintf("%s[%04d] %s", prefix, e.StepIndex, typeStr)
				}

				modelName := m.getStepModelName(e)
				if modelName != "" {
					cardLine2 = "    Model: " + modelName
				}
			}

			leftLines = append(leftLines, headerStyle.Render(truncateVisualWidth(cardLine1, topLeftContentWidth)))
			if cardLine2 != "" {
				leftLines = append(leftLines, summaryStyle.Render(truncateVisualWidth(cardLine2, topLeftContentWidth)))
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
		filterBox := fmt.Sprintf("Filter: [#%s%s]", m.historyStepQuery, lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar))
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(filterBox, listContentWidth)))
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
			e := filtered[realIdx]

			prefix := "  "
			headerStyle := DimRowStyle
			summaryStyle := lipgloss.NewStyle().Foreground(ColorMuted)

			if i == m.selectedIdx {
				prefix = "> "
				if m.focusPane == FocusList {
					headerStyle = SelectedRowStyle
					summaryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary)
				} else {
					headerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
					summaryStyle = lipgloss.NewStyle().Foreground(ColorLightText)
				}
			}

			isLocal := e.IsLocalStep() || e.Scope == core.ScopeLocalExecution

			var cardLine1 string
			var cardLine2 string

			if isLocal {
				cardLine1 = fmt.Sprintf("%s└── [%04d] OUTPUT %s", prefix, e.StepIndex, lipgloss.NewStyle().Foreground(ColorSuccess).Render("(Local)"))
				toolName := m.getLocalToolName(e)
				if toolName != "" && toolName != "OUTPUT" {
					cardLine2 = "        Tool: " + toolName
				}
			} else {
				typeStr := formatStepType(string(e.Type))
				cacheTag := formatShortCache(e)
				if cacheTag != "" {
					cardLine1 = fmt.Sprintf("%s[%04d] %s %s", prefix, e.StepIndex, typeStr, cacheTag)
				} else {
					cardLine1 = fmt.Sprintf("%s[%04d] %s", prefix, e.StepIndex, typeStr)
				}

				modelName := m.getStepModelName(e)
				if modelName != "" {
					cardLine2 = "    Model: " + modelName
				}
			}

			leftLines = append(leftLines, headerStyle.Render(truncateVisualWidth(cardLine1, listContentWidth)))
			if cardLine2 != "" {
				leftLines = append(leftLines, summaryStyle.Render(truncateVisualWidth(cardLine2, listContentWidth)))
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
	switch e.CacheStatus {
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
		if e.Tokens.CacheHitRate >= 70.0 {
			return lipgloss.NewStyle().Foreground(ColorSuccess).Render(fmt.Sprintf("[HIT %.0f%%]", e.Tokens.CacheHitRate))
		} else if e.Tokens.CacheHitRate > 0.0 {
			return lipgloss.NewStyle().Foreground(ColorWarning).Render(fmt.Sprintf("[PART %.0f%%]", e.Tokens.CacheHitRate))
		} else if e.StepIndex == 0 {
			return lipgloss.NewStyle().Foreground(ColorSecondary).Render("[WRITE]")
		}
		return ""
	}
}

func formatStepType(t string) string {
	switch t {
	case "USER_INPUT":
		return "USER_INPUT"
	case "MODEL_RESPONSE":
		return "MODEL_RESP"
	case "TOOL_CALL":
		return "TOOL_CALL"
	case "RUN_COMMAND":
		return "RUN_CMD"
	case "VIEW_FILE":
		return "VIEW_FILE"
	case "CODE_ACTION":
		return "CODE_DIFF"
	case "LIST_DIRECTORY":
		return "LIST_DIR"
	case "GENERIC":
		return "OUTPUT"
	case "ERROR_MESSAGE":
		return "ERROR"
	case "SYSTEM_INIT":
		return "SYSTEM"
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

