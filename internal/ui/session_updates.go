package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"heimdall/internal/core"
	"time"
)

type SessionUpdateMsg core.SessionUpdate

type sessionAnalysisMsg struct {
	Epoch     uint64
	Revision  uint64
	SessionID string
	Events    []core.UnifiedAgentEvent
	Report    core.SessionAnalysisReport
	Dashboard core.DashboardReadModel
	Payload   core.AgentContextPayload
	Estimate  core.VisibleContextEvidenceEstimate
	Playback  []core.VisibleContextEvidenceEstimate
}

type sessionSwitchResultMsg struct {
	Request uint64
	Err     error
}

func (m *Model) receiveSessionUpdate(update core.SessionUpdate) tea.Cmd {
	if update.Epoch < m.sessionEpoch || (m.pendingSessionUpdate != nil && update.Epoch < m.pendingSessionUpdate.Epoch) {
		return nil
	}
	m.runtimeManaged = true
	m.sourceHealth, m.sourceLoading = update.Health, update.Loading
	if m.reportReady && update.Epoch == m.sessionEpoch {
		m.analysisReport.Health = update.Health
	}
	if update.SwitchError != "" {
		m.statusMessage, m.statusMessageTime = update.SwitchError, time.Now()
	}
	if !update.Ready {
		return nil
	}
	if update.Epoch == m.sessionEpoch && update.Session.Revision <= m.sessionRevision {
		return nil
	}
	if m.pendingSessionUpdate != nil && update.Epoch == m.pendingSessionUpdate.Epoch && update.Session.Revision <= m.pendingSessionUpdate.Session.Revision {
		return nil
	}
	m.pendingSessionUpdate = &update
	return m.requestSessionAnalysis()
}

func (m *Model) requestSessionAnalysis() tea.Cmd {
	if m.sessionAnalysisPending || m.pendingSessionUpdate == nil {
		return nil
	}
	m.sessionAnalysisPending = true
	update := *m.pendingSessionUpdate
	return func() tea.Msg {
		session := update.Session
		events := core.ProjectSessionEvents(session)
		report := core.BuildSessionAnalysisReport(session)
		report.Health = update.Health
		dashboard := core.BuildDashboardReadModel(session)
		dashboard.Metrics = report.Metrics
		payload := core.BuildContextPayloadFromSession(session, core.ContextBuildInput{History: events, SessionID: session.Ref.SessionID})
		return sessionAnalysisMsg{Epoch: update.Epoch, Revision: session.Revision, SessionID: session.Ref.SessionID,
			Events: events, Report: report, Dashboard: dashboard, Payload: payload,
			Estimate: core.EstimateVisibleContextEvidence(payload), Playback: core.BuildPlaybackContextEvidence(events)}
	}
}

func (m *Model) receiveSessionAnalysis(msg sessionAnalysisMsg) tea.Cmd {
	m.sessionAnalysisPending = false
	if m.pendingSessionUpdate == nil {
		return nil
	}
	if msg.Epoch != m.pendingSessionUpdate.Epoch || msg.Revision < m.sessionRevision && msg.Epoch == m.sessionEpoch {
		return m.requestSessionAnalysis()
	}
	changedSession := msg.Epoch != m.sessionEpoch
	selectedKey, dashboardKey := "", ""
	if selected, ok := m.getSelectedEvent(); ok {
		selectedKey = historyEventKey(selected)
	}
	followLatest := len(m.history) == 0 || m.dashboardIdx == len(m.history)-1
	if m.dashboardIdx >= 0 && m.dashboardIdx < len(m.history) {
		dashboardKey = historyEventKey(m.history[m.dashboardIdx])
	}
	m.sessionID, m.sessionEpoch, m.sessionRevision = msg.SessionID, msg.Epoch, msg.Revision
	m.history, m.historyIndex = msg.Events, buildHistoryIndex(msg.Events)
	m.latestEvent = core.UnifiedAgentEvent{}
	if len(m.history) > 0 {
		m.latestEvent = m.history[len(m.history)-1]
	}
	if changedSession {
		m.selectedIdx, m.dashboardIdx, m.detailScroll, m.historyOffset = 0, 0, 0, 0
		m.insightsIndex, m.insightsGroup = 0, 0
		m.historyTypeFilter, m.historyCacheFilter, m.historyStepQuery = TypeFilterAll, CacheFilterAll, ""
		m.isSessionSwitcherOpen = false
		followLatest = true
	}
	if followLatest {
		m.dashboardIdx = max(0, len(m.history)-1)
	} else if index, ok := m.historyIndex[dashboardKey]; ok {
		m.dashboardIdx = index
	} else {
		m.dashboardIdx = max(0, min(m.dashboardIdx, len(m.history)-1))
	}
	if !changedSession {
		filtered := m.getFilteredHistory()
		m.selectedIdx = max(0, min(m.selectedIdx, len(filtered)-1))
		for i, event := range filtered {
			if historyEventKey(event) == selectedKey {
				m.selectedIdx = len(filtered) - 1 - i
				break
			}
		}
	}
	m.contextRevision++
	m.contextPayload, m.contextPayloadReady, m.contextPayloadRevision = msg.Payload, true, m.contextRevision
	m.contextEstimate, m.contextEstimateHistoryCount = msg.Estimate, len(msg.Events)
	m.contextInspectorLines = nil
	m.dashboardReadModel = msg.Dashboard
	m.dashboardReadModelHistoryCount = len(msg.Events)
	m.dashboardReadModelPending, m.dashboardReadModelDirty = false, false
	m.playbackContextEstimates, m.playbackEstimateHistoryCount = msg.Playback, len(msg.Events)
	m.playbackEstimateCachedRevision = m.playbackEstimateRevision
	m.analysisReport, m.reportReady = msg.Report, true
	_, insightItems := m.insightItems()
	m.insightsIndex = max(0, min(m.insightsIndex, min(len(insightItems), core.InsightsRankingLimit)-1))
	m.analysisReport.Health = m.sourceHealth
	if m.activeView == ViewContext {
		m.refreshContextInspectorCache()
	}
	if m.pendingSessionUpdate.Session.Revision == msg.Revision {
		m.pendingSessionUpdate = nil
	}
	return m.requestSessionAnalysis()
}
