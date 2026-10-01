package ui

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"heimdall/internal/core"
)

type ActiveView int

const (
	ViewDashboard ActiveView = iota
	ViewHistory
	ViewContext
	ViewDocs
	ViewInsights
)

type FocusPane int

const (
	FocusList FocusPane = iota
	FocusDetail
)

func (m Model) contextHistoryVisibleRows() int {
	rows := m.height - 2 - contextHistoryListBorderLines - contextHistoryListHeaderLines - 1
	if rows < minimumContextHistoryRows {
		return minimumContextHistoryRows
	}
	return rows
}

const (
	// statusMessageDuration defines the on-screen display duration for transient notifications
	statusMessageDuration = 4 * time.Second

	// defaultHistoryCapacity defines the pre-allocation slice capacity for event history
	defaultHistoryCapacity = 1000
)

// AgentEventMsg wraps core.UnifiedAgentEvent as a bubbletea message
type AgentEventMsg core.UnifiedAgentEvent

// HistoryBatchMsg applies a coherent background hydration batch in one UI
// update. It prevents one redraw and one Context invalidation per transcript row.
type HistoryBatchMsg struct {
	Events []core.UnifiedAgentEvent
}

// SessionCatalogMsg replaces the lightweight startup session with the complete
// asynchronously discovered catalog. It keeps catalog I/O outside the TUI loop.
type SessionCatalogMsg struct {
	Sessions []core.SessionInfo
}

// ContextEstimateMsg delivers snapshot-derived text measurements prepared in a
// background goroutine. Rendering the Dashboard must not open SQLite or decode
// a large protobuf blob on the UI thread.
type ContextEstimateMsg struct {
	SessionID    string
	HistoryCount int
	Revision     uint64
	Refresh      bool
	Estimate     core.VisibleContextEvidenceEstimate
	Payload      core.AgentContextPayload
}

// DashboardReadModelMsg delivers the complete dashboard projection prepared
// outside the TUI update and render paths.
type DashboardReadModelMsg struct {
	SessionID    string
	HistoryCount int
	Revision     uint64
	ReadModel    core.DashboardReadModel
}

// PlaybackContextEstimatesMsg delivers the transcript-only playback cache.
// Its work is pure local tokenization and runs off the TUI update path.
type PlaybackContextEstimatesMsg struct {
	SessionID    string
	HistoryCount int
	Revision     int
	Estimates    []core.VisibleContextEvidenceEstimate
}

// ContextPayloadBuilder is injected by the application boundary. Keeping this
// dependency out of Model prevents redraws and unit tests from touching the
// host filesystem or session database.
type ContextPayloadBuilder func([]core.UnifiedAgentEvent, string) core.AgentContextPayload

// DashboardReadModelBuilder is injected by the application boundary. The
// builder reaches core QueryService, never an adapter, parser, or database.
type DashboardReadModelBuilder func([]core.UnifiedAgentEvent, string) core.DashboardReadModel

// ModelData is the non-blocking initial data available before the TUI starts.
// History is normally empty at startup and populated by background batches.
type ModelData struct {
	Sessions                  []core.SessionInfo
	InitialHistory            []core.UnifiedAgentEvent
	ContextPayloadBuilder     ContextPayloadBuilder
	DashboardReadModelBuilder DashboardReadModelBuilder
}

// Model represents the bubbletea application state
type Model struct {
	runtimeManaged                 bool
	sessionEpoch                   uint64
	sessionRevision                uint64
	pendingSessionUpdate           *core.SessionUpdate
	sessionAnalysisPending         bool
	switchRequest                  uint64
	sourceHealth                   core.MonitorHealth
	sourceLoading                  bool
	analysisReport                 core.SessionAnalysisReport
	reportReady                    bool
	insightsIndex                  int
	insightsGroup                  int
	sessionID                      string
	activeView                     ActiveView
	focusPane                      FocusPane
	dashboardIdx                   int
	detailScroll                   int
	isVisualMode                   bool
	visualStart                    int
	visualCursor                   int
	clipboardStatus                string
	clipboardStatusTime            time.Time
	latestEvent                    core.UnifiedAgentEvent
	history                        []core.UnifiedAgentEvent
	historyIndex                   map[string]int
	selectedIdx                    int
	historyOffset                  int
	width                          int
	height                         int
	lastActivity                   time.Time
	isSessionSwitcherOpen          bool
	isShortcutsModalOpen           bool
	sessionSearchQuery             string
	availableSessions              []core.SessionInfo
	filteredSessions               []core.SessionInfo
	switcherSelectedIdx            int
	docsSearchQuery                string
	isDocsSearching                bool
	docsScroll                     int
	docsLang                       string
	historyTypeFilter              TypeFilter
	historyCacheFilter             CacheFilter
	historyStepQuery               string
	isHistorySearching             bool
	historySearchErr               string
	contextSubItemIndex            int
	isContextRawMode               bool
	contextFocusPane               FocusPane
	contextDetailScroll            int
	contextHistoryList             bool
	contextHistoryIndex            int
	contextHistoryItemCount        int
	contextHistoryScrollOffset     int
	contextPayload                 core.AgentContextPayload
	contextPayloadBuilder          ContextPayloadBuilder
	dashboardReadModel             core.DashboardReadModel
	dashboardReadModelBuilder      DashboardReadModelBuilder
	dashboardReadModelHistoryCount int
	dashboardReadModelRevision     uint64
	dashboardReadModelPending      bool
	dashboardReadModelDirty        bool
	contextEstimate                core.VisibleContextEvidenceEstimate
	contextEstimateHistoryCount    int
	contextEstimateRefreshPending  bool
	contextEstimateRefreshDirty    bool
	playbackContextEstimates       []core.VisibleContextEvidenceEstimate
	playbackEstimateHistoryCount   int
	playbackEstimateRevision       int
	playbackEstimateCachedRevision int
	playbackEstimateRefreshPending bool
	playbackEstimateRefreshDirty   bool
	contextPayloadReady            bool
	contextRevision                uint64
	contextPayloadRevision         uint64
	contextInspectorLines          []string
	contextInspectorMax            int
	contextInspectorSubcat         int
	contextInspectorRaw            bool
	contextInspectorHistory        bool
	contextInspectorEventIndex     int
	contextInspectorWidth          int
	contextInspectorHeight         int
	isCommandMode                  bool
	commandInput                   string
	statusMessage                  string
	statusMessageTime              time.Time
	switcher                       SessionSwitcher
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

// CacheFilter filters direct cache-content field observations in History Explorer.
type CacheFilter string

const (
	CacheFilterAll      CacheFilter = "All"
	CacheFilterObserved CacheFilter = "Observed"
	CacheFilterOmitted  CacheFilter = "Omitted"
)

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
		m.historyCacheFilter = CacheFilterObserved
	case CacheFilterObserved:
		m.historyCacheFilter = CacheFilterOmitted
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
	// Cache filters only report field presence. They do not infer a cache hit,
	// cache miss, TTL, or server-side cache write.
	if !e.IsCloudStep() || !e.Usage.Available {
		return false
	}
	switch filter {
	case CacheFilterObserved:
		return e.Usage.HasUncachedInputTokens && e.Usage.HasCachedInputTokens
	case CacheFilterOmitted:
		return e.Usage.HasUncachedInputTokens && !e.Usage.HasCachedInputTokens
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
		m.refreshContextInspectorCache()
	case ViewContext:
		m.activeView = ViewDocs
	case ViewDocs:
		m.activeView = ViewInsights
	case ViewInsights:
		m.activeView = ViewDashboard
	}
}

