package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"agent-observer/internal/adapters/antigravity"
	"agent-observer/internal/core"
)

type ActiveView int

const (
	ViewDashboard ActiveView = iota
	ViewHistory
)

type FocusPane int

const (
	FocusList FocusPane = iota
	FocusDetail
)

// AgentEventMsg wraps core.UnifiedAgentEvent as a bubbletea message
type AgentEventMsg core.UnifiedAgentEvent

// Model represents the bubbletea application state
type Model struct {
	sessionID             string
	activeView            ActiveView
	focusPane             FocusPane
	dashboardIdx          int
	detailScroll          int
	isVisualMode          bool
	visualStart           int
	visualCursor          int
	clipboardStatus       string
	clipboardStatusTime   time.Time
	latestEvent           core.UnifiedAgentEvent
	history               []core.UnifiedAgentEvent
	selectedIdx           int
	historyOffset         int
	width                 int
	height                int
	eventCount            int
	lastActivity          time.Time
	isSessionSwitcherOpen bool
	isHelpModalOpen       bool
	isGlossaryModalOpen   bool
	sessionSearchQuery    string
	availableSessions     []antigravity.SessionInfo
	filteredSessions      []antigravity.SessionInfo
	switcherSelectedIdx   int
}

// NewModel creates an initial TUI model
func NewModel(sessionID string, openSwitcherOnStart bool) Model {
	sessions, _ := antigravity.DiscoverAllSessions()

	m := Model{
		sessionID:             sessionID,
		activeView:            ViewDashboard,
		focusPane:             FocusList,
		dashboardIdx:          0,
		detailScroll:          0,
		history:               make([]core.UnifiedAgentEvent, 0, 1000),
		selectedIdx:           0,
		lastActivity:          time.Now(),
		availableSessions:     sessions,
		filteredSessions:      sessions,
		switcherSelectedIdx:   0,
		isSessionSwitcherOpen: openSwitcherOnStart,
		isHelpModalOpen:       false,
		isGlossaryModalOpen:   false,
	}

	// If sessionID matches one of discovered sessions, select it in the switcher
	for i, s := range sessions {
		if s.SessionID == sessionID {
			m.switcherSelectedIdx = i
			break
		}
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) getSelectedEvent() (core.UnifiedAgentEvent, bool) {
	if len(m.history) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.history) {
		return core.UnifiedAgentEvent{}, false
	}
	realIdx := len(m.history) - 1 - m.selectedIdx
	if realIdx < 0 || realIdx >= len(m.history) {
		return core.UnifiedAgentEvent{}, false
	}
	return m.history[realIdx], true
}

