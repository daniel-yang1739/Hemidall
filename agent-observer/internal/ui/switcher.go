package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"agent-observer/internal/adapters/antigravity"
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
	modalWidth := m.width - 10
	if modalWidth > 84 {
		modalWidth = 84
	}
	if modalWidth < 46 {
		modalWidth = 46
	}

	modalInnerWidth := modalWidth - 4
	modalHeight := m.height - 6
	if modalHeight > 18 {
		modalHeight = 18
	}
	if modalHeight < 10 {
		modalHeight = 10
	}

	var contentLines []string

	// 1. Modal Title
	titleText := "🔍 SWITCH SESSION [Ctrl+P]"
	contentLines = append(contentLines, TitleStyle.Render(truncateVisualWidth(titleText, modalInnerWidth)))

	// 2. Search Input Box
	queryDisplay := m.sessionSearchQuery
	cursorChar := "█"
	inputBox := fmt.Sprintf("🔎 Search: [%s%s] (Type to filter)", queryDisplay, lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar))
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(inputBox, modalInnerWidth)))
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", modalInnerWidth)))

	// 3. Filtered Session List
	maxVisibleRows := modalHeight - 6 // Reserve 3 lines for title/search/sep, 3 lines for footer
	if maxVisibleRows < 2 {
		maxVisibleRows = 2
	}

	maxCards := maxVisibleRows / 2
	if maxCards < 1 {
		maxCards = 1
	}

	sessions := m.filteredSessions
	if len(sessions) == 0 {
		contentLines = append(contentLines, "  No matching sessions found.")
		contentLines = append(contentLines, "  Type a different search keyword or press [Esc] to cancel.")
	} else {
		// Calculate viewport window for switcher
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
			statusTag := "[PAST  ]"
			if isActive {
				statusTag = "[ACTIVE]"
			}

			cardStyle1 := DimRowStyle
			cardStyle2 := lipgloss.NewStyle().Foreground(ColorMuted)

			if isSelected {
				prefix = "▶ "
				cardStyle1 = SelectedRowStyle
				cardStyle2 = lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Background(ColorDarkBg)
			} else if isActive {
				cardStyle1 = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)
			}

			// Line 1: Badge + ID + Steps + Size
			line1 := fmt.Sprintf("%s%s %s  (%4d steps | %4.1fMB)",
				prefix, statusTag, s.SessionID, s.StepCount, s.SizeMB)
			contentLines = append(contentLines, cardStyle1.Render(truncateVisualWidth(line1, modalInnerWidth)))

			// Line 2: Last activity timestamp
			line2 := fmt.Sprintf("    🕒 Last Modified: %s", s.LastModified.Format("2006-01-02 15:04:05"))
			contentLines = append(contentLines, cardStyle2.Render(truncateVisualWidth(line2, modalInnerWidth)))
		}
	}

	// Pad content to modalHeight - 2
	for len(contentLines) < modalHeight-2 {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > modalHeight-2 {
		contentLines = contentLines[:modalHeight-2]
	}

	// 4. Modal Footer / Keybindings
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", modalInnerWidth)))
	footerHints := fmt.Sprintf(" %s Move  %s Attach  %s Cancel  %s Filter",
		KeyStyle.Render("[Ctrl+j/k, ↑/↓]"), KeyStyle.Render("[Enter]"), KeyStyle.Render("[Esc/Ctrl+p]"), KeyStyle.Render("[Type]"))
	contentLines = append(contentLines, lipgloss.NewStyle().MaxWidth(modalInnerWidth).Render(footerHints))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorHighlight).
		Background(ColorDarkBg).
		Padding(0, 1).
		Width(modalInnerWidth).
		Render(strings.Join(contentLines, "\n"))

	// Center the modal dialog over the screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
}

func filterSessions(sessions []antigravity.SessionInfo, query string) []antigravity.SessionInfo {
	if strings.TrimSpace(query) == "" {
		return sessions
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var matched []antigravity.SessionInfo
	for _, s := range sessions {
		if strings.Contains(strings.ToLower(s.SessionID), q) ||
			strings.Contains(strings.ToLower(s.Title), q) ||
			strings.Contains(strings.ToLower(s.DBPath), q) {
			matched = append(matched, s)
		}
	}
	return matched
}
