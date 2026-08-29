package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"heimdall/internal/adapters/antigravity"
	"heimdall/internal/core"
)

type ActiveView int

const (
	ViewDashboard ActiveView = iota
	ViewHistory
	ViewContext
	ViewDocs
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
	isShortcutsModalOpen  bool
	sessionSearchQuery    string
	selectedAgentTab      core.AgentType
	availableSessions     []core.SessionInfo
	filteredSessions      []core.SessionInfo
	switcherSelectedIdx   int
	docsSearchQuery       string
	isDocsSearching       bool
	docsScroll            int
	docsLang              string
	historyTypeFilter     TypeFilter
	historyCacheFilter    CacheFilter
	historyStepQuery      string
	isHistorySearching    bool
	historySearchErr      string
	contextSubItemIndex   int
	isContextRawMode      bool
	contextFocusPane      FocusPane
	contextDetailScroll   int
	contextTargetAgent    string
	isCommandMode         bool
	commandInput          string
	statusMessage         string
	statusMessageTime     time.Time
	switcher              SessionSwitcher
}

// TypeFilter defines step category filter in History Explorer
type TypeFilter string

const (
	TypeFilterAll     TypeFilter = "All"
	TypeFilterTool    TypeFilter = "Tool"
	TypeFilterModel   TypeFilter = "Model"
	TypeFilterUser    TypeFilter = "User"
	TypeFilterCode    TypeFilter = "Code"
	TypeFilterGeneric TypeFilter = "Generic"
)

// CacheFilter defines cache status filter in History Explorer (aligned with Docs)
type CacheFilter string

const (
	CacheFilterAll     CacheFilter = "All"
	CacheFilterHit     CacheFilter = "Hit"
	CacheFilterPartial CacheFilter = "Partial"
	CacheFilterWrite   CacheFilter = "Write"
	CacheFilterExpired CacheFilter = "Expired"
	CacheFilterMiss    CacheFilter = "Miss"
)

// nextAgentTab cycles agent filter tab: Antigravity -> ClaudeCode -> OpenCode -> Antigravity
func (m *Model) nextAgentTab() {
	switch m.selectedAgentTab {
	case core.AgentTypeClaudeCode:
		m.selectedAgentTab = core.AgentTypeOpenCode
	case core.AgentTypeOpenCode:
		m.selectedAgentTab = core.AgentTypeAntigravity
	default:
		m.selectedAgentTab = core.AgentTypeClaudeCode
	}
	m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery, m.selectedAgentTab)
	m.switcherSelectedIdx = 0
}

// prevAgentTab cycles agent filter tab: Antigravity -> OpenCode -> ClaudeCode -> Antigravity
func (m *Model) prevAgentTab() {
	switch m.selectedAgentTab {
	case core.AgentTypeClaudeCode:
		m.selectedAgentTab = core.AgentTypeAntigravity
	case core.AgentTypeOpenCode:
		m.selectedAgentTab = core.AgentTypeClaudeCode
	default:
		m.selectedAgentTab = core.AgentTypeOpenCode
	}
	m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery, m.selectedAgentTab)
	m.switcherSelectedIdx = 0
}

func (m *Model) cycleTypeFilter() {
	switch m.historyTypeFilter {
	case TypeFilterAll:
		m.historyTypeFilter = TypeFilterTool
	case TypeFilterTool:
		m.historyTypeFilter = TypeFilterModel
	case TypeFilterModel:
		m.historyTypeFilter = TypeFilterUser
	case TypeFilterUser:
		m.historyTypeFilter = TypeFilterCode
	case TypeFilterCode:
		m.historyTypeFilter = TypeFilterGeneric
	default:
		m.historyTypeFilter = TypeFilterAll
	}
	m.selectedIdx = 0
	m.historyOffset = 0
	m.detailScroll = 0
}

func (m *Model) cycleCacheFilter() {
	switch m.historyCacheFilter {
	case CacheFilterAll:
		m.historyCacheFilter = CacheFilterHit
	case CacheFilterHit:
		m.historyCacheFilter = CacheFilterPartial
	case CacheFilterPartial:
		m.historyCacheFilter = CacheFilterWrite
	case CacheFilterWrite:
		m.historyCacheFilter = CacheFilterExpired
	case CacheFilterExpired:
		m.historyCacheFilter = CacheFilterMiss
	default:
		m.historyCacheFilter = CacheFilterAll
	}
	m.selectedIdx = 0
	m.historyOffset = 0
	m.detailScroll = 0
}

func (m Model) getFilteredHistory() []core.UnifiedAgentEvent {
	if m.historyTypeFilter == TypeFilterAll && m.historyCacheFilter == CacheFilterAll {
		return m.history
	}

	var result []core.UnifiedAgentEvent
	for _, e := range m.history {
		if !matchTypeFilter(e.Type, m.historyTypeFilter) {
			continue
		}
		if !matchCacheFilter(e, m.historyCacheFilter) {
			continue
		}
		result = append(result, e)
	}
	return result
}

func matchTypeFilter(stepType core.StepType, filter TypeFilter) bool {
	switch filter {
	case TypeFilterAll:
		return true
	case TypeFilterTool:
		return stepType == core.StepTypeToolCall || stepType == core.StepTypeToolResult || stepType == core.StepTypeRunCommand || stepType == core.StepTypeViewFile || stepType == core.StepTypeListDirectory || stepType == core.StepTypeAskQuestion
	case TypeFilterModel:
		return stepType == core.StepTypeModelResponse
	case TypeFilterUser:
		return stepType == core.StepTypeUserInput
	case TypeFilterCode:
		return stepType == core.StepTypeCodeAction
	case TypeFilterGeneric:
		return stepType == core.StepTypeSystemInit || stepType == core.StepTypeUnknown || stepType == ""
	}
	return true
}

func matchCacheFilter(e core.UnifiedAgentEvent, filter CacheFilter) bool {
	if filter == CacheFilterAll {
		return true
	}
	// Only cloud inference steps have GPU KV-Cache telemetry and participate in cache filtering!
	if !e.IsCloudStep() {
		return false
	}
	status := e.CacheStatus
	if status == "" {
		status = core.ClassifyCacheStatus(e.Tokens.CacheHitRate, e.Tokens.CachedTokens, e.Tokens.TotalTokens, false)
	}
	switch filter {
	case CacheFilterHit:
		return status == "HIT"
	case CacheFilterPartial:
		return status == "PARTIAL"
	case CacheFilterWrite:
		return status == "WRITE" || (e.StepIndex == 0 && e.Tokens.CachedTokens == 0)
	case CacheFilterExpired:
		return status == "EXPIRED" || status == "TTL_EXPIRED"
	case CacheFilterMiss:
		return status == "MISS"
	}
	return true
}