func (m Model) buildFullInspectorLines(e core.UnifiedAgentEvent, maxWidth int) []string {
	if maxWidth <= 10 {
		maxWidth = 60
	}
	var lines []string
	t := e.Tokens

	headerLine1 := fmt.Sprintf("• Step %03d (%s) at %s | Type: %s | Model: %s",
		e.StepIndex, e.Status, e.Timestamp.Format("15:04:05"), e.Type, t.OfficialModel)
	lines = append(lines, wrapVisualLines(headerLine1, maxWidth)...)

	headerLine2 := fmt.Sprintf("• Tokens: %d | Cached: %d (%.1f%%) | New: %d",
		t.TotalTokens, t.CachedTokens, t.CacheHitRate, t.NewTokens)
	lines = append(lines, wrapVisualLines(headerLine2, maxWidth)...)

	headerLine3 := fmt.Sprintf("• 5-Dims: Sys=%d | Tools=%d | Res=%d | Hist=%d | Act=%d",
		t.SystemTokens, t.ToolsDefTokens, t.ToolResultTokens, t.HistoryTokens, t.ActiveTurnTokens)
	lines = append(lines, wrapVisualLines(headerLine3, maxWidth)...)

	sepWidth := maxWidth
	if sepWidth > 80 {
		sepWidth = 80
	}
	lines = append(lines, strings.Repeat("─", sepWidth))

	if len(e.ToolCalls) > 0 {
		for _, tc := range e.ToolCalls {
			tcText := fmt.Sprintf("Tool: %s (args: %v)", tc.ToolName, tc.Arguments)
			lines = append(lines, wrapVisualLines(tcText, maxWidth)...)
		}
		lines = append(lines, strings.Repeat("─", sepWidth))
	}

	if strings.TrimSpace(e.Thinking) != "" {
		lines = append(lines, "Thinking (Chain of Thought):")
		lines = append(lines, wrapVisualLines(e.Thinking, maxWidth)...)
		lines = append(lines, strings.Repeat("─", sepWidth))
	}

	lines = append(lines, "Content Payload:")
	if strings.TrimSpace(e.RawContent) != "" {
		wrappedPayload := wrapVisualLines(e.RawContent, maxWidth)
		lines = append(lines, wrappedPayload...)
	} else if len(e.ToolCalls) > 0 {
		lines = append(lines, "  (No conversational text; pure tool call invocation)")
	} else {
		wrappedPayload := wrapVisualLines(e.Summary, maxWidth)
		lines = append(lines, wrappedPayload...)
	}

	return lines
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case AgentEventMsg:
		event := core.UnifiedAgentEvent(msg)
		m.latestEvent = event
		m.eventCount++
		m.lastActivity = time.Now()

		wasAtLatestDashboard := (m.dashboardIdx == len(m.history)-1 || len(m.history) == 0)
		isInspectingPastStep := (m.activeView == ViewHistory && (m.selectedIdx > 0 || m.focusPane == FocusDetail))

		m.history = append(m.history, event)

		if wasAtLatestDashboard {
			m.dashboardIdx = len(m.history) - 1
		}

		// If inspecting a past step or focused on the Inspector, lock the current inspection step in place
		if isInspectingPastStep {
			m.selectedIdx++
			if m.historyOffset > 0 {
				m.historyOffset++
			}
		}
		return m, nil

	case SwitchSessionReqMsg:
		analyzer := core.NewPayloadAnalyzer()
		events, _ := antigravity.LoadSessionHistory(msg.SessionID, analyzer)
		m.sessionID = msg.SessionID
		m.history = events
		m.eventCount = len(events)
		if len(events) > 0 {
			m.latestEvent = events[len(events)-1]
			m.dashboardIdx = len(events) - 1
		} else {
			m.latestEvent = core.UnifiedAgentEvent{}
			m.dashboardIdx = 0
		}
		m.selectedIdx = 0
		m.detailScroll = 0
		m.historyOffset = 0
		m.isSessionSwitcherOpen = false
		m.isHelpModalOpen = false
		m.isGlossaryModalOpen = false
		m.activeView = ViewDashboard
		m.clipboardStatus = fmt.Sprintf("Attached session %s (%d steps)", truncateStr(msg.SessionID, 8), len(events))
		m.clipboardStatusTime = time.Now()
		return m, nil

	case SessionSwitchedMsg:
		m.sessionID = msg.SessionID
		m.history = msg.Events
		m.eventCount = len(msg.Events)
		if len(msg.Events) > 0 {
			m.latestEvent = msg.Events[len(msg.Events)-1]
			m.dashboardIdx = len(msg.Events) - 1
		} else {
			m.latestEvent = core.UnifiedAgentEvent{}
			m.dashboardIdx = 0
		}
		m.selectedIdx = 0
		m.detailScroll = 0
		m.historyOffset = 0
		m.isSessionSwitcherOpen = false
		m.isHelpModalOpen = false
		m.isGlossaryModalOpen = false
		m.activeView = ViewDashboard
		m.clipboardStatus = fmt.Sprintf("Switched to session %s (%d steps)", truncateStr(msg.SessionID, 8), len(msg.Events))
		m.clipboardStatusTime = time.Now()
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// Global toggle for Glossary Modal: h (when in Dashboard or global without typing)
		if key == "h" && !m.isSessionSwitcherOpen && !m.isHelpModalOpen && (m.activeView == ViewDashboard || m.focusPane == FocusList) {
			m.isGlossaryModalOpen = !m.isGlossaryModalOpen
			return m, nil
		}

		// Glossary Modal Interaction
		if m.isGlossaryModalOpen {
			if key == "esc" || key == "h" || key == "q" || key == "enter" || key == "?" {
				m.isGlossaryModalOpen = false
				return m, nil
			}
			return m, nil
		}

		// Global toggle for Help Modal: ? or F1
		if key == "?" || key == "f1" {
			if !m.isSessionSwitcherOpen && !m.isGlossaryModalOpen {
				m.isHelpModalOpen = !m.isHelpModalOpen
				return m, nil
			}
		}

		// Help Modal Interaction
		if m.isHelpModalOpen {
			if key == "esc" || key == "?" || key == "q" || key == "enter" || key == "h" {
				m.isHelpModalOpen = false
				return m, nil
			}
			return m, nil
		}

		// Global toggle for session switcher modal: Ctrl+P
		if key == "ctrl+p" {
			m.isSessionSwitcherOpen = !m.isSessionSwitcherOpen
			if m.isSessionSwitcherOpen {
				m.isHelpModalOpen = false
				m.isGlossaryModalOpen = false
				m.availableSessions, _ = antigravity.DiscoverAllSessions()
				m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery)
				m.switcherSelectedIdx = 0
				for i, s := range m.filteredSessions {
					if s.SessionID == m.sessionID {
						m.switcherSelectedIdx = i
						break
					}
				}
			}
			return m, nil
		}

		// ==================== SESSION SWITCHER MODAL INTERACTION (VIM-FIRST) ====================
		if m.isSessionSwitcherOpen {
			switch key {
			case "esc":
				m.isSessionSwitcherOpen = false
				return m, nil
			case "up", "ctrl+k", "ctrl+p":
				if m.switcherSelectedIdx > 0 {
					m.switcherSelectedIdx--
				}
				return m, nil
			case "down", "ctrl+j", "ctrl+n", "tab":
				if m.switcherSelectedIdx < len(m.filteredSessions)-1 {
					m.switcherSelectedIdx++
				}
				return m, nil
			case "shift+tab":
				if m.switcherSelectedIdx > 0 {
					m.switcherSelectedIdx--
				}
				return m, nil
			case "enter":
				if len(m.filteredSessions) > 0 && m.switcherSelectedIdx < len(m.filteredSessions) {
					targetSession := m.filteredSessions[m.switcherSelectedIdx].SessionID
					m.isSessionSwitcherOpen = false
					return m.Update(SwitchSessionReqMsg{SessionID: targetSession})
				}
				return m, nil
			case "backspace":
				if len(m.sessionSearchQuery) > 0 {
					m.sessionSearchQuery = m.sessionSearchQuery[:len(m.sessionSearchQuery)-1]
					m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery)
					m.switcherSelectedIdx = 0
				}
				return m, nil
			default:
				// If search query is empty, allow direct 'j' and 'k' navigation without typing
				if m.sessionSearchQuery == "" {
					if key == "j" {
						if m.switcherSelectedIdx < len(m.filteredSessions)-1 {
							m.switcherSelectedIdx++
						}
						return m, nil
					} else if key == "k" {
						if m.switcherSelectedIdx > 0 {
							m.switcherSelectedIdx--
						}
						return m, nil
					}
				}

				// Type characters to filter sessions
				if len(msg.Runes) > 0 {
					m.sessionSearchQuery += string(msg.Runes)
					m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery)
					m.switcherSelectedIdx = 0
					return m, nil
				}
			}
			return m, nil
		}

		// ==================== GLOBAL VIEW SWITCHING ====================
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1", "d":
			m.activeView = ViewDashboard
			return m, nil
		case "2", "s":
			m.activeView = ViewHistory
			m.focusPane = FocusList
			return m, nil
		}

		// ==================== DASHBOARD VIEW KEYBINDINGS ====================
		if m.activeView == ViewDashboard {
			switch key {
			case "enter":
				if len(m.history) > 0 {
					m.selectedIdx = len(m.history) - 1 - m.dashboardIdx
					if m.selectedIdx < 0 {
						m.selectedIdx = 0
					}
					if m.selectedIdx >= len(m.history) {
						m.selectedIdx = len(m.history) - 1
					}
					maxCards := (m.height - 5) / 2
					if maxCards < 1 {
						maxCards = 1
					}
					m.historyOffset = m.selectedIdx - (maxCards / 2)
					if m.historyOffset < 0 {
						m.historyOffset = 0
					}
					m.detailScroll = 0
					m.isVisualMode = false
					m.activeView = ViewHistory
					m.focusPane = FocusList
				}
				return m, nil
			case "up", "k":
				if m.dashboardIdx > 0 {
					m.dashboardIdx--
				}
			case "down", "j":
				if m.dashboardIdx < len(m.history)-1 {
					m.dashboardIdx++
				}
			case "ctrl+u":
				m.dashboardIdx -= 10
				if m.dashboardIdx < 0 {
					m.dashboardIdx = 0
				}
			case "ctrl+d":
				m.dashboardIdx += 10
				if m.dashboardIdx >= len(m.history) {
					m.dashboardIdx = len(m.history) - 1
				}
			case "home", "g":
				m.dashboardIdx = 0
			case "end", "G":
				if len(m.history) > 0 {
					m.dashboardIdx = len(m.history) - 1
				}
			}
			return m, nil
		}

		// ==================== HISTORY EXPLORER VIEW KEYBINDINGS ====================
		if m.activeView == ViewHistory {
			leftOuterWidth := int(float64(m.width) * 0.32)
			detailInnerWidth := m.width - leftOuterWidth - 4 - 2

			selectedEvent, hasEvent := m.getSelectedEvent()
			var fullLines []string
			if hasEvent {
				fullLines = m.buildFullInspectorLines(selectedEvent, detailInnerWidth)
			}
			totalInspectorLines := len(fullLines)

			availableLines := m.height - 5
			if availableLines < 1 {
				availableLines = 1
			}
			maxScroll := totalInspectorLines - availableLines
			if maxScroll < 0 {
				maxScroll = 0
			}

			maxVisibleCards := (m.height - 5) / 2
			if maxVisibleCards < 1 {
				maxVisibleCards = 1
			}

			if m.focusPane == FocusDetail {
				if key == "v" || key == "V" {
					m.isVisualMode = !m.isVisualMode
					if m.isVisualMode {
						m.visualStart = m.detailScroll
						m.visualCursor = m.detailScroll
					}
					return m, nil
				}

				if m.isVisualMode {
					switch key {
					case "esc":
						m.isVisualMode = false
						return m, nil
					case "y":
						start := m.visualStart
						end := m.visualCursor
						if start > end {
							start, end = end, start
						}
						if start < 0 {
							start = 0
						}
						if end >= totalInspectorLines {
							end = totalInspectorLines - 1
						}
						if len(fullLines) > 0 && start <= end {
							selectedText := strings.Join(fullLines[start:end+1], "\n")
							_ = CopyToClipboard(selectedText)
							m.clipboardStatus = fmt.Sprintf("Copied %d lines to clipboard", end-start+1)
							m.clipboardStatusTime = time.Now()
						}
						m.isVisualMode = false
						return m, nil
					case "up", "k":
						if m.visualCursor > 0 {
							m.visualCursor--
							if m.visualCursor < m.detailScroll {
								m.detailScroll = m.visualCursor
							}
						}
						return m, nil
					case "down", "j":
						if m.visualCursor < totalInspectorLines-1 {
							m.visualCursor++
							if m.visualCursor >= m.detailScroll+availableLines {
								m.detailScroll = m.visualCursor - availableLines + 1
							}
						}
						return m, nil
					case "ctrl+u":
						m.visualCursor -= 10
						if m.visualCursor < 0 {
							m.visualCursor = 0
						}
						if m.visualCursor < m.detailScroll {
							m.detailScroll = m.visualCursor
						}
						return m, nil
					case "ctrl+d":
						m.visualCursor += 10
						if m.visualCursor >= totalInspectorLines {
							m.visualCursor = totalInspectorLines - 1
						}
						if m.visualCursor >= m.detailScroll+availableLines {
							m.detailScroll = m.visualCursor - availableLines + 1
						}
						return m, nil
					}
				}
			}

			switch key {
			case "tab":
				if m.focusPane == FocusList {
					m.focusPane = FocusDetail
				} else {
					m.focusPane = FocusList
					m.isVisualMode = false
				}
			case "enter", "right", "l":
				if m.focusPane == FocusList {
					m.focusPane = FocusDetail
				}
			case "esc", "left":
				if m.focusPane == FocusDetail {
					m.focusPane = FocusList
					m.isVisualMode = false
				}
			case "up", "k":
				if m.focusPane == FocusList {
					if m.selectedIdx > 0 {
						m.selectedIdx--
						m.detailScroll = 0
						if m.selectedIdx < m.historyOffset {
							m.historyOffset = m.selectedIdx
						}
					}
				} else {
					if m.detailScroll > 0 {
						m.detailScroll--
					}
				}
			case "down", "j":
				if m.focusPane == FocusList {
					if m.selectedIdx < len(m.history)-1 {
						m.selectedIdx++
						m.detailScroll = 0
						if m.selectedIdx >= m.historyOffset+maxVisibleCards {
							m.historyOffset = m.selectedIdx - maxVisibleCards + 1
						}
					}
				} else {
					if m.detailScroll < maxScroll {
						m.detailScroll++
					}
				}
			case "ctrl+u":
				if m.focusPane == FocusDetail {
					m.detailScroll -= 10
					if m.detailScroll < 0 {
						m.detailScroll = 0
					}
				} else if m.focusPane == FocusList {
					m.selectedIdx -= maxVisibleCards
					if m.selectedIdx < 0 {
						m.selectedIdx = 0
					}
					m.detailScroll = 0
					if m.selectedIdx < m.historyOffset {
						m.historyOffset = m.selectedIdx
					}
				}
			case "ctrl+d":
				if m.focusPane == FocusDetail {
					m.detailScroll += 10
					if m.detailScroll > maxScroll {
						m.detailScroll = maxScroll
					}
				} else if m.focusPane == FocusList {
					m.selectedIdx += maxVisibleCards
					if m.selectedIdx >= len(m.history) {
						m.selectedIdx = len(m.history) - 1
					}
					m.detailScroll = 0
					if m.selectedIdx >= m.historyOffset+maxVisibleCards {
						m.historyOffset = m.selectedIdx - maxVisibleCards + 1
					}
				}
			case "ctrl+b", "pgup":
				if m.focusPane == FocusDetail {
					m.detailScroll -= availableLines
					if m.detailScroll < 0 {
						m.detailScroll = 0
					}
				}
			case "ctrl+f", "pgdown", " ":
				if m.focusPane == FocusDetail {
					m.detailScroll += availableLines
					if m.detailScroll > maxScroll {
						m.detailScroll = maxScroll
					}
				}
			case "home", "g":
				if m.focusPane == FocusList {
					m.selectedIdx = 0
					m.historyOffset = 0
					m.detailScroll = 0
				} else {
					m.detailScroll = 0
				}
			case "end", "G":
				if m.focusPane == FocusList {
					if len(m.history) > 0 {
						m.selectedIdx = len(m.history) - 1
						m.detailScroll = 0
						if len(m.history) > maxVisibleCards {
							m.historyOffset = len(m.history) - maxVisibleCards
						}
					}
				} else {
					m.detailScroll = maxScroll
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width <= 0 {
		m.width = 80
	}
	if m.height <= 0 {
		m.height = 24
	}

	if m.isGlossaryModalOpen {
		return m.renderGlossaryModal()
	}

	if m.isHelpModalOpen {
		return m.renderHelpModal()
	}

	if m.isSessionSwitcherOpen {
		return m.renderSessionSwitcherModal()
	}

	var content string
	switch m.activeView {
	case ViewHistory:
		content = m.renderHistoryView()
	default:
		content = m.renderDashboardView()
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	fullView := lipgloss.JoinVertical(lipgloss.Left, header, content, footer)

	lines := strings.Split(fullView, "\n")
	if len(lines) > m.height && m.height > 0 {
		lines = lines[:m.height]
	}
	for i, line := range lines {
		if lipgloss.Width(line) > m.width && m.width > 0 {
			lines[i] = lipgloss.NewStyle().MaxWidth(m.width).Render(line)
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderHeader() string {
	title := HeaderStyle.Render(" AGENT-OBSERVER ")
	if m.width < 90 {
		title = HeaderStyle.Render(" OBSERVER ")
	}

	viewTabs := ""
	if m.activeView == ViewDashboard {
		viewTabs = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render(" [1] Dashboard ") +
			lipgloss.NewStyle().Foreground(ColorMuted).Render(" [2] History ")
	} else {
		viewTabs = lipgloss.NewStyle().Foreground(ColorMuted).Render(" [1] Dashboard ") +
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render(" [2] History ")
	}

	sessionBadge := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(fmt.Sprintf(" [Ctrl+P] %s ", truncateStr(m.sessionID, 8)))

	statusText := fmt.Sprintf("Events: %d | %s", m.eventCount, time.Now().Format("15:04:05"))
	status := lipgloss.NewStyle().Foreground(ColorLightText).Render(statusText)

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, viewTabs, sessionBadge)
	gapWidth := m.width - lipgloss.Width(left) - lipgloss.Width(status) - 1
	if gapWidth < 1 {
		gapWidth = 1
	}
	gap := strings.Repeat(" ", gapWidth)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Center, left, gap, status)
	return lipgloss.NewStyle().MaxWidth(m.width).Render(headerLine)
}

func (m Model) renderFooter() string {
	if m.clipboardStatus != "" && time.Since(m.clipboardStatusTime) < 4*time.Second {
		alert := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorSuccess).Padding(0, 1).Render(m.clipboardStatus)
		return lipgloss.NewStyle().MaxWidth(m.width).Render(alert)
	}

	var hints string
	if m.activeView == ViewHistory {
		if m.isVisualMode {
			hints = fmt.Sprintf(" %s Yank  %s Move  %s Cancel",
				KeyStyle.Render("[y]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[Esc]"))
		} else if m.focusPane == FocusList {
			hints = fmt.Sprintf(" %s Focus Detail  %s Select  %s Glossary  %s Help  %s Switch  %s Dash  %s Quit",
				KeyStyle.Render("[Tab/l]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[h]"), KeyStyle.Render("[?]"), KeyStyle.Render("[Ctrl+p]"), KeyStyle.Render("[1]"), KeyStyle.Render("[q]"))
		} else {
			hints = fmt.Sprintf(" %s Visual  %s Focus List  %s Scroll  %s Glossary  %s Help  %s Switch  %s Quit",
				KeyStyle.Render("[v]"), KeyStyle.Render("[Tab/Esc]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[h]"), KeyStyle.Render("[?]"), KeyStyle.Render("[Ctrl+p]"), KeyStyle.Render("[q]"))
		}
	} else {
		if len(m.history) > 0 {
			hints = fmt.Sprintf(" %s Inspect  %s Playback  %s Glossary  %s Help  %s Switch  %s LIVE  %s Quit",
				KeyStyle.Render("[Enter]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[h]"), KeyStyle.Render("[?]"), KeyStyle.Render("[Ctrl+p]"), KeyStyle.Render("[G]"), KeyStyle.Render("[q]"))
		} else {
			hints = fmt.Sprintf(" %s Glossary  %s Help  %s Switch  %s History  %s Quit",
				KeyStyle.Render("[h]"), KeyStyle.Render("[?]"), KeyStyle.Render("[Ctrl+p]"), KeyStyle.Render("[2]"), KeyStyle.Render("[q]"))
		}
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(hints)
}

// wrapVisualLines wraps text into slices where each chunk strictly has visual terminal column width <= maxWidth.
func wrapVisualLines(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{""}
	}
	var result []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.ReplaceAll(line, "\r", "")
		line = strings.ReplaceAll(line, "\t", "    ")
		if line == "" {
			result = append(result, "")
			continue
		}
		runes := []rune(line)
		var currentChunk []rune
		currentW := 0
		for _, r := range runes {
			rw := runewidth.RuneWidth(r)
			if currentW+rw > maxWidth {
				if len(currentChunk) > 0 {
					result = append(result, string(currentChunk))
					currentChunk = nil
					currentW = 0
				}
			}
			currentChunk = append(currentChunk, r)
			currentW += rw
		}
		if len(currentChunk) > 0 || len(runes) == 0 {
			result = append(result, string(currentChunk))
		}
	}
	return result
}
