package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/agent_adapters/antigravity"
	"heimdall/internal/core"
)

const supervisorEventChannelCapacity = 1_024

// SessionSupervisor is the composition-root coordinator for one selected
// session. It is the only runtime package that knows both an adapter and core.
type SessionSupervisor struct {
	mu            sync.RWMutex
	rootContext   context.Context
	eventChannel  chan<- core.UnifiedAgentEvent
	homeDirectory string
	cancelActive  context.CancelFunc
	activeID      string
	query         *core.QueryService
	analysis      *core.AnalysisService
}

// NewSessionSupervisor creates an adapter-aware runtime coordinator.
func NewSessionSupervisor(rootContext context.Context, eventChannel chan<- core.UnifiedAgentEvent) (*SessionSupervisor, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	return newSessionSupervisor(rootContext, eventChannel, homeDirectory), nil
}

func newSessionSupervisor(rootContext context.Context, eventChannel chan<- core.UnifiedAgentEvent, homeDirectory string) *SessionSupervisor {
	return &SessionSupervisor{rootContext: rootContext, eventChannel: eventChannel, homeDirectory: homeDirectory}
}

// SwitchSession replaces the active monitor only after the new session's
// sources and initial snapshot have been successfully parsed.
func (supervisor *SessionSupervisor) SwitchSession(sessionID string) error {
	return supervisor.startSession(sessionID, "", "")
}

// StartSession supports explicit source paths for CLI diagnostics while normal
// operation resolves the session's provider-standard artifact locations.
func (supervisor *SessionSupervisor) StartSession(sessionID, transcriptPath, databasePath string) error {
	return supervisor.startSession(sessionID, transcriptPath, databasePath)
}

func (supervisor *SessionSupervisor) startSession(sessionID, transcriptPath, databasePath string) error {
	sources, err := supervisor.sourcesFor(sessionID, transcriptPath, databasePath)
	if err != nil {
		return err
	}
	store, err := antigravity.NewSessionStore(agents.SessionRef{
		AgentID:      agents.AgentID("antigravity"),
		SessionID:    sessionID,
		DiscoveredAt: time.Now(),
	}, sources)
	if err != nil {
		return err
	}
	query := core.NewQueryService(store)
	initialDelta, err := query.Refresh()
	if err != nil {
		return fmt.Errorf("hydrate session %s: %w", sessionID, err)
	}

	supervisor.mu.Lock()
	if supervisor.cancelActive != nil {
		supervisor.cancelActive()
	}
	activeContext, cancelActive := context.WithCancel(supervisor.rootContext)
	supervisor.cancelActive = cancelActive
	supervisor.activeID = sessionID
	supervisor.query = query
	supervisor.analysis = core.NewAnalysisService(query)
	supervisor.mu.Unlock()

	supervisor.emitSession(query.Session())
	monitor := antigravity.NewMonitor(query)
	deltas := make(chan core.SessionDelta, supervisorEventChannelCapacity)
	go supervisor.forwardMonitor(activeContext, monitor, deltas, query, initialDelta.Revision)
	return nil
}

func (supervisor *SessionSupervisor) sourcesFor(sessionID, transcriptPath, databasePath string) ([]agents.SourceRef, error) {
	if transcriptPath == "" && databasePath == "" {
		return antigravity.DiscoverSessionSources(supervisor.homeDirectory, sessionID)
	}
	sources := make([]agents.SourceRef, 0, 2)
	if transcriptPath != "" {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindTranscript, Path: transcriptPath})
	}
	if databasePath != "" {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})
	}
	return sources, nil
}

func (supervisor *SessionSupervisor) forwardMonitor(ctx context.Context, monitor *antigravity.Monitor, deltas chan core.SessionDelta, query *core.QueryService, initialRevision uint64) {
	monitorErrors := make(chan error, 1)
	go func() {
		monitorErrors <- monitor.Run(ctx, deltas)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case delta := <-deltas:
			if delta.Revision <= initialRevision {
				continue
			}
			supervisor.emitSession(query.Session())
		case <-monitorErrors:
			return
		}
	}
}

// Query exposes only the core query boundary to application composition.
func (supervisor *SessionSupervisor) Query() *core.QueryService {
	supervisor.mu.RLock()
	defer supervisor.mu.RUnlock()
	return supervisor.query
}

// Analysis exposes only the query-backed analysis service.
func (supervisor *SessionSupervisor) Analysis() *core.AnalysisService {
	supervisor.mu.RLock()
	defer supervisor.mu.RUnlock()
	return supervisor.analysis
}

func (supervisor *SessionSupervisor) emitSession(session core.Session) {
	for _, event := range core.ProjectSessionEvents(session) {
		select {
		case supervisor.eventChannel <- event:
		case <-supervisor.rootContext.Done():
			return
		}
	}
}