// nextView cycles active view clockwise: Dashboard (1) -> History (2) -> Context (3) -> Docs (4) -> Dashboard (1)
func (m *Model) nextView() {
	m.isDocsSearching = false
	m.isHistorySearching = false
	switch m.activeView {
	case ViewDashboard:
		m.activeView = ViewHistory
		m.focusPane = FocusList
	case ViewHistory:
		m.activeView = ViewContext
		m.contextFocusPane = FocusList
	case ViewContext:
		m.activeView = ViewDocs
	case ViewDocs:
		m.activeView = ViewDashboard
	}
}

// prevView cycles active view counter-clockwise: Dashboard -> Docs -> Context -> History -> Dashboard
func (m *Model) prevView() {
	m.isDocsSearching = false
	m.isHistorySearching = false
	switch m.activeView {
	case ViewDashboard:
		m.activeView = ViewDocs
	case ViewHistory:
		m.activeView = ViewDashboard
	case ViewContext:
		m.activeView = ViewHistory
		m.focusPane = FocusList
	case ViewDocs:
		m.activeView = ViewContext
		m.contextFocusPane = FocusList
	}
}

// NewModel creates an initial TUI model
func NewModel(sessionID string, openSwitcherOnStart bool, switcher ...SessionSwitcher) Model {
	sessions, _ := antigravity.DiscoverAllSessions()
	initialTab := core.AgentTypeAntigravity

	var sw SessionSwitcher
	if len(switcher) > 0 {
		sw = switcher[0]
	}

	var initialHistory []core.UnifiedAgentEvent
	if sessionID != "" {
		analyzer := core.NewPayloadAnalyzer()
		if hist, err := antigravity.LoadSessionHistory(sessionID, analyzer); err == nil {
			initialHistory = hist
		}
	}
	if initialHistory == nil {
		initialHistory = make([]core.UnifiedAgentEvent, 0, 1000)
	}

	var latestEvent core.UnifiedAgentEvent
	dashboardIdx := 0
	if len(initialHistory) > 0 {
		latestEvent = initialHistory[len(initialHistory)-1]
		dashboardIdx = len(initialHistory) - 1
	}

	m := Model{
		sessionID:             sessionID,
		activeView:            ViewDashboard,
		focusPane:             FocusList,
		dashboardIdx:          dashboardIdx,
		detailScroll:          0,
		history:               initialHistory,
		latestEvent:           latestEvent,
		eventCount:            len(initialHistory),
		selectedIdx:           0,
		lastActivity:          time.Now(),
		selectedAgentTab:      initialTab,
		availableSessions:     sessions,
		filteredSessions:      filterSessions(sessions, "", initialTab),
		switcherSelectedIdx:   0,
		isSessionSwitcherOpen: openSwitcherOnStart,
		isShortcutsModalOpen:  false,
		docsSearchQuery:       "",
		isDocsSearching:       false,
		docsScroll:            0,
		docsLang:              "en",
		historyTypeFilter:     TypeFilterAll,
		historyCacheFilter:    CacheFilterAll,
		historyStepQuery:      "",
		isHistorySearching:    false,
		switcher:              sw,
	}

	// If sessionID matches one of discovered sessions, select it in the switcher
	for i, s := range m.filteredSessions {
		if s.SessionID == sessionID {
			m.switcherSelectedIdx = i
			break
		}
	}

	return m
}

func (m Model) getHistoryVisibleCards() int {
	var availLines int
	if m.width < 100 {
		bodyHeight := m.height - 2
		if bodyHeight < 8 {
			bodyHeight = 8
		}
		topContentRows := (bodyHeight - 4) * 45 / 100
		if topContentRows < 11 {
			topContentRows = 11
		}
		availLines = topContentRows - 2
	} else {
		innerRowsLimit := m.height - 4
		if innerRowsLimit < 4 {
			innerRowsLimit = 4
		}
		availLines = innerRowsLimit - 2
	}

	if m.isHistorySearching || m.historyStepQuery != "" {
		availLines -= 2
	}

	if m.historyOffset > 0 {
		availLines--
	}
	if availLines < 1 {
		availLines = 1
	}

	filtered := m.getFilteredHistory()
	if len(filtered) == 0 {
		return 1
	}

	usedLines := 0
	cardCount := 0
	for i := m.historyOffset; i < len(filtered); i++ {
		realIdx := len(filtered) - 1 - i
		e := filtered[realIdx]
		linesNeeded := 1
		isLocal := e.IsLocalStep() || e.Scope == core.ScopeLocalExecution
		if isLocal {
			toolName := m.getLocalToolName(e)
			if toolName != "" && toolName != "OUTPUT" {
				linesNeeded = 2
			}
		} else {
			modelName := m.getStepModelName(e)
			if modelName != "" {
				linesNeeded = 2
			}
		}

		if i < len(filtered)-1 {
			// Reserve 1 line for bottom "..." if there are more cards
			if usedLines+linesNeeded > (availLines - 1) && cardCount > 0 {
				break
			}
		} else {
			if usedLines+linesNeeded > availLines && cardCount > 0 {
				break
			}
		}

		usedLines += linesNeeded
		cardCount++
	}

	if cardCount < 1 {
		cardCount = 1
	}
	return cardCount
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) getSelectedEvent() (core.UnifiedAgentEvent, bool) {
	filtered := m.getFilteredHistory()
	if len(filtered) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(filtered) {
		return core.UnifiedAgentEvent{}, false
	}
	realIdx := len(filtered) - 1 - m.selectedIdx
	if realIdx < 0 || realIdx >= len(filtered) {
		return core.UnifiedAgentEvent{}, false
	}
	return filtered[realIdx], true
}

func (m Model) getSessionModelName() string {
	for _, s := range m.availableSessions {
		if s.SessionID == m.sessionID && s.ModelName != "" {
			return s.ModelName
		}
	}
	return "gemini-3.7-flash"
}