// prevView cycles active view counter-clockwise: Dashboard -> Docs -> Context -> History -> Dashboard
func (m *Model) prevView() {
	m.isDocsSearching = false
	m.isHistorySearching = false
	switch m.activeView {
	case ViewDashboard:
		m.activeView = ViewInsights
	case ViewInsights:
		m.activeView = ViewDocs
	case ViewHistory:
		m.activeView = ViewDashboard
	case ViewContext:
		m.activeView = ViewHistory
		m.focusPane = FocusList
	case ViewDocs:
		m.activeView = ViewContext
		m.contextFocusPane = FocusList
		m.refreshContextInspectorCache()
	}
}

// NewModel creates an empty, dependency-free TUI model. Production callers use
// NewModelWithData to inject the session catalog and cached context builder.
func NewModel(sessionID string, openSwitcherOnStart bool, switcher ...SessionSwitcher) Model {
	return NewModelWithData(sessionID, openSwitcherOnStart, ModelData{}, switcher...)
}

// NewModelWithData creates an initial TUI model without performing discovery,
// history I/O, SQLite access, or tokenization on the UI thread.
func NewModelWithData(sessionID string, openSwitcherOnStart bool, data ModelData, switcher ...SessionSwitcher) Model {
	sessions := data.Sessions
	var sw SessionSwitcher
	if len(switcher) > 0 {
		sw = switcher[0]
	}

	initialHistory := data.InitialHistory
	if initialHistory == nil {
		initialHistory = make([]core.UnifiedAgentEvent, 0, defaultHistoryCapacity)
	}

	var latestEvent core.UnifiedAgentEvent
	dashboardIdx := 0
	if len(initialHistory) > 0 {
		latestEvent = initialHistory[len(initialHistory)-1]
		dashboardIdx = len(initialHistory) - 1
	}

	m := Model{
		sessionID:                 sessionID,
		activeView:                ViewDashboard,
		focusPane:                 FocusList,
		dashboardIdx:              dashboardIdx,
		detailScroll:              0,
		history:                   initialHistory,
		historyIndex:              buildHistoryIndex(initialHistory),
		latestEvent:               latestEvent,
		selectedIdx:               0,
		lastActivity:              time.Now(),
		availableSessions:         sessions,
		filteredSessions:          filterSessions(sessions, ""),
		switcherSelectedIdx:       0,
		isSessionSwitcherOpen:     openSwitcherOnStart,
		isShortcutsModalOpen:      false,
		docsSearchQuery:           "",
		isDocsSearching:           false,
		docsScroll:                0,
		docsLang:                  "en",
		historyTypeFilter:         TypeFilterAll,
		historyCacheFilter:        CacheFilterAll,
		historyStepQuery:          "",
		isHistorySearching:        false,
		contextPayloadBuilder:     data.ContextPayloadBuilder,
		dashboardReadModelBuilder: data.DashboardReadModelBuilder,
		switcher:                  sw,
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

func buildHistoryIndex(history []core.UnifiedAgentEvent) map[string]int {
	index := make(map[string]int, len(history))
	for position, event := range history {
		index[historyEventKey(event)] = position
	}
	return index
}

func historyEventKey(event core.UnifiedAgentEvent) string {
	return event.SessionID + ":" + strconv.Itoa(event.StepIndex)
}

func (m *Model) ensureHistoryIndex() {
	if len(m.historyIndex) == len(m.history) {
		return
	}
	m.historyIndex = buildHistoryIndex(m.history)
}

// applyHistoryEvents incorporates one live event or a chronological hydration
// batch without repeatedly rebuilding the Context inspector. The selected
// history item is identified by its stable session/step key so background
// hydration cannot make an operator lose the item they are inspecting.
func (m *Model) applyHistoryEvents(events []core.UnifiedAgentEvent) {
	if len(events) == 0 {
		return
	}

	m.ensureHistoryIndex()
	wasAtLatestDashboard := m.dashboardIdx == len(m.history)-1 || len(m.history) == 0
	lockInspection := m.activeView == ViewHistory && (m.selectedIdx > 0 || m.focusPane == FocusDetail)
	inspectedKey := ""
	if lockInspection {
		if selected, ok := m.getSelectedEvent(); ok {
			inspectedKey = historyEventKey(selected)
		}
	}

	processed := false
	historyNeedsSort := false
	for _, event := range events {
		if event.SessionID != m.sessionID && m.sessionID != "" && event.SessionID != "" {
			continue
		}

		processed = true
		key := historyEventKey(event)
		if existingIndex, exists := m.historyIndex[key]; exists {
			m.history[existingIndex] = mergeHistoryEventUpdate(m.history[existingIndex], event)
		} else {
			if len(m.history) > 0 && event.StepIndex < m.history[len(m.history)-1].StepIndex {
				historyNeedsSort = true
			}
			m.history = append(m.history, event)
			m.historyIndex[key] = len(m.history) - 1
		}

		if len(event.ConsumedStepIndices) > 0 {
			for _, childStepIndex := range event.ConsumedStepIndices {
				childKey := event.SessionID + ":" + strconv.Itoa(childStepIndex)
				if childIndex, exists := m.historyIndex[childKey]; exists {
					m.history[childIndex].PackagedInStepIdx = event.StepIndex
				}
			}
		}

	}

	if !processed {
		return
	}

	if historyNeedsSort {
		m.sortHistoryByStep()
		m.reconcileHistoryPackaging()
	}

	if len(m.history) > 0 {
		m.latestEvent = m.history[len(m.history)-1]
	}
	m.lastActivity = time.Now()
	if wasAtLatestDashboard {
		m.dashboardIdx = len(m.history) - 1
	}
	if lockInspection && inspectedKey != "" {
		filtered := m.getFilteredHistory()
		for index := len(filtered) - 1; index >= 0; index-- {
			if historyEventKey(filtered[index]) == inspectedKey {
				m.selectedIdx = len(filtered) - 1 - index
				break
			}
		}
	}
	m.refreshContextPayload()
	m.invalidateDashboardReadModel()
	m.playbackEstimateRevision++
}

// mergeHistoryEventUpdate keeps enrichment that belongs to a prior observation
// of the same transcript step. Antigravity may write RUNNING and DONE records
// with the same step index; the newer record owns display content and status,
// while absent linkage, local estimates, and persisted usage must not regress.
func mergeHistoryEventUpdate(existing, update core.UnifiedAgentEvent) core.UnifiedAgentEvent {
	if update.Scope == "" {
		update.Scope = existing.Scope
	}
	if update.ParentStepIdx == 0 {
		update.ParentStepIdx = existing.ParentStepIdx
	}
	if update.PackagedInStepIdx == 0 {
		update.PackagedInStepIdx = existing.PackagedInStepIdx
	}
	if len(update.ConsumedStepIndices) == 0 {
		update.ConsumedStepIndices = existing.ConsumedStepIndices
	}
	if update.Tokens.StepDelta == 0 && existing.Tokens.StepDelta > 0 {
		update.Tokens = existing.Tokens
	}
	if !update.Usage.Available && existing.Usage.Available {
		update.Usage = existing.Usage
	}
	return update
}

// sortHistoryByStep restores the transcript's chronological order after a
// latest-event preview is followed by older background hydration. Live events
// are already append-only, so they avoid this work entirely.
func (m *Model) sortHistoryByStep() {
	sort.SliceStable(m.history, func(left, right int) bool {
		return m.history[left].StepIndex < m.history[right].StepIndex
	})
	m.historyIndex = buildHistoryIndex(m.history)
}

// reconcileHistoryPackaging repairs child-to-cloud links after an out-of-order
// backfill. The source cloud event remains authoritative for its listed child
// steps, so no relationship is invented when a parent does not list a child.
func (m *Model) reconcileHistoryPackaging() {
	packagedByStep := make(map[int]int)
	for _, event := range m.history {
		for _, childStep := range event.ConsumedStepIndices {
			packagedByStep[childStep] = event.StepIndex
		}
	}
	for index := range m.history {
		m.history[index].PackagedInStepIdx = packagedByStep[m.history[index].StepIndex]
	}
}

// refreshContextPayload invalidates derived Context UI state. The next Context
// render rebuilds it lazily; AgentEventMsg handling therefore performs no DB or
// filesystem work while background history is hydrating.
func (m *Model) refreshContextPayload() {
	m.contextRevision++
	m.contextPayloadReady = false
	m.contextInspectorLines = nil
	m.contextInspectorMax = 0
}

// invalidateDashboardReadModel marks the presentation projection stale after
// session history changes. The replacement is prepared asynchronously.
func (m *Model) invalidateDashboardReadModel() {
	m.dashboardReadModelRevision++
}

// requestDashboardReadModelRefresh rebuilds the dashboard projection outside
// Bubble Tea's render path. Cursor movement only reads the last completed
// projection, so it cannot trigger tokenization or session queries.
func (m *Model) requestDashboardReadModelRefresh() tea.Cmd {
	if m.runtimeManaged {
		return nil
	}
	if m.dashboardReadModelPending {
		m.dashboardReadModelDirty = true
		return nil
	}
	m.dashboardReadModelPending = true
	m.dashboardReadModelDirty = false
	history := append([]core.UnifiedAgentEvent(nil), m.history...)
	sessionID := m.sessionID
	revision := m.dashboardReadModelRevision
	builder := m.dashboardReadModelBuilder
	return func() tea.Msg {
		readModel := core.BuildDashboardReadModelFromEvents(sessionID, history)
		if builder != nil {
			readModel = builder(history, sessionID)
		}
		return DashboardReadModelMsg{
			SessionID:    sessionID,
			HistoryCount: len(history),
			Revision:     revision,
			ReadModel:    readModel,
		}
	}
}

// requestContextEstimateRefresh rebuilds snapshot-derived visible-text counts
// outside the UI update path. Repeated hydration batches coalesce into one
// follow-up refresh so a long replay cannot schedule one payload build per row.
func (m *Model) requestContextEstimateRefresh() tea.Cmd {
	if m.runtimeManaged {
		return nil
	}
	if m.contextPayloadBuilder == nil {
		return nil
	}
	if m.contextEstimateRefreshPending {
		m.contextEstimateRefreshDirty = true
		return nil
	}
	m.contextEstimateRefreshPending = true
	m.contextEstimateRefreshDirty = false
	history := make([]core.UnifiedAgentEvent, len(m.history))
	copy(history, m.history)
	sessionID := m.sessionID
	revision := m.contextRevision
	builder := m.contextPayloadBuilder
	return func() tea.Msg {
		payload := builder(history, sessionID)
		return ContextEstimateMsg{
			SessionID:    sessionID,
			HistoryCount: len(history),
			Revision:     revision,
			Refresh:      true,
			Estimate:     core.EstimateVisibleContextEvidence(payload),
			Payload:      payload,
		}
	}
}

// requestPlaybackContextEstimateRefresh builds all per-step estimates once
// after history changes. Dashboard navigation reads this cache only, so j/k
// never performs SQLite I/O or scans the transcript.
func (m *Model) requestPlaybackContextEstimateRefresh() tea.Cmd {
	if m.runtimeManaged {
		return nil
	}
	if m.playbackEstimateRefreshPending {
		m.playbackEstimateRefreshDirty = true
		return nil
	}
	m.playbackEstimateRefreshPending = true
	m.playbackEstimateRefreshDirty = false
	history := make([]core.UnifiedAgentEvent, len(m.history))
	copy(history, m.history)
	sessionID := m.sessionID
	revision := m.playbackEstimateRevision
	return func() tea.Msg {
		return PlaybackContextEstimatesMsg{
			SessionID:    sessionID,
			HistoryCount: len(history),
			Revision:     revision,
			Estimates:    core.BuildPlaybackContextEvidence(history),
		}
	}
}

// playbackContextEstimateFor returns the cached estimate for the selected
// timeline event. An exact persisted snapshot can replace only the dimensions
// it truly contains; selected-step inbound and buffer evidence remain tied to
// the playback timeline.
func (m Model) playbackContextEstimateFor(historyIndex int, event core.UnifiedAgentEvent) core.VisibleContextEvidenceEstimate {
	if m.playbackEstimateHistoryCount != len(m.history) || m.playbackEstimateCachedRevision != m.playbackEstimateRevision {
		return core.VisibleContextEvidenceEstimate{}
	}
	if historyIndex < 0 || historyIndex >= len(m.playbackContextEstimates) {
		return core.VisibleContextEvidenceEstimate{}
	}

	estimate := m.playbackContextEstimates[historyIndex]
	if m.contextEstimate.SourceKind != core.ContextEvidencePersistedSnapshot || !m.contextEstimate.HasSnapshotGeneratedStep || m.contextEstimate.SnapshotGeneratedStepIndex != event.StepIndex {
		return estimate
	}

	estimate.SourceKind = core.ContextEvidencePersistedSnapshot
	estimate.SnapshotGenIndex = m.contextEstimate.SnapshotGenIndex
	estimate.SnapshotGeneratedStepIndex = m.contextEstimate.SnapshotGeneratedStepIndex
	estimate.HasSnapshotGeneratedStep = true
	estimate.SystemTokens = m.contextEstimate.SystemTokens
	estimate.ToolsTokens = m.contextEstimate.ToolsTokens
	estimate.HistoryTokens = m.contextEstimate.HistoryTokens
	estimate.TotalTokens = estimate.SystemTokens + estimate.ToolsTokens + estimate.ToolBufferTokens + estimate.HistoryTokens + estimate.InboundTokens
	return estimate
}

func (m *Model) refreshContextInspectorCache() {
	if m.width == 0 || m.height == 0 {
		return
	}
	payload := m.cachedContextPayload()
	m.contextPayload = payload
	m.contextPayloadReady = true
	m.contextPayloadRevision = m.contextRevision
	innerWidth, innerHeight := m.contextInspectorDimensions()
	payload.IsRawMode = m.isContextRawMode
	m.contextInspectorLines = m.buildRefinedInspectorLines(payload, innerWidth)
	if m.isContextRawMode {
		m.contextInspectorLines = m.buildRawWireLines(payload, innerWidth)
	}
	m.contextInspectorMax = len(m.contextInspectorLines) - innerHeight
	if m.contextInspectorMax < 0 {
		m.contextInspectorMax = 0
	}
	m.contextInspectorSubcat = m.contextSubItemIndex
	m.contextInspectorRaw = m.isContextRawMode
	m.contextInspectorHistory = m.contextHistoryList
	m.contextInspectorEventIndex = m.contextHistoryIndex
	m.contextInspectorWidth = innerWidth
	m.contextInspectorHeight = innerHeight
}

func (m Model) hasContextInspectorCache(innerWidth, innerHeight int) bool {
	return len(m.contextInspectorLines) > 0 &&
		m.contextInspectorSubcat == m.contextSubItemIndex &&
		m.contextInspectorRaw == m.isContextRawMode &&
		m.contextInspectorHistory == m.contextHistoryList &&
		m.contextInspectorEventIndex == m.contextHistoryIndex &&
		m.contextInspectorWidth == innerWidth &&
		m.contextInspectorHeight == innerHeight
}

func (m Model) cachedContextPayload() core.AgentContextPayload {
	if m.contextPayloadReady && m.contextPayloadRevision == m.contextRevision {
		return m.contextPayload
	}
	if m.contextPayloadBuilder != nil {
		return m.contextPayload
	}
	return core.BuildContextPayloadFromHistory(core.ContextBuildInput{History: m.history, SessionID: m.sessionID, NativeTools: core.GetNativeToolsDefinitions(m.history)})
}

func (m Model) latestContextHistoryStep() int {
	if len(m.history) == 0 {
		return 0
	}
	return m.history[len(m.history)-1].StepIndex
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
			if usedLines+linesNeeded > (availLines-1) && cardCount > 0 {
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

func formatObservedUsageLine(usage core.PersistedUsageObservation) string {
	if !usage.Available {
		return "• Persisted usage: unavailable"
	}
	if usage.HasUncachedInputTokens && usage.HasCachedInputTokens {
		return fmt.Sprintf("• Persisted: %s uncached input | %s cached input", formatCompactNumber(usage.UncachedInputTokens), formatCompactNumber(usage.CachedInputTokens))
	}
	if usage.HasUncachedInputTokens {
		return fmt.Sprintf("• Persisted: %s uncached input | 0 cached input (inferred)", formatCompactNumber(usage.UncachedInputTokens))
	}
	if !usage.HasObservedContextTokens {
		return "• Persisted usage: unavailable"
	}
	if usage.HasCachedInputTokens {
		return fmt.Sprintf("• Persisted: %s context | %s cached input observed", formatCompactNumber(usage.ObservedContextTokens), formatCompactNumber(usage.CachedInputTokens))
	}
	return fmt.Sprintf("• Persisted: %s context | input usage unavailable", formatCompactNumber(usage.ObservedContextTokens))
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
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role  : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
			toolName := m.getLocalToolName(e)
			if toolName == "" {
				toolName = string(e.Type)
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Action: Tool Output (%s)", toolName), maxWidth))
			lines = append(lines, truncateVisualWidth("• Status: Local process", maxWidth))
			if e.PackagedInStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("  └ Link: Cloud step #%04d", e.PackagedInStepIdx), maxWidth))
			} else {
				estTok := t.ActiveTurnTokens + t.ToolResultTokens
				if estTok == 0 {
					estTok = core.CountTokens(e.RawContent)
				}
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("  └ Local text: ~%s tok", formatCompactNumber(estTok)), maxWidth))
			}
			if e.ParentStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Parent: Triggered by #%04d", e.ParentStepIdx), maxWidth))
			}
			lines = append(lines, truncateVisualWidth("• Origin: Local machine process", maxWidth))
		} else if e.IsCloudStep() || e.Scope == core.ScopeCloudInference {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role  : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
			modelName := m.getStepModelName(e)
			if modelName == "" {
				modelName = "unknown"
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Model : %s", modelName), maxWidth))
			lines = append(lines, truncateVisualWidth(formatObservedUsageLine(e.Usage), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Local text (cl100k_base): +%s this event", formatCompactNumber(t.StepDelta)), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("  └ Transcript accumulated: %s", formatCompactNumber(t.RawLocalAccumulated)), maxWidth))
			if e.ParentStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Parent: Step #%04d (User Prompt)", e.ParentStepIdx), maxWidth))
			}
			if len(e.ToolCalls) > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Tool  : %s (%d call)", e.ToolCalls[0].ToolName, len(e.ToolCalls)), maxWidth))
			}
		} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role  : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
			lines = append(lines, truncateVisualWidth("• Origin: Human Client Prompt", maxWidth))
			promptTok := core.CountTokens(e.RawContent)
			if promptTok == 0 {
				promptTok = 1
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Payload: ~%d Prompt Tokens", promptTok), maxWidth))
			if e.PackagedInStepIdx > 0 {
				lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Link   : Cloud step #%04d", e.PackagedInStepIdx), maxWidth))
			} else {
				lines = append(lines, truncateVisualWidth("• Link   : No cloud step linked", maxWidth))
			}
		} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role  : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
			lines = append(lines, truncateVisualWidth("• Event : Context Compaction (Checkpoint)", maxWidth))
			lines = append(lines, truncateVisualWidth("• Scope : Transcript compaction record", maxWidth))
			summaryTok := core.CountTokens(e.RawContent)
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Summary: ~%s Tok (Prunes Old Turns)", formatCompactNumber(summaryTok)), maxWidth))
			lines = append(lines, truncateVisualWidth("• Status: Previous history was compacted", maxWidth))
		} else {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step  : #%04d (%s) at %s", e.StepIndex, e.Status, timeStr), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role  : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Type  : %s", e.Type), maxWidth))
			lines = append(lines, truncateVisualWidth("• Scope : System Bootstrap & Rules", maxWidth))
		}
		return lines
	}

	// ==================== FULL-WIDTH MODE (DETAILED TELEMETRY) ====================
	if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | 💻 LOCAL EXECUTION STEP", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role   : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Origin : Local Host Process (%s)", e.Type), maxWidth))
		if e.ParentStepIdx > 0 {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Parent : Triggered by Tool Call in Step #%04d", e.ParentStepIdx), maxWidth))
		}
		if e.PackagedInStepIdx > 0 {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Link   : Associated with cloud step #%04d", e.PackagedInStepIdx), maxWidth))
		} else {
			estTok := t.ActiveTurnTokens + t.ToolResultTokens
			if estTok == 0 {
				estTok = core.CountTokens(e.RawContent)
			}
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Local text estimate: ~%d tok", estTok), maxWidth))
		}
	} else if e.IsCloudStep() || e.Scope == core.ScopeCloudInference {
		modelName := e.Usage.ModelName
		if modelName == "" {
			modelName = "unknown"
		}
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | ☁️ CLOUD INFERENCE TURN", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role   : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Model  : %s (persisted metadata)", modelName), maxWidth))
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
		lines = append(lines, truncateVisualWidth(formatObservedUsageLine(e.Usage), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Local text (cl100k_base): +%d this event | %d accumulated transcript", t.StepDelta, t.RawLocalAccumulated), maxWidth))
	} else if e.Scope == core.ScopeUserInteraction || e.Type == core.StepTypeUserInput {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | 👤 USER INPUT", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role   : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
		lines = append(lines, truncateVisualWidth("• Origin : Human client prompt", maxWidth))
		promptTok := core.CountTokens(e.RawContent)
		if promptTok == 0 {
			promptTok = 1
		}
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Payload: ~%d Prompt Tokens (Local Inbound Intent)", promptTok), maxWidth))
		if e.PackagedInStepIdx > 0 {
			lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Link   : Associated with cloud step #%04d", e.PackagedInStepIdx), maxWidth))
		} else {
			lines = append(lines, truncateVisualWidth("• Link   : No cloud step linked", maxWidth))
		}
	} else if e.IsCompactionStep() || e.Scope == core.ScopeSystemCompaction {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | ⚙️ CONTEXT COMPACTION (CHECKPOINT)", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role   : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
		lines = append(lines, truncateVisualWidth("• Event  : Observed compaction checkpoint", maxWidth))
		lines = append(lines, truncateVisualWidth("• Scope  : Transcript record", maxWidth))
		summaryTok := core.CountTokens(e.RawContent)
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Payload: ~%d locally estimated summary tokens", summaryTok), maxWidth))
		lines = append(lines, truncateVisualWidth("• Scope  : The exact retained and omitted context is unavailable", maxWidth))
	} else {
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Step #%04d (%s) at %s | 📜 SYSTEM BOOTSTRAP", e.StepIndex, e.Status, timeStr), maxWidth))
		lines = append(lines, truncateVisualWidth(fmt.Sprintf("• Role   : %s (%s)", e.GetMessageRole(), e.GetMessageRoleDescription()), maxWidth))
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

	payloadHeader := fmt.Sprintf("Content Payload (Role: %s", e.GetMessageRole())
	if e.ContentSource == core.SourceKindArtifacts {
		payloadHeader += ", Source: STEP OUTPUT ARTIFACT"
	} else if e.ContentSource == core.SourceKindTranscript {
		payloadHeader += ", Source: TRANSCRIPT"
	}
	lines = append(lines, payloadHeader+"):")
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
	case SessionUpdateMsg:
		cmd := m.receiveSessionUpdate(core.SessionUpdate(msg))
		return m, cmd
	case sessionAnalysisMsg:
		cmd := m.receiveSessionAnalysis(msg)
		return m, cmd
	case sessionSwitchResultMsg:
		if msg.Request == m.switchRequest && msg.Err != nil {
			m.statusMessage, m.statusMessageTime = msg.Err.Error(), time.Now()
		}
		return m, nil
	case reportExportMsg:
		m.statusMessage = "Report saved: " + msg.Path
		if msg.Err != nil {
			m.statusMessage = "Report export failed: " + msg.Err.Error()
		}
		m.statusMessageTime = time.Now()
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.refreshContextInspectorCache()
		return m, nil

	case AgentEventMsg:
		if m.runtimeManaged {
			return m, nil
		}
		m.applyHistoryEvents([]core.UnifiedAgentEvent{core.UnifiedAgentEvent(msg)})
		return m, m.requestDashboardReadModelRefresh()

	case HistoryBatchMsg:
		if m.runtimeManaged {
			return m, nil
		}
		m.applyHistoryEvents(msg.Events)
		if m.activeView == ViewContext {
			m.refreshContextInspectorCache()
		}
		return m, tea.Batch(m.requestContextEstimateRefresh(), m.requestDashboardReadModelRefresh())

	case SessionCatalogMsg:
		m.availableSessions = msg.Sessions
		m.filteredSessions = filterSessions(m.availableSessions, m.sessionSearchQuery)
		if m.switcherSelectedIdx >= len(m.filteredSessions) {
			m.switcherSelectedIdx = 0
		}
		return m, nil

	case ContextEstimateMsg:
		if m.runtimeManaged {
			return m, nil
		}
		if msg.SessionID == m.sessionID && msg.HistoryCount >= m.contextEstimateHistoryCount {
			m.contextEstimate = msg.Estimate
			m.contextEstimateHistoryCount = msg.HistoryCount
		}
		if msg.Refresh && msg.SessionID == m.sessionID && msg.Revision == m.contextRevision {
			m.contextPayload = msg.Payload
			m.contextPayloadReady = true
			m.contextPayloadRevision = msg.Revision
			if m.activeView == ViewContext {
				m.refreshContextInspectorCache()
			}
		}
		if msg.Refresh && msg.SessionID == m.sessionID {
			m.contextEstimateRefreshPending = false
			if m.contextEstimateRefreshDirty {
				return m, m.requestContextEstimateRefresh()
			}
		}
		return m, nil

	case DashboardReadModelMsg:
		if m.runtimeManaged {
			return m, nil
		}
		if msg.SessionID != m.sessionID {
			return m, nil
		}
		m.dashboardReadModelPending = false
		if msg.HistoryCount == len(m.history) && msg.Revision == m.dashboardReadModelRevision {
			m.dashboardReadModel = msg.ReadModel
			m.dashboardReadModelHistoryCount = msg.HistoryCount
		}
		if m.dashboardReadModelDirty {
			return m, m.requestDashboardReadModelRefresh()
		}
		return m, nil

	case PlaybackContextEstimatesMsg:
		if m.runtimeManaged {
			return m, nil
		}
		if msg.SessionID != m.sessionID {
			return m, nil
		}
		m.playbackEstimateRefreshPending = false
		if msg.HistoryCount == len(m.history) && msg.Revision == m.playbackEstimateRevision {
			m.playbackContextEstimates = msg.Estimates
			m.playbackEstimateHistoryCount = msg.HistoryCount
			m.playbackEstimateCachedRevision = msg.Revision
		}
		if m.playbackEstimateRefreshDirty {
			return m, m.requestPlaybackContextEstimateRefresh()
		}
		return m, nil

	case SessionResetMsg:
		if m.runtimeManaged {
			return m, nil
		}
		m.sessionID = msg.SessionID
		m.history = make([]core.UnifiedAgentEvent, 0, 1000)
		m.historyIndex = buildHistoryIndex(m.history)
		m.latestEvent = core.UnifiedAgentEvent{}
		m.dashboardIdx = 0
		m.selectedIdx = 0
		m.detailScroll = 0
		m.historyOffset = 0
		m.isSessionSwitcherOpen = false
		m.isShortcutsModalOpen = false
		m.activeView = ViewDashboard
		m.refreshContextPayload()
		m.contextEstimate = core.VisibleContextEvidenceEstimate{}
		m.contextEstimateHistoryCount = 0
		m.contextEstimateRefreshPending = false
		m.contextEstimateRefreshDirty = false
		m.dashboardReadModel = core.DashboardReadModel{}
		m.dashboardReadModelHistoryCount = 0
		m.dashboardReadModelRevision++
		m.dashboardReadModelPending = false
		m.dashboardReadModelDirty = false
		m.playbackContextEstimates = nil
		m.playbackEstimateHistoryCount = 0
		m.playbackEstimateRevision = 0
		m.playbackEstimateCachedRevision = 0
		m.playbackEstimateRefreshPending = false
		m.playbackEstimateRefreshDirty = false
		m.clipboardStatus = fmt.Sprintf("Live attached to %s", truncateStr(msg.SessionID, 8))
		m.clipboardStatusTime = time.Now()
		return m, nil

	case SwitchSessionReqMsg:
		if m.switcher == nil {
			return m, nil
		}
		m.switchRequest++
		request, switcher, id := m.switchRequest, m.switcher, msg.SessionID
		return m, func() tea.Msg { return sessionSwitchResultMsg{Request: request, Err: switcher.SwitchSession(id)} }

	case SessionSwitchedMsg:
		if m.runtimeManaged {
			return m, nil
		}
		m.sessionID = msg.SessionID
		m.history = msg.Events
		m.historyIndex = buildHistoryIndex(m.history)
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
		m.refreshContextPayload()
		m.contextEstimate = core.VisibleContextEvidenceEstimate{}
		m.contextEstimateHistoryCount = 0
		m.contextEstimateRefreshPending = false
		m.contextEstimateRefreshDirty = false
		m.dashboardReadModel = core.DashboardReadModel{}
		m.dashboardReadModelHistoryCount = 0
		m.dashboardReadModelRevision++
		m.dashboardReadModelPending = false
		m.dashboardReadModelDirty = false
		m.playbackContextEstimates = nil
		m.playbackEstimateHistoryCount = 0
		m.playbackEstimateRevision = 0
		m.playbackEstimateCachedRevision = 0
		m.playbackEstimateRefreshPending = false
		m.playbackEstimateRefreshDirty = false
		m.clipboardStatus = fmt.Sprintf("Switched to session %s (%d steps)", truncateStr(msg.SessionID, 8), len(msg.Events))
		m.clipboardStatusTime = time.Now()
		return m, tea.Batch(m.requestContextEstimateRefresh(), m.requestDashboardReadModelRefresh())

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
				case "report":
					if !m.reportReady {
						m.statusMessage, m.statusMessageTime = "Report is not ready", time.Now()
						return m, nil
					}
					return m, m.exportAnalysisReportCmd()
				case "q", "quit", "q!", "exit":
					return m, tea.Quit
				case "w", "write":
					payload := m.cachedContextPayload()
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

			if (key == "y" || key == "c") && m.activeView == ViewContext {
				payload := m.cachedContextPayload()
				text := m.GetContextInspectorContent(payload)
				_ = CopyToClipboard(text)
				subcatName := m.getSubcategoryName(m.contextSubItemIndex)
				msg := fmt.Sprintf("📋 Copied [%s] to clipboard!", subcatName)
				m.clipboardStatus = msg
				m.clipboardStatusTime = time.Now()
				m.statusMessage = msg
				m.statusMessageTime = time.Now()
				return m, nil
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
				m.refreshContextInspectorCache()
				return m, nil
			case "4":
				m.activeView = ViewDocs
				m.isDocsSearching = false
				m.isHistorySearching = false
				return m, nil
			case "5":
				m.activeView = ViewInsights
				m.isDocsSearching, m.isHistorySearching = false, false
				return m, nil
			}
		}
		if m.activeView == ViewInsights {
			m.updateInsightsKey(key)
			return m, nil
		}

		// ==================== CONTEXT VIEW KEYBINDINGS ====================
		if m.activeView == ViewContext {
			switch key {
			case "r", "R":
				m.isContextRawMode = !m.isContextRawMode
				m.refreshContextInspectorCache()
				return m, nil
			case "j", "down":
				if m.contextHistoryList && m.contextFocusPane == FocusList {
					if m.contextHistoryIndex < m.contextHistoryItemCount-1 {
						m.contextHistoryIndex++
						if m.contextHistoryIndex >= m.contextHistoryScrollOffset+m.contextHistoryVisibleRows() {
							m.contextHistoryScrollOffset = m.contextHistoryIndex - m.contextHistoryVisibleRows() + 1
						}
					}
					m.refreshContextInspectorCache()
					return m, nil
				}
				if m.contextFocusPane == FocusList {
					if m.contextSubItemIndex < SubcatLast {
						m.contextSubItemIndex++
						m.contextDetailScroll = 0
					}
					m.refreshContextInspectorCache()
				} else {
					if m.contextDetailScroll < m.contextInspectorMaxScroll() {
						m.contextDetailScroll++
					}
				}
				return m, nil
			case "k", "up":
				if m.contextHistoryList && m.contextFocusPane == FocusList {
					if m.contextHistoryIndex > 0 {
						m.contextHistoryIndex--
						if m.contextHistoryIndex < m.contextHistoryScrollOffset {
							m.contextHistoryScrollOffset = m.contextHistoryIndex
						}
					}
					m.refreshContextInspectorCache()
					return m, nil
				}
				if m.contextFocusPane == FocusList {
					if m.contextSubItemIndex > 0 {
						m.contextSubItemIndex--
						m.contextDetailScroll = 0
					}
					m.refreshContextInspectorCache()
				} else {
					maxScroll := m.contextInspectorMaxScroll()
					if m.contextDetailScroll > maxScroll {
						m.contextDetailScroll = maxScroll
					}
					if m.contextDetailScroll > 0 {
						m.contextDetailScroll--
					}
				}
				return m, nil
			case "enter":
				if m.contextFocusPane == FocusList && !m.contextHistoryList && m.contextSubItemIndex == SubcatCurrentHistory {
					payload := m.cachedContextPayload()
					m.contextHistoryList = true
					m.contextHistoryIndex = 0
					m.contextHistoryItemCount = historyListItemCount(payload)
					m.contextHistoryScrollOffset = 0
					m.contextDetailScroll = 0
					m.refreshContextInspectorCache()
					return m, nil
				}
				m.contextFocusPane = FocusDetail
				return m, nil
			case "l", "right":
				m.contextFocusPane = FocusDetail
				return m, nil
			case "h", "left":
				if m.contextFocusPane == FocusDetail {
					m.contextFocusPane = FocusList
					m.contextDetailScroll = 0
				} else if !m.contextHistoryList {
					m.activeView = ViewDashboard
				}
				return m, nil
			case "esc":
				if m.contextFocusPane == FocusDetail {
					m.contextFocusPane = FocusList
					m.contextDetailScroll = 0
				} else if m.contextHistoryList {
					m.contextHistoryList = false
					m.contextDetailScroll = 0
					m.contextHistoryItemCount = 0
					m.contextHistoryScrollOffset = 0
					m.refreshContextInspectorCache()
				} else {
					m.activeView = ViewDashboard
				}
				return m, nil
			case "ctrl+d":
				m.contextDetailScroll += 10
				maxScroll := m.contextInspectorMaxScroll()
				if m.contextDetailScroll > maxScroll {
					m.contextDetailScroll = maxScroll
				}
				return m, nil
			case "ctrl+u":
				m.contextDetailScroll -= 10
				if m.contextDetailScroll < 0 {
					m.contextDetailScroll = 0
				}
				return m, nil
			case "g", "home":
				if m.contextHistoryList && m.contextFocusPane == FocusList {
					m.contextHistoryIndex = 0
					m.contextHistoryScrollOffset = 0
					m.refreshContextInspectorCache()
					return m, nil
				}
				if m.contextFocusPane == FocusList {
					m.contextSubItemIndex = 0
					m.contextDetailScroll = 0
					m.refreshContextInspectorCache()
				} else {
					m.contextDetailScroll = 0
				}
				return m, nil
			case "G", "end":
				if m.contextHistoryList && m.contextFocusPane == FocusList {
					m.contextHistoryIndex = m.contextHistoryItemCount - 1
					m.contextHistoryScrollOffset = m.contextHistoryItemCount - m.contextHistoryVisibleRows()
					if m.contextHistoryScrollOffset < 0 {
						m.contextHistoryScrollOffset = 0
					}
					m.refreshContextInspectorCache()
					return m, nil
				}
				if m.contextFocusPane == FocusList {
					m.contextSubItemIndex = SubcatLast
					m.contextDetailScroll = 0
					m.refreshContextInspectorCache()
				} else {
					m.contextDetailScroll = m.contextInspectorMaxScroll()
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
					case "esc", "q":
						m.isVisualMode = false
						return m, nil
					case "y", "c", "enter":
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
							toast := fmt.Sprintf("📋 Copied %d lines to clipboard!", end-start+1)
							m.clipboardStatus = toast
							m.clipboardStatusTime = time.Now()
							m.statusMessage = toast
							m.statusMessageTime = time.Now()
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
					case "ctrl+u", "pgup":
						m.visualCursor -= availableLines / 2
						if m.visualCursor < 0 {
							m.visualCursor = 0
						}
						if m.visualCursor < m.detailScroll {
							m.detailScroll = m.visualCursor
						}
						return m, nil
					case "ctrl+d", "pgdown", " ":
						m.visualCursor += availableLines / 2
						if m.visualCursor >= totalInspectorLines {
							m.visualCursor = totalInspectorLines - 1
						}
						if m.visualCursor < 0 {
							m.visualCursor = 0
						}
						if m.visualCursor >= m.detailScroll+availableLines {
							m.detailScroll = m.visualCursor - availableLines + 1
						}
						return m, nil
					case "G", "end":
						if totalInspectorLines > 0 {
							m.visualCursor = totalInspectorLines - 1
							if m.visualCursor >= m.detailScroll+availableLines {
								m.detailScroll = m.visualCursor - availableLines + 1
							}
							if m.detailScroll < 0 {
								m.detailScroll = 0
							}
						}
						return m, nil
					case "g", "home":
						m.visualCursor = 0
						m.detailScroll = 0
						return m, nil
					default:
						return m, nil
					}
				}

				// Normal mode copy in FocusDetail
				if key == "y" || key == "c" {
					if hasEvent {
						if len(fullLines) > 0 {
							text := strings.Join(fullLines, "\n")
							_ = CopyToClipboard(text)
							toast := fmt.Sprintf("📋 Copied Step #%d detail (%d lines)!", selectedEvent.StepIndex, len(fullLines))
							m.clipboardStatus = toast
							m.clipboardStatusTime = time.Now()
							m.statusMessage = toast
							m.statusMessageTime = time.Now()
						} else {
							_ = CopyToClipboard(selectedEvent.RawContent)
							toast := fmt.Sprintf("📋 Copied Step #%d payload to clipboard!", selectedEvent.StepIndex)
							m.clipboardStatus = toast
							m.clipboardStatusTime = time.Now()
							m.statusMessage = toast
							m.statusMessageTime = time.Now()
						}
					}
					return m, nil
				}
			}

			// FocusList hotkeys for filter, search, and copy
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
				if key == "y" || key == "Y" {
					if hasEvent {
						content := selectedEvent.RawContent
						if strings.TrimSpace(content) == "" {
							content = selectedEvent.Summary
						}
						_ = CopyToClipboard(content)
						toast := fmt.Sprintf("📋 Copied Step #%d payload to clipboard!", selectedEvent.StepIndex)
						m.clipboardStatus = toast
						m.clipboardStatusTime = time.Now()
						m.statusMessage = toast
						m.statusMessageTime = time.Now()
					}
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
	case ViewInsights:
		content = m.renderInsightsView()
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
	const (
		fullNavigationWidth    = 105
		compactNavigationWidth = 70
		sessionTagLength       = 8
	)
	left := HeaderStyle.Render(" HEIMDALL ")
	labels := []string{"Dashboard", "History", "Context", "Docs", "Insights"}
	for index, label := range labels {
		if m.width < fullNavigationWidth && index == int(ViewDashboard) {
			label = "Home"
		}
		if m.width < compactNavigationWidth && index != int(m.activeView) {
			label = ""
		}
		text := fmt.Sprintf(" [%d]", index+1)
		if label != "" {
			text += " " + label
		}
		style := lipgloss.NewStyle().Foreground(insightsCaptionColor)
		if index == int(m.activeView) {
			style = style.Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg)
		}
		left += style.Render(text)
	}
	shortID := truncateVisualWidth(m.sessionID, sessionTagLength)
	right := lipgloss.NewStyle().Foreground(ColorHighlight).Render(shortID)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap > 1 {
		return left + strings.Repeat(" ", gap) + right
	}
	return truncateVisualWidth(left, m.width)
}

