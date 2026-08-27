package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"agent-observer/internal/core"
)

// SwitchSessionReqMsg is sent when the user selects a new session in the switcher
type SwitchSessionReqMsg struct {
	SessionID string
}

// SessionSwitchedMsg is sent to update the UI model with the new session's state
type SessionSwitchedMsg struct {
	SessionID string
	Events    []core.UnifiedAgentEvent
}

func (m Model) renderSessionSwitcherModal() string {
	modalWidth := m.width - 6
	if modalWidth > 116 {
		modalWidth = 116
	}
	if modalWidth < 46 {
		modalWidth = 46
	}

	modalInnerWidth := modalWidth - 4
	contentWidth := modalInnerWidth - 2 // Account for Padding(0, 1)
	modalHeight := m.height - 4
	if modalHeight > 24 {
		modalHeight = 24
	}
	if modalHeight < 12 {
		modalHeight = 12
	}

	var contentLines []string

	// 1. Modal Title
	titleText := "SWITCH AGENT SESSION [Ctrl+P]"
	contentLines = append(contentLines, TitleStyle.Render(truncateVisualWidth(titleText, contentWidth)))

	// 2. Real Visual Tabs for AI Agent Types
	var agyCount, claudeCount, opencodeCount int
	for _, s := range m.availableSessions {
		switch s.AgentType {
		case core.AgentTypeClaudeCode:
			claudeCount++
		case core.AgentTypeOpenCode:
			opencodeCount++
		default:
			agyCount++
		}
	}

	tabAStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	tabCStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	tabOStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	tabAText := fmt.Sprintf(" Antigravity (%d) ", agyCount)
	tabCText := fmt.Sprintf(" Claude Code (%d) ", claudeCount)
	tabOText := fmt.Sprintf(" OpenCode (%d) ", opencodeCount)

	switch m.selectedAgentTab {
	case core.AgentTypeClaudeCode:
		tabCText = fmt.Sprintf("[%s]", strings.TrimSpace(tabCText))
		tabCStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg)
	case core.AgentTypeOpenCode:
		tabOText = fmt.Sprintf("[%s]", strings.TrimSpace(tabOText))
		tabOStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg)
	default:
		tabAText = fmt.Sprintf("[%s]", strings.TrimSpace(tabAText))
		tabAStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg)
	}

	tabA := tabAStyle.Render(tabAText)
	tabC := tabCStyle.Render(tabCText)
	tabO := tabOStyle.Render(tabOText)
	tabsLine := tabA + "   " + tabC + "   " + tabO
	contentLines = append(contentLines, lipgloss.NewStyle().MaxWidth(contentWidth).Render(tabsLine))

	// 3. Search Input Box
	queryDisplay := m.sessionSearchQuery
	cursorChar := "█"
	inputBox := fmt.Sprintf("Filter: [%s%s] (Type to filter by path, hash, or prompt keywords)", queryDisplay, lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar))
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(inputBox, contentWidth)))
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", contentWidth)))

	// 4. Body Content (Dual Pane vs Single Pane)
	bodyHeight := modalHeight - 7
	if bodyHeight < 4 {
		bodyHeight = 4
	}

	sessions := m.filteredSessions
	isDualPane := contentWidth >= 76

	if isDualPane {
		leftWidth := (contentWidth - 3) * 44 / 100
		if leftWidth < 32 {
			leftWidth = 32
		}
		rightWidth := contentWidth - leftWidth - 3
		if rightWidth < 30 {
			rightWidth = 30
		}

		// Build Left Lines (Session List Cards)
		var leftLines []string
		maxCards := bodyHeight / 2
		if maxCards < 1 {
			maxCards = 1
		}

		if len(sessions) == 0 {
			leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  No matching sessions."))
			leftLines = append(leftLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  Press [ / ] for tabs."))
		} else {
			startIdx := m.switcherSelectedIdx - (maxCards / 2)
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx := startIdx + maxCards
			if endIdx > len(sessions) {
				endIdx = len(sessions)
				startIdx = endIdx - maxCards
				if startIdx < 0 {
					startIdx = 0
				}
			}

			for i := startIdx; i < endIdx; i++ {
				s := sessions[i]
				isActive := (s.SessionID == m.sessionID)
				isSelected := (i == m.switcherSelectedIdx)

				prefix := "  "
				shortHash := s.SessionID
				if len(shortHash) > 8 {
					shortHash = shortHash[:8]
				}

				agentBadge := "[AGY]"
				switch s.AgentType {
				case core.AgentTypeClaudeCode:
					agentBadge = "[CLAUDE]"
				case core.AgentTypeOpenCode:
					agentBadge = "[OPEN]"
				}

				cardStyle1 := DimRowStyle
				cardStyle2 := lipgloss.NewStyle().Foreground(ColorMuted)

				if isSelected {
					prefix = "> "
					cardStyle1 = SelectedRowStyle
					cardStyle2 = lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Background(ColorDarkBg)
				} else if isActive {
					cardStyle1 = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)
				}

				relTime := formatRelativeTime(s.LastModified)
				statusTag := ""
				if isActive {
					statusTag = " | [ACTIVE]"
				}

				// Line 1: Prefix + Badge + ShortPath + (#Hash)
				line1 := fmt.Sprintf("%s%s %s (#%s)", prefix, agentBadge, s.ShortPath, shortHash)
				leftLines = append(leftLines, cardStyle1.Render(truncateVisualWidth(line1, leftWidth)))

				// Line 2: Steps + Size + Relative Time + Status
				line2 := fmt.Sprintf("    %d steps | %.1fMB | %s%s", s.StepCount, s.SizeMB, relTime, statusTag)
				leftLines = append(leftLines, cardStyle2.Render(truncateVisualWidth(line2, leftWidth)))
			}
		}

		for len(leftLines) < bodyHeight {
			leftLines = append(leftLines, strings.Repeat(" ", leftWidth))
		}
		if len(leftLines) > bodyHeight {
			leftLines = leftLines[:bodyHeight]
		}

		// Build Right Lines (Live Session Inspector)
		var rightLines []string
		if len(sessions) > 0 && m.switcherSelectedIdx >= 0 && m.switcherSelectedIdx < len(sessions) {
			s := sessions[m.switcherSelectedIdx]

			// Section 1: Initial Goal
			rightLines = append(rightLines, TitleStyle.Render(truncateVisualWidth("[INITIAL GOAL / FIRST PROMPT]", rightWidth)))
			goalText := s.InitialGoal
			if strings.TrimSpace(goalText) == "" {
				goalText = "(No initial user prompt recorded)"
			}
			goalWrapped := wrapVisualLines(goalText, rightWidth)
			maxGoalLines := (bodyHeight - 3) / 2
			if maxGoalLines < 2 {
				maxGoalLines = 2
			}
			for idx, gl := range goalWrapped {
				if idx >= maxGoalLines {
					rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("..."))
					break
				}
				rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(gl))
			}

			// Divider
			rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", rightWidth)))

			// Section 2: Latest Progress
			rightLines = append(rightLines, TitleStyle.Render(truncateVisualWidth("[LATEST PROGRESS / LAST ACTION]", rightWidth)))
			lastText := s.LastPrompt
			if strings.TrimSpace(lastText) == "" {
				lastText = "(No recent user action recorded)"
			}
			lastWrapped := wrapVisualLines(lastText, rightWidth)
			for _, ll := range lastWrapped {
				if len(rightLines) >= bodyHeight {
					break
				}
				rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(ll))
			}
		} else {
			rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("(Select a session to preview context)"))
		}

		for len(rightLines) < bodyHeight {
			rightLines = append(rightLines, strings.Repeat(" ", rightWidth))
		}
		if len(rightLines) > bodyHeight {
			rightLines = rightLines[:bodyHeight]
		}

		// Join Left and Right with Vertical Divider
		for i := 0; i < bodyHeight; i++ {
			lL := truncateVisualWidth(leftLines[i], leftWidth)
			// Pad lL to leftWidth
			padL := leftWidth - lipgloss.Width(lL)
			if padL > 0 {
				lL += strings.Repeat(" ", padL)
			}
			rL := truncateVisualWidth(rightLines[i], rightWidth)
			combined := fmt.Sprintf("%s %s %s", lL, lipgloss.NewStyle().Foreground(ColorBorder).Render("│"), rL)
			contentLines = append(contentLines, combined)
		}
	} else {
		// Single Pane Fallback for narrow windows
		maxCards := bodyHeight / 2
		if maxCards < 1 {
			maxCards = 1
		}

		if len(sessions) == 0 {
			contentLines = append(contentLines, "  No matching sessions found.")
		} else {
			startIdx := m.switcherSelectedIdx - (maxCards / 2)
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx := startIdx + maxCards
			if endIdx > len(sessions) {
				endIdx = len(sessions)
				startIdx = endIdx - maxCards
				if startIdx < 0 {
					startIdx = 0
				}
			}

			for i := startIdx; i < endIdx; i++ {
				s := sessions[i]
				isActive := (s.SessionID == m.sessionID)
				isSelected := (i == m.switcherSelectedIdx)
				prefix := "  "
				statusTag := ""
				if isSelected {
					prefix = "> "
				}
				if isActive {
					statusTag = " [ACTIVE]"
				}
				line1 := fmt.Sprintf("%s[%s] %s (%d steps)%s", prefix, s.AgentType, s.ShortPath, s.StepCount, statusTag)
				line2 := fmt.Sprintf("    %s | %s", formatRelativeTime(s.LastModified), s.InitialGoal)
				contentLines = append(contentLines, truncateVisualWidth(line1, contentWidth))
				contentLines = append(contentLines, truncateVisualWidth(line2, contentWidth))
			}
		}

		for len(contentLines) < modalHeight-2 {
			contentLines = append(contentLines, "")
		}
	}

	// 5. Modal Footer / Keybindings
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", contentWidth)))
	footerHints := fmt.Sprintf(" %s Move  %s Attach  %s Tab  %s Cancel",
		KeyStyle.Render("[Ctrl+j/k, ↑/↓]"), KeyStyle.Render("[Enter]"), KeyStyle.Render("[[ / ]]"), KeyStyle.Render("[Esc]"))
	contentLines = append(contentLines, lipgloss.NewStyle().MaxWidth(contentWidth).Render(footerHints))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorHighlight).
		Background(ColorDarkBg).
		Padding(0, 1).
		Width(modalInnerWidth).
		Render(strings.Join(contentLines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
}

func filterSessions(sessions []core.SessionInfo, query string, agentTab core.AgentType) []core.SessionInfo {
	var matched []core.SessionInfo
	q := strings.ToLower(strings.TrimSpace(query))

	for _, s := range sessions {
		// Filter by Agent Type
		if agentTab != "" && s.AgentType != agentTab {
			continue
		}

		// Filter by Query
		if q == "" {
			matched = append(matched, s)
			continue
		}

		if strings.Contains(strings.ToLower(s.SessionID), q) ||
			strings.Contains(strings.ToLower(s.ShortPath), q) ||
			strings.Contains(strings.ToLower(s.WorkspaceDir), q) ||
			strings.Contains(strings.ToLower(s.InitialGoal), q) ||
			strings.Contains(strings.ToLower(s.LastPrompt), q) ||
			strings.Contains(strings.ToLower(s.ModelName), q) {
			matched = append(matched, s)
		}
	}
	return matched
}