func (m *Model) jumpToStep(targetStepIdx int) bool {
	filtered := m.getFilteredHistory()
	for i, e := range filtered {
		if e.StepIndex == targetStepIdx {
			m.selectedIdx = len(filtered) - 1 - i
			m.detailScroll = 0

			visCards := m.getHistoryVisibleCards()
			if m.selectedIdx < m.historyOffset {
				m.historyOffset = m.selectedIdx
			} else if m.selectedIdx >= m.historyOffset+visCards {
				m.historyOffset = m.selectedIdx - visCards + 1
				if m.historyOffset < 0 {
					m.historyOffset = 0
				}
			}
			return true
		}
	}
	return false
}

func (m *Model) jumpToParent(curr core.UnifiedAgentEvent) bool {
	if curr.ParentStepIdx > 0 {
		return m.jumpToStep(curr.ParentStepIdx)
	}
	for i := len(m.history) - 1; i >= 0; i-- {
		if m.history[i].StepIndex < curr.StepIndex && (m.history[i].Type == core.StepTypeToolCall || m.history[i].Type == core.StepTypeUserInput) {
			return m.jumpToStep(m.history[i].StepIndex)
		}
	}
	return false
}

func (m *Model) jumpToChildOrNext(curr core.UnifiedAgentEvent) bool {
	if len(curr.ConsumedStepIndices) > 0 {
		return m.jumpToStep(curr.ConsumedStepIndices[0])
	}
	if curr.PackagedInStepIdx > 0 {
		return m.jumpToStep(curr.PackagedInStepIdx)
	}
	for _, e := range m.history {
		if e.ParentStepIdx == curr.StepIndex {
			return m.jumpToStep(e.StepIndex)
		}
	}
	if curr.Type == core.StepTypeToolCall {
		return m.jumpToStep(curr.StepIndex + 1)
	}
	for _, e := range m.history {
		if e.StepIndex > curr.StepIndex && (e.Type == core.StepTypeToolCall || e.Type == core.StepTypeModelResponse) {
			return m.jumpToStep(e.StepIndex)
		}
	}
	return false
}

func formatCompactNumber(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000.0)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}

func (m Model) buildTelemetryPanelLines(e core.UnifiedAgentEvent, maxWidth int, isCompact bool) []string {
	if maxWidth <= 10 {
		maxWidth = 40
	}
	var lines []string
	t := e.Tokens
	timeStr := e.Timestamp.Local().Format("15:04:05")

	if isCompact {
		// ==================== COMPACT MODE (STRUCTURED SUB-BULLETS, ZERO TRUNCATION) ====================
		if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			toolName := m.getLocalToolName(e)
			if toolName == "" {
				toolName = string(e.Type)
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Action: Tool Output (%s)", toolName), maxWidth))
			lines = append(lines, truncateVisualWidth("• Status: Offline Process (0 tok)", maxWidth))
			if e.PackagedInStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("  └ Billed: Packaged in #%04d", e.PackagedInStepIdx), maxWidth))
			} else {
				estTok := t.ActiveTurnTokens + t.ToolResultTokens
				if estTok == 0 {
					estTok = core.CountTokens(e.RawContent)
				}
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("  └ Staged: ~%s tok (Pending)", formatCompactNumber(estTok)), maxWidth))
			}
			if e.ParentStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Parent: Triggered by #%04d", e.ParentStepIdx), maxWidth))
			}
			lines = append(lines, truncateVisualWidth("• Origin: Local Machine Subprocess", maxWidth))
		} else if e.IsCloudStep() || e.Scope == core.ScopeCloudInference {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			modelName := m.getStepModelName(e)
			if modelName == "" {
				modelName = "Gemini 3.7 Flash"
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Model : %s", modelName), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Tokens: %s Total Context", formatCompactNumber(t.TotalTokens)), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("  ├ Cached: %s (%.1f%% HIT)", formatCompactNumber(t.CachedTokens), t.CacheHitRate), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("  └ New   : %s new tokens", formatCompactNumber(t.NewTokens)), maxWidth))
			lines = append(lines, truncateVisualWidth("• 5-Dims:", maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("  ├ Sys: %s | Tools: %s", formatCompactNumber(t.SystemTokens), formatCompactNumber(t.ToolsDefTokens)), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("  └ Hist: %s | Act: %s", formatCompactNumber(t.HistoryTokens), formatCompactNumber(t.ActiveTurnTokens)), maxWidth))
			if e.ParentStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Parent: Step #%04d (User Prompt)", e.ParentStepIdx), maxWidth))
			}
			if len(e.ToolCalls) > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Tool  : %s (%d call)", e.ToolCalls[0].ToolName, len(e.ToolCalls)), maxWidth))
			}
		} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			lines = append(lines, truncateVisualWidth("• Origin: Human Client Prompt", maxWidth))
			promptTok := core.CountTokens(e.RawContent)
			if promptTok == 0 {
				promptTok = 1
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Payload: ~%d Prompt Tokens", promptTok), maxWidth))
			if e.PackagedInStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Status : Staged ➔ Settled in Step #%04d", e.PackagedInStepIdx), maxWidth))
			} else {
				lines = append(lines, truncateVisualWidth("• Status : Inbound (Awaiting Cloud Turn)", maxWidth))
			}
		} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			lines = append(lines, truncateVisualWidth("• Event : Context Compaction (Checkpoint)", maxWidth))
			lines = append(lines, truncateVisualWidth("• Scope : Harness Middleware (Sidecar GC)", maxWidth))
			summaryTok := core.CountTokens(e.RawContent)
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Summary: ~%s Tok (Prunes Old Turns)", formatCompactNumber(summaryTok)), maxWidth))
			lines = append(lines, truncateVisualWidth("• Status: Active Context Re-anchored", maxWidth))
		} else {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Type  : %s", e.Type), maxWidth))
			lines = append(lines, truncateVisualWidth("• Scope : System Bootstrap & Rules", maxWidth))
		}
		return lines
	}

	// ==================== FULL-WIDTH MODE (DETAILED TELEMETRY) ====================
	if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | 💻 LOCAL EXECUTION STEP", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Origin : Local Host Process (%s)", e.Type), maxWidth))
		if e.ParentStepIdx > 0 {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Parent : Triggered by Tool Call in Step #%04d", e.ParentStepIdx), maxWidth))
		}
		if e.PackagedInStepIdx > 0 {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Billing: Offline (0 tok) ➔ Packaged in Step #%04d", e.PackagedInStepIdx), maxWidth))
		} else {
			estTok := t.ActiveTurnTokens + t.ToolResultTokens
			if estTok == 0 {
				estTok = core.CountTokens(e.RawContent)
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Billing: Offline (0 tok) ➔ Staged (~%d tok, Pending Next Turn ⏳)", estTok), maxWidth))
		}
	} else if e.IsCloudStep() || e.Scope == core.ScopeCloudInference {
		modelName := t.OfficialModel
		if modelName == "" {
			modelName = m.getSessionModelName()
		}
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | ☁️ CLOUD INFERENCE TURN", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Model  : %s (Official Telemetry)", modelName), maxWidth))
		if e.ParentStepIdx > 0 {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Parent : User Request in Step #%04d", e.ParentStepIdx), maxWidth))
		}
		if len(e.ConsumedStepIndices) > 0 {
			var childStrs []string
			for _, c := range e.ConsumedStepIndices {
				childStrs = append(childStrs, fmt.Sprintf("#%04d", c))
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Input  : Consumed Local Step %s", strings.Join(childStrs, ", ")), maxWidth))
		}
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Tokens : Total: %d | Cached: %d (%.1f%% HIT) | New: %d", t.TotalTokens, t.CachedTokens, t.CacheHitRate, t.NewTokens), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• 5-Dims : Sys=%d | Tools=%d | Res=%d | Hist=%d | Act=%d", t.SystemTokens, t.ToolsDefTokens, t.ToolResultTokens, t.HistoryTokens, t.ActiveTurnTokens), maxWidth))
	} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | 👤 USER INPUT", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth("• Origin : Human Client Prompt (Inbound to Remote GPU Cluster)", maxWidth))
		promptTok := core.CountTokens(e.RawContent)
		if promptTok == 0 {
			promptTok = 1
		}
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Payload: ~%d Prompt Tokens (Local Inbound Intent)", promptTok), maxWidth))
		if e.PackagedInStepIdx > 0 {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Billing: Staged ➔ Official Cache & Tokens settled in Step #%04d ☁️", e.PackagedInStepIdx), maxWidth))
		} else {
			lines = append(lines, truncateVisualWidth("• Billing: Staged ➔ Awaiting next Cloud Inference Step for official cache settlement ⏳", maxWidth))
		}
	} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | ⚙️ CONTEXT COMPACTION (CHECKPOINT)", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth("• Event  : Background AI Compaction & Truncation Injection", maxWidth))
		lines = append(lines, truncateVisualWidth("• Scope  : Harness Middleware (Sidecar Context GC)", maxWidth))
		summaryTok := core.CountTokens(e.RawContent)
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Payload: ~%d Summary Tokens (Replaces ~200k+ historical context)", summaryTok), maxWidth))
		lines = append(lines, truncateVisualWidth("• Action : Prepend [System + Tools + Checkpoint Summary] ➔ Re-anchor Window Base", maxWidth))
	} else {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | 📜 SYSTEM BOOTSTRAP", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth("• Scope  : Static Rules, Identity & Environment Configuration", maxWidth))
	}

	if len(e.ToolCalls) > 0 {
		for _, tc := range e.ToolCalls {
			tcText := fmt.Sprintf("• Tool   : %s (args: %v)", tc.ToolName, tc.Arguments)
			lines = append(lines, truncateVisualWidth(tcText, maxWidth))
		}
	}

	return lines
}

