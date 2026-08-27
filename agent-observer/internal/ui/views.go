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
	leftOuterWidth := int(float64(m.width) * 0.30)
	if leftOuterWidth < 28 {
		leftOuterWidth = 28
	}
	if leftOuterWidth > 34 {
		leftOuterWidth = 34
	}
	rightOuterWidth := m.width - leftOuterWidth
	if rightOuterWidth < 30 {
		rightOuterWidth = 30
	}

	listInnerWidth := leftOuterWidth - 2
	detailInnerWidth := rightOuterWidth - 2

	listContentWidth := listInnerWidth - 2
	detailContentWidth := detailInnerWidth - 2

	innerRowsLimit := m.height - 4
	if innerRowsLimit < 4 {
		innerRowsLimit = 4
	}

	linesPerCard := m.getHistoryLinesPerCard()

	// ==================== 1. Left Pane: Step List ====================
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

	// Always render both [T:...] and [C:...] filter indicators
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

	var titleLine string
	if listContentWidth < 28 {
		titleLine = fmt.Sprintf("%s %s%s", TitleStyle.Render(leftTitle), typeBadgeStr, cacheBadgeStr)
	} else {
		titleLine = fmt.Sprintf("%s %s %s", TitleStyle.Render(leftTitle), typeBadgeStr, cacheBadgeStr)
	}
	leftLines = append(leftLines, truncateVisualWidth(titleLine, listContentWidth))

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

		// Top indicator: only if there are newer steps above (m.historyOffset > 0)
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
			typeBadge := fmt.Sprintf("[%03d|%-4s]", e.StepIndex, shortenType(string(e.Type)))
			label := getStepDistinctiveLabel(e)

			var cardLine1 string
			var cardLine2 string

			if isLocal {
				cardLine1 = fmt.Sprintf("%s└── %s %s %s", prefix, typeBadge, label, lipgloss.NewStyle().Foreground(ColorSuccess).Render("💻"))
				summaryText := e.Summary
				if summaryText == "" {
					summaryText = "(empty content)"
				}
				cardLine2 = "      " + summaryText
			} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
				cacheTag := formatShortCache(e)
				if cacheTag != "" {
					cardLine1 = fmt.Sprintf("%s%s %s %s", prefix, typeBadge, label, cacheTag)
				} else {
					cardLine1 = fmt.Sprintf("%s%s %s %s", prefix, typeBadge, label, lipgloss.NewStyle().Foreground(ColorPrimary).Render("👤"))
				}
				summaryText := e.Summary
				if summaryText == "" {
					summaryText = "(empty content)"
				}
				cardLine2 = "  " + summaryText
			} else {
				cacheTag := formatShortCache(e)
				if cacheTag != "" {
					cardLine1 = fmt.Sprintf("%s%s %s %s", prefix, typeBadge, label, cacheTag)
				} else {
					cardLine1 = fmt.Sprintf("%s%s %s", prefix, typeBadge, label)
				}
				summaryText := e.Summary
				if summaryText == "" {
					summaryText = "(empty content)"
				}
				cardLine2 = "  " + summaryText
			}

			leftLines = append(leftLines, headerStyle.Render(truncateVisualWidth(cardLine1, listContentWidth)))
			if linesPerCard == 2 {
				leftLines = append(leftLines, summaryStyle.Render(truncateVisualWidth(cardLine2, listContentWidth)))
			}
		}

		// Bottom indicator: only if there are older steps below (endIdx < len(filtered))
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

	// ==================== 2. Right Pane: Selected Step Detail Inspector (Word-Wrapped Buffer) ====================
	var rightLines []string
	rightTitle := "STEP INSPECTOR"
	if m.isVisualMode {
		start := m.visualStart
		end := m.visualCursor
		if start > end {
			start, end = end, start
		}
		rightTitle = fmt.Sprintf("STEP INSPECTOR (VISUAL: %d lines | [y] Copy)", end-start+1)
	} else if m.focusPane == FocusDetail {
		rightTitle = "STEP INSPECTOR < [Scroll: j/k, Ctrl+u/d, g/G]"
	}
	rightLines = append(rightLines, TitleStyle.Render(truncateVisualWidth(rightTitle, detailContentWidth)))

	selectedEvent, hasEvent := m.getSelectedEvent()
	if hasEvent {
		allInspectorLines := m.buildFullInspectorLines(selectedEvent, detailContentWidth)
		totalInspectorLines := len(allInspectorLines)

		availableLines := innerRowsLimit - 1
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
			rawLine := allInspectorLines[i]
			lineText := truncateVisualWidth(rawLine, detailContentWidth)
			if m.isVisualMode && i >= vStart && i <= vEnd {
				rightLines = append(rightLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary).Render(lineText))
			} else {
				rightLines = append(rightLines, lineText)
			}
		}
	} else {
		rightLines = append(rightLines, truncateVisualWidth("  Select a step on the left to inspect details.", detailContentWidth))
	}

	for len(rightLines) < innerRowsLimit {
		rightLines = append(rightLines, "")
	}
	if len(rightLines) > innerRowsLimit {
		rightLines = rightLines[:innerRowsLimit]
	}

	var rightBox string
	if m.isVisualMode {
		rightBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorHighlight).Padding(0, 1).Width(detailInnerWidth).Render(strings.Join(rightLines, "\n"))
	} else if m.focusPane == FocusDetail {
		rightBox = ActivePanelStyle.Width(detailInnerWidth).Render(strings.Join(rightLines, "\n"))
	} else {
		rightBox = PanelStyle.Width(detailInnerWidth).Render(strings.Join(rightLines, "\n"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
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

func getStepDistinctiveLabel(e core.UnifiedAgentEvent) string {
	switch e.Type {
	case core.StepTypeToolCall:
		if len(e.ToolCalls) == 1 {
			return shortenToolName(e.ToolCalls[0].ToolName)
		} else if len(e.ToolCalls) > 1 {
			return fmt.Sprintf("%s +%d", shortenToolName(e.ToolCalls[0].ToolName), len(e.ToolCalls)-1)
		}
		return "tool_call"
	case core.StepTypeRunCommand:
		return "run_cmd"
	case core.StepTypeViewFile:
		return "view_file"
	case core.StepTypeCodeAction:
		return "code_diff"
	case core.StepTypeListDirectory:
		return "list_dir"
	case core.StepTypeAskQuestion:
		return "ask_user"
	case core.StepTypeGeneric, core.StepTypeToolResult:
		return "Output"
	case core.StepTypeError:
		return "Error"
	case core.StepTypeUserInput:
		return "Prompt"
	case core.StepTypeModelResponse:
		if e.Tokens.OfficialModel != "" {
			return shortenModelName(e.Tokens.OfficialModel)
		}
		return "Gemini"
	case core.StepTypeSystemInit:
		return "System"
	default:
		if e.IsLocalStep() {
			return "Output"
		}
		return string(e.Type)
	}
}

func shortenToolName(n string) string {
	switch n {
	case "run_command":
		return "run_cmd"
	case "view_file":
		return "view_file"
	case "replace_file_content", "write_to_file":
		return "edit_file"
	case "list_dir":
		return "list_dir"
	case "grep_search", "find_by_name":
		return "search"
	default:
		if len(n) > 9 {
			return n[:9]
		}
		return n
	}
}

func shortenModelName(m string) string {
	m = strings.TrimPrefix(m, "models/")
	if strings.Contains(m, "3.7-flash") {
		return "Gemini 3.7"
	}
	if strings.Contains(m, "3.7-pro") {
		return "Gemini 3.7 Pro"
	}
	if strings.Contains(m, "flash") {
		return "Gemini Flash"
	}
	if strings.Contains(m, "pro") {
		return "Gemini Pro"
	}
	return m
}

