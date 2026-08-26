package ui

import (
	"fmt"
	"strings"

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
		cacheBadge = BadgeSuccess.Render(fmt.Sprintf("🟢 [CACHE HIT %.1f%%]", t.CacheHitRate))
	case "PARTIAL":
		cacheBadge = BadgeWarning.Render(fmt.Sprintf("🟡 [PARTIAL HIT %.1f%%]", t.CacheHitRate))
	case "WRITE":
		cacheBadge = TitleStyle.Render("🔵 [CACHE WRITE / INITIAL]")
	case "EXPIRED":
		cacheBadge = BadgeDanger.Render("🔴 [TTL EXPIRED / COLD START]")
	default:
		cacheBadge = BadgeDanger.Render("🔴 [CACHE MISS / BROKEN]")
	}

	panelInnerWidth := m.width - 4
	if panelInnerWidth < 40 {
		panelInnerWidth = 40
	}

	// ==================== PANEL 1: GOOGLE OFFICIAL TELEMETRY ====================
	var p1 strings.Builder
	p1Title := "👑 TRACK 1: Google 官方真實物理帳單 (Official Gemini Telemetry)"
	if isPlayback {
		p1Title = fmt.Sprintf("👑 TRACK 1: 官方帳單 %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf("(⏮️ PLAYBACK: Step #%d | %d of %d)", e.StepIndex, m.dashboardIdx+1, len(m.history))))
	}
	p1.WriteString(TitleStyle.Render(p1Title) + "\n")

	modelName := t.OfficialModel
	if modelName == "" {
		modelName = "gemini-3.7-flash-high (Official API)"
	}

	p1.WriteString(fmt.Sprintf("  • 🎯 Backend Model         : %s\n", lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(modelName)))
	p1.WriteString(fmt.Sprintf("  • 📊 Total Active Context  : %s Tokens (%5.1f%% of %dk Window)\n",
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", total)), ctxUsagePct, ctxLimit/1000))
	p1.WriteString(fmt.Sprintf("  • ⚡ Prefix Cache Hit      : %s Tokens (%5.1f%%)  %s\n",
		BadgeSuccess.Render(fmt.Sprintf("%d", t.CachedTokens)), t.CacheHitRate, cacheBadge))
	p1.WriteString(fmt.Sprintf("  • 🔥 New Billable Tokens   : %s Tokens (%5.1f%%)",
		lipgloss.NewStyle().Foreground(ColorHighlight).Render(fmt.Sprintf("%d", t.NewTokens)), 100.0-t.CacheHitRate))

	panel1Box := PanelStyle.Width(panelInnerWidth).Render(p1.String())

	// ==================== PANEL 2: 5-DIMENSION CONTEXT ANATOMY ====================
	var p2 strings.Builder
	p2Title := TitleStyle.Render("🔬 TRACK 2: 本地 5 維度 Context 載荷深度解剖 (Context Payload Anatomy)")
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
		p2.WriteString(fmt.Sprintf("\n  💡 Raw Log Accumulated: %d Tokens (✂️ %d Tokens Truncated by Cloud Window)",
			t.RawLocalAccumulated, truncated))
	}

	panel2Box := PanelStyle.Width(panelInnerWidth).Render(p2.String())

	if m.height >= 32 {
		var p3 strings.Builder
		p3.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("📋 Recent Live Events (Press [Enter] or [2] to inspect history):") + "\n")
		eventsCount := 2
		if m.height >= 38 {
			eventsCount = 4
		}
		startIdx := len(m.history) - eventsCount
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(m.history); i++ {
			ev := m.history[i]
			p3.WriteString(fmt.Sprintf("  [%s] [Step %03d | %-12s] %s\n",
				ev.Timestamp.Format("15:04:05"), ev.StepIndex, ev.Type, truncateVisualWidth(ev.Summary, panelInnerWidth-30)))
		}
		panel3Box := lipgloss.NewStyle().Padding(0, 1).Render(p3.String())
		return lipgloss.JoinVertical(lipgloss.Left, panel1Box, panel2Box, panel3Box)
	}

	return lipgloss.JoinVertical(lipgloss.Left, panel1Box, panel2Box)
}

func (m Model) renderHistoryView() string {
	// Full width allocation: leftOuterWidth + rightOuterWidth = m.width (100% full screen width)
	leftOuterWidth := int(float64(m.width) * 0.32)
	if leftOuterWidth < 26 {
		leftOuterWidth = 26
	}
	rightOuterWidth := m.width - leftOuterWidth
	if rightOuterWidth < 30 {
		rightOuterWidth = 30
	}

	listInnerWidth := leftOuterWidth - 4
	detailInnerWidth := rightOuterWidth - 4

	// Total screen height = 1 (header) + OuterBoxHeight (m.height - 2) + 1 (footer) = m.height
	// OuterBoxHeight = Top Border (1) + InnerLines (innerRowsLimit) + Bottom Border (1)
	// Therefore: innerRowsLimit = m.height - 4
	innerRowsLimit := m.height - 4
	if innerRowsLimit < 4 {
		innerRowsLimit = 4
	}

	// ==================== 1. Left Pane: Step List (2 Lines per Step Card) ====================
	var leftLines []string
	leftTitle := "📜 Steps (Newest First)"
	if m.focusPane == FocusList {
		leftTitle = "📜 Steps (Newest First ◀)"
	}
	leftLines = append(leftLines, TitleStyle.Render(truncateVisualWidth(leftTitle, listInnerWidth-2)))

	if len(m.history) == 0 {
		leftLines = append(leftLines, truncateVisualWidth("  No events yet...", listInnerWidth-2))
	} else {
		// Each step card takes 2 lines (Line 1: Header/Badge/Time, Line 2: Summary)
		maxCards := (innerRowsLimit - 1) / 2
		if maxCards < 1 {
			maxCards = 1
		}
		endIdx := m.historyOffset + maxCards
		if endIdx > len(m.history) {
			endIdx = len(m.history)
		}

		for i := m.historyOffset; i < endIdx; i++ {
			realIdx := len(m.history) - 1 - i
			e := m.history[realIdx]

			prefix := "  "
			headerStyle := DimRowStyle
			summaryStyle := lipgloss.NewStyle().Foreground(ColorMuted)

			if i == m.selectedIdx {
				prefix = "▶ "
				if m.focusPane == FocusList {
					headerStyle = SelectedRowStyle
					summaryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary)
				} else {
					headerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
					summaryStyle = lipgloss.NewStyle().Foreground(ColorLightText)
				}
			}

			typeBadge := fmt.Sprintf("[%03d|%-5s]", e.StepIndex, shortenType(string(e.Type)))
			timeStr := e.Timestamp.Format("15:04:05")

			// Card Line 1: Header + Type + Timestamp
			cardLine1 := fmt.Sprintf("%s%s %s", prefix, typeBadge, timeStr)
			leftLines = append(leftLines, headerStyle.Render(truncateVisualWidth(cardLine1, listInnerWidth-2)))

			// Card Line 2: Summary preview (indented by 2 spaces)
			summaryText := e.Summary
			if summaryText == "" {
				summaryText = "(empty content)"
			}
			cardLine2 := "  " + summaryText
			leftLines = append(leftLines, summaryStyle.Render(truncateVisualWidth(cardLine2, listInnerWidth-2)))
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
	rightTitle := "🔍 Step Inspector"
	if m.isVisualMode {
		start := m.visualStart
		end := m.visualCursor
		if start > end {
			start, end = end, start
		}
		rightTitle = fmt.Sprintf("🔍 Inspector (VISUAL: %d lines | [y] Copy)", end-start+1)
	} else if m.focusPane == FocusDetail {
		rightTitle = "🔍 Step Inspector ◀ [Scroll: ↑/↓, Ctrl+u/d, g/G]"
	}
	rightLines = append(rightLines, TitleStyle.Render(truncateVisualWidth(rightTitle, detailInnerWidth-2)))

	selectedEvent, hasEvent := m.getSelectedEvent()
	if hasEvent {
		// Build pre-wrapped virtual buffer where every line strictly fits detailInnerWidth-2
		allInspectorLines := m.buildFullInspectorLines(selectedEvent, detailInnerWidth-2)
		totalInspectorLines := len(allInspectorLines)

		availableLines := innerRowsLimit - 1 // 1 line reserved for title
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
			// Strict visual width clamp to guarantee zero terminal line wrapping
			lineText := truncateVisualWidth(rawLine, detailInnerWidth-2)
			if m.isVisualMode && i >= vStart && i <= vEnd {
				rightLines = append(rightLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary).Render(lineText))
			} else {
				rightLines = append(rightLines, lineText)
			}
		}
	} else {
		rightLines = append(rightLines, truncateVisualWidth("  Select a step on the left to inspect details.", detailInnerWidth-2))
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

// truncateVisualWidth truncates a string by its visual terminal column width (CJK & Emoji aware)
// This implements CSS `overflow: hidden; white-space: nowrap` for terminal grid cells.
func truncateVisualWidth(s string, maxVisualWidth int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "    ")

	if maxVisualWidth <= 0 {
		return ""
	}

	w := 0
	var res []rune
	for _, r := range []rune(s) {
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
	default:
		return t
	}
}