func (m Model) renderFooter() string {
	if m.isCommandMode {
		prompt := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(":" + m.commandInput + "█")
		hint := lipgloss.NewStyle().Foreground(ColorLightText).Render("  (type 'q' + Enter to quit, 'w' to export, Esc to cancel)")
		return lipgloss.NewStyle().MaxWidth(m.width).Render(prompt + hint)
	}

	if m.statusMessage != "" && time.Since(m.statusMessageTime) < statusMessageDuration {
		alert := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorSuccess).Padding(0, 1).Render(m.statusMessage)
		return lipgloss.NewStyle().MaxWidth(m.width).Render(alert)
	}
	if m.sourceLoading {
		return truncateVisualWidth("Loading session... current session remains available", m.width)
	}
	if m.sourceHealth.State == core.MonitorDegraded || m.sourceHealth.State == core.MonitorStopped {
		return truncateVisualWidth(fmt.Sprintf("Source %s · last read %s · %s", m.sourceHealth.State, m.sourceHealth.LastSuccess.Format(time.TimeOnly), m.sourceHealth.Error), m.width)
	}

	if m.clipboardStatus != "" && time.Since(m.clipboardStatusTime) < statusMessageDuration {
		alert := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorSuccess).Padding(0, 1).Render(m.clipboardStatus)
		return lipgloss.NewStyle().MaxWidth(m.width).Render(alert)
	}

	var hints string
	if m.activeView == ViewInsights {
		hints = " h/l Category · j/k Select · Enter Evidence · :report Export"
	} else if m.activeView == ViewContext {
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
			hints = fmt.Sprintf(" %s Yank/Copy  %s Move  %s Top/Bottom  %s Cancel",
				KeyStyle.Render("[y/Enter]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[g/G]"), KeyStyle.Render("[Esc]"))
		} else if m.focusPane == FocusList {
			if m.width < 90 {
				hints = fmt.Sprintf(" %s Focus  %s Type  %s Cache  %s Copy  %s Command  %s Shortcuts",
					KeyStyle.Render("[l]"), KeyStyle.Render("[t]"), KeyStyle.Render("[c]"), KeyStyle.Render("[y]"), KeyStyle.Render("[:] (:q)"), KeyStyle.Render("[?]"))
			} else {
				hints = fmt.Sprintf(" %s Focus  %s Step #  %s Type  %s Cache  %s Copy  %s Command  %s Shortcuts",
					KeyStyle.Render("[l]"), KeyStyle.Render("[/]"), KeyStyle.Render("[t]"), KeyStyle.Render("[c]"), KeyStyle.Render("[y]"), KeyStyle.Render("[:] (:q)"), KeyStyle.Render("[?]"))
			}
		} else {
			hints = fmt.Sprintf(" %s Visual  %s Copy All  %s List  %s Scroll  %s Top/Bottom  %s Cycle  %s Shortcuts",
				KeyStyle.Render("[v/V]"), KeyStyle.Render("[y]"), KeyStyle.Render("[h/Esc]"), KeyStyle.Render("[j/k]"), KeyStyle.Render("[g/G]"), KeyStyle.Render("[Tab]"), KeyStyle.Render("[?]"))
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