func (m Model) buildContentPayloadLines(e core.UnifiedAgentEvent, maxWidth int) []string {
	if maxWidth <= 10 {
		maxWidth = 60
	}
	var lines []string
	sepWidth := maxWidth
	if sepWidth > 80 {
		sepWidth = 80
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

func (m Model) buildFullInspectorLines(e core.UnifiedAgentEvent, maxWidth int) []string {
	telemetry := m.buildTelemetryPanelLines(e, maxWidth, false)
	payload := m.buildContentPayloadLines(e, maxWidth)
	var all []string
	all = append(all, telemetry...)
	all = append(all, strings.Repeat("─", maxWidth))
	all = append(all, payload...)
	return all
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case AgentEventMsg:
		event := core.UnifiedAgentEvent(msg)

		// 🛡️ Drop stale events that do not belong to the currently active session
		if event.SessionID != m.sessionID && m.sessionID != "" && event.SessionID != "" {
			return m, nil
		}

		m.latestEvent = event
		m.eventCount++
		m.lastActivity = time.Now()
		wasAtLatestDashboard := (m.dashboardIdx == len(m.history)-1 || len(m.history) == 0)
		isInspectingPastStep := (m.activeView == ViewHistory && (m.selectedIdx > 0 || m.focusPane == FocusDetail))

		// If this is a cloud step consuming previous local steps, update their PackagedInStepIdx in m.history!
		if len(event.ConsumedStepIndices) > 0 {
			for _, childIdx := range event.ConsumedStepIndices {
				for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
					if m.history[hIdx].StepIndex == childIdx {
						m.history[hIdx].PackagedInStepIdx = event.StepIndex
						break
					}
				}
			}
		}

		// Update in-place if this step index already exists in history (e.g. streaming update or status transition), otherwise append
		existingIdx := -1
		for i := len(m.history) - 1; i >= 0; i-- {
			if m.history[i].StepIndex == event.StepIndex && m.history[i].SessionID == event.SessionID {
				existingIdx = i
				break
			}
		}

		if existingIdx >= 0 {
			m.history[existingIdx] = event
		} else {
			m.history = append(m.history, event)
		}

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

	case SessionResetMsg:
		m.sessionID = msg.SessionID
		m.history = make([]core.UnifiedAgentEvent, 0, 1000)
		m.eventCount = 0
		m.latestEvent = core.UnifiedAgentEvent{}
		m.dashboardIdx = 0
		m.selectedIdx = 0
		m.detailScroll = 0
		m.historyOffset = 0
		m.isSessionSwitcherOpen = false
		m.isShortcutsModalOpen = false
		m.activeView = ViewDashboard
		m.clipboardStatus = fmt.Sprintf("Live attached to %s", truncateStr(msg.SessionID, 8))
		m.clipboardStatusTime = time.Now()
		return m, nil

	case SwitchSessionReqMsg:
		analyzer := core.NewPayloadAnalyzer()
		events, _ := antigravity.LoadSessionHistory(msg.SessionID, analyzer)

		if m.switcher != nil {
			_ = m.switcher.SwitchSession(msg.SessionID, msg.AgentType)
		}
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
		m.isShortcutsModalOpen = false
		m.activeView = ViewDashboard
		m.clipboardStatus = fmt.Sprintf("Attached session %s (%d steps, live)", truncateStr(msg.SessionID, 8), len(events))
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
		m.isShortcutsModalOpen = false
		m.activeView = ViewDashboard
		m.clipboardStatus = fmt.Sprintf("Switched to session %s (%d steps)", truncateStr(msg.SessionID, 8), len(msg.Events))
		m.clipboardStatusTime = time.Now()
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// ==================== VIM COMMAND MODE (:q, :w, :help) ====================
		if m.isCommandMode {
			switch key {
			case "esc":
				m.isCommandMode = false
				m.commandInput = ""
				return m, nil
			case "backspace":
				if len(m.commandInput) > 0 {
					m.commandInput = m.commandInput[:len(m.commandInput)-1]
				} else {
					m.isCommandMode = false
				}
				return m, nil
			case "enter":
				cmd := strings.TrimSpace(m.commandInput)
				m.isCommandMode = false
				m.commandInput = ""
				switch cmd {
				case "q", "quit", "q!", "exit":
					return m, tea.Quit
				case "w", "write":
					payload := core.ExtractAgentContextPayload(m.history, m.sessionID, "Gemini 3.7 Flash")
					content := m.GetContextInspectorContent(payload)
					_ = os.WriteFile("context_payload_export.json", []byte(content), 0644)
					m.statusMessage = "💾 Saved payload to context_payload_export.json!"
					m.statusMessageTime = time.Now()
					return m, nil
				case "h", "help", "?":
					m.isShortcutsModalOpen = true
					return m, nil
				default:
					if cmd != "" {
						m.statusMessage = fmt.Sprintf("⚠️ Unknown command ':%s' (type :q to quit)", cmd)
						m.statusMessageTime = time.Now()
					}
					return m, nil
				}
			default:
				if len(key) == 1 {
					m.commandInput += key
				}
				return m, nil
			}
		}

		// Global toggle for Keyboard Shortcuts Modal: ? or F1
		if key == "?" || key == "f1" {
			if !m.isSessionSwitcherOpen && !m.isDocsSearching {
				m.isShortcutsModalOpen = !m.isShortcutsModalOpen
				return m, nil
			}
		}

		// Shortcuts Modal Interaction
		if m.isShortcutsModalOpen {
			if key == "esc" || key == "?" || key == "q" || key == "enter" {
				m.isShortcutsModalOpen = false
				return m, nil
			}
			return m, nil
		}

		// Global toggle for session switcher modal: Ctrl+P
		if key == "ctrl+p" {
			m.isSessionSwitcherOpen = !m.isSessionSwitcherOpen
			if m.isSessionSwitcherOpen {
				m.isShortcutsModalOpen = false
				m.availableSessions, _ = antigravity.DiscoverAllSessions()
				m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery, m.selectedAgentTab)
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
			case "tab", "]":
				m.nextAgentTab()
				return m, nil
			case "shift+tab", "backtab", "[":
				m.prevAgentTab()
				return m, nil
			case "up", "ctrl+k":
				if m.switcherSelectedIdx > 0 {
					m.switcherSelectedIdx--
				}
				return m, nil
			case "down", "ctrl+j", "ctrl+n":
				if m.switcherSelectedIdx < len(m.filteredSessions)-1 {
					m.switcherSelectedIdx++
				}
				return m, nil
			case "enter":
				if len(m.filteredSessions) > 0 && m.switcherSelectedIdx < len(m.filteredSessions) {
					targetSession := m.filteredSessions[m.switcherSelectedIdx].SessionID
					targetAgent := m.filteredSessions[m.switcherSelectedIdx].AgentType
					m.isSessionSwitcherOpen = false
					return m.Update(SwitchSessionReqMsg{SessionID: targetSession, AgentType: targetAgent})
				}
				return m, nil
			case "backspace":
				if len(m.sessionSearchQuery) > 0 {
					m.sessionSearchQuery = m.sessionSearchQuery[:len(m.sessionSearchQuery)-1]
					m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery, m.selectedAgentTab)
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
					} else if key == "h" {
						m.prevAgentTab()
						return m, nil
					} else if key == "l" {
						m.nextAgentTab()
						return m, nil
					}
				}

				// Type characters to filter sessions
				if len(msg.Runes) > 0 {
					m.sessionSearchQuery += string(msg.Runes)
					m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery, m.selectedAgentTab)
					m.switcherSelectedIdx = 0
					return m, nil
				}
			}
			return m, nil
		}

		// ==================== VIEW 3: DOCS & GLOSSARY KEYBINDINGS ====================
		if m.activeView == ViewDocs {
			if m.isDocsSearching {
				switch key {
				case "enter":
					m.isDocsSearching = false
					return m, nil
				case "esc":
					m.isDocsSearching = false
					m.docsSearchQuery = ""
					m.docsScroll = 0
					return m, nil
				case "backspace":
					if len(m.docsSearchQuery) > 0 {
						m.docsSearchQuery = m.docsSearchQuery[:len(m.docsSearchQuery)-1]
						m.docsScroll = 0
					}
					return m, nil
				default:
					if len(msg.Runes) > 0 {
						m.docsSearchQuery += string(msg.Runes)
						m.docsScroll = 0
						return m, nil
					}
				}
			} else {
				switch key {
				case "/":
					m.isDocsSearching = true
					return m, nil
				case "t", "T":
					if m.docsLang == "zh" {
						m.docsLang = "en"
					} else {
						m.docsLang = "zh"
					}
					return m, nil
				case "esc":
					if m.docsSearchQuery != "" {
						m.docsSearchQuery = ""
						m.docsScroll = 0
					} else {
						m.activeView = ViewDashboard
					}
					return m, nil
				case "j", "down":
					maxScroll := m.getDocsMaxScroll()
					if m.docsScroll < maxScroll {
						m.docsScroll++
					}
					return m, nil
				case "k", "up":
					maxScroll := m.getDocsMaxScroll()
					if m.docsScroll > maxScroll {
						m.docsScroll = maxScroll
					}
					if m.docsScroll > 0 {
						m.docsScroll--
					}
					return m, nil
				case "ctrl+d":
					maxScroll := m.getDocsMaxScroll()
					m.docsScroll += 10
					if m.docsScroll > maxScroll {
						m.docsScroll = maxScroll
					}
					return m, nil
				case "ctrl+u":
					m.docsScroll -= 10
					if m.docsScroll < 0 {
						m.docsScroll = 0
					}
					return m, nil
				case "g", "home":
					m.docsScroll = 0
					return m, nil
				case "G", "end":
					m.docsScroll = m.getDocsMaxScroll()
					return m, nil
				}
			}
		}

		// ==================== GLOBAL VIEW CYCLING (TAB / SHIFT+TAB) ====================
		if !(m.activeView == ViewDocs && m.isDocsSearching) && !(m.activeView == ViewHistory && m.isHistorySearching) {
			switch key {
			case "tab":
				m.nextView()
				return m, nil
			case "shift+tab", "backtab":
				m.prevView()
				return m, nil
			}

			// ==================== GLOBAL COMMAND / COPY / SHORTCUTS ====================
			if key == ":" {
				m.isCommandMode = true
				m.commandInput = ""
				return m, nil
			}

			if key == "y" || (key == "c" && m.activeView == ViewContext) {
				if m.activeView == ViewContext {
					payload := core.ExtractAgentContextPayload(m.history, m.sessionID, "Gemini 3.7 Flash")
					text := m.GetContextInspectorContent(payload)
					_ = CopyToClipboard(text)
					subcatName := m.getSubcategoryName(m.contextSubItemIndex)
					m.statusMessage = fmt.Sprintf("📋 Copied [%s] to clipboard!", subcatName)
					m.statusMessageTime = time.Now()
					return m, nil
				} else if m.activeView == ViewHistory && len(m.history) > 0 {
					event := m.latestEvent
					if m.selectedIdx >= 0 && m.selectedIdx < len(m.history) {
						event = m.history[m.selectedIdx]
					}
					_ = CopyToClipboard(event.RawContent)
					m.statusMessage = fmt.Sprintf("📋 Copied Step #%d payload to clipboard!", event.StepIndex)
					m.statusMessageTime = time.Now()
					return m, nil
				}
			}

			// ==================== GLOBAL VIEW SWITCHING (DIRECT SHORTCUTS) ====================
			switch key {
			case "ctrl+c":
				return m, tea.Quit
			case "q":
				m.statusMessage = "💡 Type :q and press Enter to quit, or press Ctrl+C"
				m.statusMessageTime = time.Now()
				return m, nil
			case "1":
				m.activeView = ViewDashboard
				m.isDocsSearching = false
				m.isHistorySearching = false
				return m, nil
			case "2":
				m.activeView = ViewHistory
				m.focusPane = FocusList
				m.isDocsSearching = false
				m.isHistorySearching = false
				return m, nil
			case "3":
				m.activeView = ViewContext
				m.contextFocusPane = FocusList
				m.isDocsSearching = false
				m.isHistorySearching = false
				return m, nil
			case "4":
				m.activeView = ViewDocs
				m.isDocsSearching = false
				m.isHistorySearching = false
				return m, nil
			}
		}

		// ==================== CONTEXT VIEW KEYBINDINGS ====================
		if m.activeView == ViewContext {
			switch key {
			case "r", "R":
				m.isContextRawMode = !m.isContextRawMode
				return m, nil
			case "m", "M":
				if m.contextTargetAgent == "" || m.contextTargetAgent == "MAIN" {
					m.contextTargetAgent = "SUBAGENT"
				} else {
					m.contextTargetAgent = "MAIN"
				}
				return m, nil
			case "j", "down":
				if m.contextFocusPane == FocusList {
					if m.contextSubItemIndex < 11 {
						m.contextSubItemIndex++
						m.contextDetailScroll = 0
					}
				} else {
					m.contextDetailScroll++
				}
				return m, nil
			case "k", "up":
				if m.contextFocusPane == FocusList {
					if m.contextSubItemIndex > 0 {
						m.contextSubItemIndex--
						m.contextDetailScroll = 0
					}
				} else {
					if m.contextDetailScroll > 0 {
						m.contextDetailScroll--
					}
				}
				return m, nil
			case "l", "enter", "right":
				m.contextFocusPane = FocusDetail
				return m, nil
			case "h", "esc", "left":
				if m.contextFocusPane == FocusDetail {
					m.contextFocusPane = FocusList
					m.contextDetailScroll = 0
				} else {
					m.activeView = ViewDashboard
				}
				return m, nil
			case "ctrl+d":
				m.contextDetailScroll += 10
				return m, nil
			case "ctrl+u":
				m.contextDetailScroll -= 10
				if m.contextDetailScroll < 0 {
					m.contextDetailScroll = 0
				}
				return m, nil
			case "g", "home":
				if m.contextFocusPane == FocusList {
					m.contextSubItemIndex = 0
				} else {
					m.contextDetailScroll = 0
				}
				return m, nil
			case "G", "end":
				if m.contextFocusPane == FocusList {
					m.contextSubItemIndex = 11
				}
				return m, nil
			}
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
			var detailInnerWidth int
			var availableLines int
			if m.width < 100 {
				detailInnerWidth = m.width - 4
				bodyHeight := m.height - 2
				if bodyHeight < 8 {
					bodyHeight = 8
				}
				topContentRows := (bodyHeight - 4) * 45 / 100
				if topContentRows < 11 {
					topContentRows = 11
				}
				topBoxHeight := topContentRows + 2
				bottomInner := bodyHeight - topBoxHeight - 2
				if bottomInner < 4 {
					bottomInner = 4
				}
				availableLines = bottomInner - 1
			} else {
				detailInnerWidth = m.width - 38 - 4
				innerRowsLimit := m.height - 4
				topInner := 6
				bottomInner := innerRowsLimit - topInner - 2
				availableLines = bottomInner - 1
			}
			if availableLines < 1 {
				availableLines = 1
			}

			selectedEvent, hasEvent := m.getSelectedEvent()
			var fullLines []string
			if hasEvent {
				fullLines = m.buildContentPayloadLines(selectedEvent, detailInnerWidth)
			}
			totalInspectorLines := len(fullLines)

			maxScroll := totalInspectorLines - availableLines
			if maxScroll < 0 {
				maxScroll = 0
			}
			filtered := m.getFilteredHistory()

			// Search mode for jump-to-step
			if m.isHistorySearching {
				switch key {
				case "esc":
					m.isHistorySearching = false
					m.historyStepQuery = ""
					m.historySearchErr = ""
					return m, nil
				case "enter":
					m.isHistorySearching = false
					query := strings.TrimSpace(m.historyStepQuery)
					if query != "" {
						targetStep, err := strconv.Atoi(query)
						found := false
						if err == nil {
							for i, e := range filtered {
								if e.StepIndex == targetStep {
									m.selectedIdx = len(filtered) - 1 - i
									m.historyOffset = m.selectedIdx - 2
									if m.historyOffset < 0 {
										m.historyOffset = 0
									}
									m.historySearchErr = ""
									found = true
									break
								}
							}
						}
						if !found {
							lowerQ := strings.ToLower(query)
							for i, e := range filtered {
								if strings.Contains(strings.ToLower(e.Summary), lowerQ) ||
									strings.Contains(strings.ToLower(string(e.Type)), lowerQ) {
									m.selectedIdx = len(filtered) - 1 - i
									m.historyOffset = m.selectedIdx - 2
									if m.historyOffset < 0 {
										m.historyOffset = 0
									}
									m.historySearchErr = ""
									found = true
									break
								}
							}
						}
						if !found {
							m.historySearchErr = fmt.Sprintf("Step '%s' not found", query)
						} else {
							m.historyStepQuery = ""
						}
					}
					return m, nil
				case "backspace":
					if len(m.historyStepQuery) > 0 {
						m.historyStepQuery = m.historyStepQuery[:len(m.historyStepQuery)-1]
					}
					m.historySearchErr = ""
					return m, nil
				default:
					if len(msg.Runes) > 0 {
						r := msg.Runes[0]
						m.historyStepQuery += string(r)
						m.historySearchErr = ""
					}
					return m, nil
				}
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

			// FocusList hotkeys for filter and search
			if m.focusPane == FocusList && !m.isVisualMode {
				if key == "/" {
					m.isHistorySearching = true
					return m, nil
				}
				if key == "t" || key == "T" {
					m.cycleTypeFilter()
					return m, nil
				}
				if key == "c" || key == "C" {
					m.cycleCacheFilter()
					return m, nil
				}
			}

			switch key {
			case "enter", "right", "l":
				if m.focusPane == FocusList {
					m.focusPane = FocusDetail
				}
			case "esc", "left", "h":
				m.historySearchErr = ""
				if m.focusPane == FocusDetail {
					m.focusPane = FocusList
					m.isVisualMode = false
				} else if m.historyStepQuery != "" || m.historyTypeFilter != TypeFilterAll || m.historyCacheFilter != CacheFilterAll {
					m.historyStepQuery = ""
					m.historyTypeFilter = TypeFilterAll
					m.historyCacheFilter = CacheFilterAll
					m.selectedIdx = 0
					m.historyOffset = 0
				}
			case "up", "k":
				m.historySearchErr = ""
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
				m.historySearchErr = ""
				if m.focusPane == FocusList {
					if m.selectedIdx < len(filtered)-1 {
						m.selectedIdx++
						m.detailScroll = 0
						for m.selectedIdx >= m.historyOffset+m.getHistoryVisibleCards() {
							m.historyOffset++
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
					maxVisibleCards := m.getHistoryVisibleCards()
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
					maxVisibleCards := m.getHistoryVisibleCards()
					m.selectedIdx += maxVisibleCards
					if m.selectedIdx >= len(filtered) {
						m.selectedIdx = len(filtered) - 1
					}
					if m.selectedIdx < 0 {
						m.selectedIdx = 0
					}
					m.detailScroll = 0
					for m.selectedIdx >= m.historyOffset+m.getHistoryVisibleCards() {
						m.historyOffset++
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
					if len(filtered) > 0 {
						m.selectedIdx = len(filtered) - 1
						m.detailScroll = 0
						for m.selectedIdx >= m.historyOffset+m.getHistoryVisibleCards() {
							m.historyOffset++
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

	if m.isShortcutsModalOpen {
		return m.renderShortcutsModal()
	}

	if m.isSessionSwitcherOpen {
		return m.renderSessionSwitcherModal()
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	var content string
	switch m.activeView {
	case ViewDocs:
		content = m.renderDocsView()
	case ViewHistory:
		content = m.renderHistoryView()
	case ViewContext:
		content = m.renderContextView()
	default:
		content = m.renderDashboardView()
	}

	// Content must occupy strictly (m.height - 2) lines
	contentRows := m.height - 2
	if contentRows < 1 {
		contentRows = 1
	}

	rawContentLines := strings.Split(content, "\n")
	var finalContentLines []string
	for i := 0; i < contentRows; i++ {
		if i < len(rawContentLines) {
			line := rawContentLines[i]
			if lipgloss.Width(line) > m.width && m.width > 0 {
				line = lipgloss.NewStyle().MaxWidth(m.width).Render(line)
			}
			finalContentLines = append(finalContentLines, line)
		} else {
			finalContentLines = append(finalContentLines, "")
		}
	}

	headerLine := lipgloss.NewStyle().MaxWidth(m.width).Render(header)
	footerLine := lipgloss.NewStyle().MaxWidth(m.width).Render(footer)

	allLines := make([]string, 0, m.height)
	allLines = append(allLines, headerLine)
	allLines = append(allLines, finalContentLines...)
	allLines = append(allLines, footerLine)

	return strings.Join(allLines, "\n")
}

func (m Model) renderHeader() string {
	title := HeaderStyle.Render(" HEIMDALL ")
	if m.width < 90 {
		title = HeaderStyle.Render(" HEIMDALL ")
	}

	tab1 := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [1] Dashboard ")
	tab2 := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [2] History ")
	tab3 := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [3] Context ")
	tab4 := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [4] Docs ")

	switch m.activeView {
	case ViewDashboard:
		tab1 = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render(" [1] Dashboard ")
	case ViewHistory:
		tab2 = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render(" [2] History ")
	case ViewContext:
		tab3 = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render(" [3] Context ")
	case ViewDocs:
		tab4 = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render(" [4] Docs ")
	}

	viewTabs := tab1 + tab2 + tab3 + tab4
	left := lipgloss.JoinHorizontal(lipgloss.Center, title, viewTabs)

	shortHash := m.sessionID
	if len(shortHash) > 8 {
		shortHash = shortHash[:8]
	}
	timeInfo := lipgloss.NewStyle().Foreground(ColorLightText).Render(time.Now().Format("15:04:05"))
	sep := lipgloss.NewStyle().Foreground(ColorBorder).Render(" | ")
	stepsInfo := lipgloss.NewStyle().Foreground(ColorLightText).Render(fmt.Sprintf("Steps: %d", len(m.history)))
	sessionTag := lipgloss.NewStyle().Foreground(ColorHighlight).Render(fmt.Sprintf("(%s)", shortHash))
	right := timeInfo + sep + stepsInfo + sep + sessionTag

	gapWidth := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gapWidth < 1 {
		gapWidth = 1
	}
	gap := strings.Repeat(" ", gapWidth)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Center, left, gap, right)
	return lipgloss.NewStyle().MaxWidth(m.width).Render(headerLine)
}

func (m Model) renderFooter() string {
	if m.isCommandMode {
		prompt := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(":" + m.commandInput + "█")
		hint := lipgloss.NewStyle().Foreground(ColorLightText).Render("  (type 'q' + Enter to quit, 'w' to export, Esc to cancel)")
		return lipgloss.NewStyle().MaxWidth(m.width).Render(prompt + hint)
	}

	if m.statusMessage != "" && time.Since(m.statusMessageTime) < 4*time.Second {
		alert := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorSuccess).Padding(0, 1).Render(m.statusMessage)
		return lipgloss.NewStyle().MaxWidth(m.width).Render(alert)
	}

	if m.clipboardStatus != "" && time.Since(m.clipboardStatusTime) < 4*time.Second {
		alert := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorSuccess).Padding(0, 1).Render(m.clipboardStatus)
		return lipgloss.NewStyle().MaxWidth(m.width).Render(alert)
	}

	var hints string
	if m.activeView == ViewContext {
		modeLabel := "[r] Raw"
		if m.isContextRawMode {
			modeLabel = "[r] Refined"
		}
		hints = fmt.Sprintf(" %s Select  %s  %s Focus  %s Copy  %s Command  %s Shortcuts",
			KeyStyle.Render("[j/k]"), KeyStyle.Render(modeLabel),
			KeyStyle.Render("[l/h]"), KeyStyle.Render("[y/c]"), KeyStyle.Render("[:] (:q)"), KeyStyle.Render("[?]"))
	} else if m.activeView == ViewDocs {
		if m.isDocsSearching {
			hints = fmt.Sprintf(" %s Finish Search  %s Clear & Return",
				KeyStyle.Render("[Enter]"), KeyStyle.Render("[Esc]"))
		} else {
			langLabel := "[t] Lang (繁中)"
			if m.docsLang == "en" {
				langLabel = "[t] Lang (EN)"
			}
			hints = fmt.Sprintf(" %s Filter  %s  %s Scroll  %s Cycle  %s Shortcuts  %s Switch  %s Command",
				KeyStyle.Render("[/]"), KeyStyle.Render(langLabel), KeyStyle.Render("[j/k]"), KeyStyle.Render("[Tab]"), KeyStyle.Render("[?]"), KeyStyle.Render("[Ctrl+p]"), KeyStyle.Render("[:] (:q)"))
		}
	} else if m.activeView == ViewHistory {
		if m.isHistorySearching {
			hints = fmt.Sprintf(" %s Confirm  %s Clear/Exit  [0-9] Type Step #",
				KeyStyle.Render("[Enter]"), KeyStyle.Render("[Esc]"))
		} else if m.isVisualMode {
			hints = fmt.Sprintf(" %s Yank  %s Move  %s Cancel",
				KeyStyle.Render("[y]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[Esc]"))
		} else if m.focusPane == FocusList {
			if m.width < 90 {
				hints = fmt.Sprintf(" %s Focus  %s Type  %s Cache  %s Copy  %s Command  %s Shortcuts",
					KeyStyle.Render("[l]"), KeyStyle.Render("[t]"), KeyStyle.Render("[c]"), KeyStyle.Render("[y/c]"), KeyStyle.Render("[:] (:q)"), KeyStyle.Render("[?]"))
			} else {
				hints = fmt.Sprintf(" %s Focus  %s Step #  %s Type  %s Cache  %s Copy  %s Command  %s Shortcuts",
					KeyStyle.Render("[l]"), KeyStyle.Render("[/]"), KeyStyle.Render("[t]"), KeyStyle.Render("[c]"), KeyStyle.Render("[y/c]"), KeyStyle.Render("[:] (:q)"), KeyStyle.Render("[?]"))
			}
		} else {
			hints = fmt.Sprintf(" %s List  %s Copy  %s Scroll  %s Cycle  %s Shortcuts  %s Command",
				KeyStyle.Render("[h/Esc]"), KeyStyle.Render("[y/c]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[Tab]"), KeyStyle.Render("[?]"), KeyStyle.Render("[:] (:q)"))
		}
	} else {
		if len(m.history) > 0 {
			hints = fmt.Sprintf(" %s Inspect  %s Playback  %s Cycle  %s Shortcuts  %s Switch  %s LIVE  %s Command",
				KeyStyle.Render("[Enter]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[Tab]"), KeyStyle.Render("[?]"), KeyStyle.Render("[Ctrl+p]"), KeyStyle.Render("[G]"), KeyStyle.Render("[:] (:q)"))
		} else {
			hints = fmt.Sprintf(" %s Shortcuts  %s Cycle  %s Switch  %s Context  %s Command",
				KeyStyle.Render("[?]"), KeyStyle.Render("[Tab]"), KeyStyle.Render("[Ctrl+p]"), KeyStyle.Render("[2]"), KeyStyle.Render("[:] (:q)"))
		}
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(hints)
}

// wrapVisualLines wraps text into slices where each chunk strictly has visual terminal column width <= maxWidth.
// It is fully aware of ANSI escape sequences and CJK double-width characters.
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
		inAnsi := false

		for _, r := range runes {
			if r == 0x1b {
				inAnsi = true
				currentChunk = append(currentChunk, r)
				continue
			}
			if inAnsi {
				currentChunk = append(currentChunk, r)
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					inAnsi = false
				}
				continue
			}

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
