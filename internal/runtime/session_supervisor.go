package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	agents "heimdall/internal/agent_adapters"
	"heimdall/internal/agent_adapters/antigravity"
	"heimdall/internal/core"
)

const (
	refreshInterval      = 250 * time.Millisecond
	maximumRetryInterval = 4 * time.Second
	retryMultiplier      = 2
	healthHeartbeat      = time.Second
	updateCapacity       = 1
)

var errSuperseded = errors.New("session load superseded")

// SessionSupervisor owns one active source and publishes replaceable, complete
// snapshots. A slow consumer can coalesce updates without losing removals.
type SessionSupervisor struct {
	mu             sync.RWMutex
	rootContext    context.Context
	homeDirectory  string
	requestedEpoch uint64
	activeEpoch    uint64
	cancelActive   context.CancelFunc
	cancelLoading  context.CancelFunc
	query          *core.QueryService
	analysis       *core.AnalysisService
	updates        chan core.SessionUpdate
	current        core.SessionUpdate
}

func NewSessionSupervisor(ctx context.Context) (*SessionSupervisor, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	return newSessionSupervisor(ctx, home), nil
}

func newSessionSupervisor(ctx context.Context, home string) *SessionSupervisor {
	return &SessionSupervisor{rootContext: ctx, homeDirectory: home, updates: make(chan core.SessionUpdate, updateCapacity)}
}

func (s *SessionSupervisor) Updates() <-chan core.SessionUpdate { return s.updates }

func (s *SessionSupervisor) SwitchSession(id string) error { return s.StartSession(id, "", "") }

func (s *SessionSupervisor) StartSession(id, transcript, database string) error {
	s.mu.Lock()
	s.requestedEpoch++
	if s.cancelLoading != nil {
		s.cancelLoading()
	}
	loadingContext, cancelLoading := context.WithCancel(s.rootContext)
	s.cancelLoading = cancelLoading
	ticket := s.requestedEpoch
	s.current.RequestedSessionID = id
	s.current.SwitchError = ""
	s.current.Loading = true
	s.publishLocked()
	s.mu.Unlock()

	query, err := loadSession(loadingContext, s.homeDirectory, id, transcript, database)
	s.mu.Lock()
	defer s.mu.Unlock()
	if ticket != s.requestedEpoch || s.rootContext.Err() != nil {
		cancelLoading()
		return errSuperseded
	}
	s.current.Loading = false
	s.cancelLoading = nil
	if err != nil {
		cancelLoading()
		s.current.SwitchError = err.Error()
		s.publishLocked()
		return err
	}
	if s.cancelActive != nil {
		s.cancelActive()
	}
	s.cancelActive, s.activeEpoch = cancelLoading, ticket
	s.query, s.analysis = query, core.NewAnalysisService(query)
	s.current = core.SessionUpdate{
		Epoch: ticket, Ready: true, Session: query.Session(),
		Health: core.MonitorHealth{State: core.MonitorHealthy, LastSuccess: time.Now()},
	}
	s.publishLocked()
	go s.run(loadingContext, query, ticket)
	return nil
}

// LoadSession performs a single read, suitable for offline reports and startup.
func LoadSession(home, id, transcript, database string) (*core.QueryService, error) {
	return loadSession(context.Background(), home, id, transcript, database)
}

func loadSession(ctx context.Context, home, id, transcript, database string) (*core.QueryService, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var sources []agents.SourceRef
	if transcript == "" && database == "" {
		var err error
		sources, err = antigravity.DiscoverSessionSources(home, id)
		if err != nil {
			return nil, err
		}
	} else {
		if transcript != "" {
			sources = append(sources, agents.SourceRef{Kind: agents.SourceKindTranscript, Path: transcript})
		}
		if database != "" {
			sources = append(sources, agents.SourceRef{Kind: agents.SourceKindConversation, Path: database})
		}
	}
	store, err := antigravity.NewSessionStoreWithContext(ctx, core.SessionRef{AgentID: "antigravity", SessionID: id}, sources)
	if err != nil {
		return nil, err
	}
	query := core.NewQueryService(store)
	if _, err := query.Refresh(); err != nil {
		return nil, fmt.Errorf("hydrate session %s: %w", id, err)
	}
	return query, nil
}

func (s *SessionSupervisor) run(ctx context.Context, query *core.QueryService, epoch uint64) {
	defer s.markStopped(epoch)
	delay := refreshInterval
	timer := time.NewTimer(delay)
	defer timer.Stop()
	lastPublish := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		delta, err := query.Refresh()
		s.mu.Lock()
		if epoch != s.activeEpoch || ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		now := time.Now()
		changed := false
		if err != nil {
			changed = s.current.Health.State != core.MonitorDegraded || s.current.Health.Error != err.Error()
			s.current.Health.State, s.current.Health.Error = core.MonitorDegraded, err.Error()
			delay *= retryMultiplier
			if delay > maximumRetryInterval {
				delay = maximumRetryInterval
			}
		} else {
			changed = s.current.Health.State != core.MonitorHealthy || delta.Revision != s.current.Session.Revision
			if delta.Revision != s.current.Session.Revision {
				s.current.Session = query.Session()
			}
			s.current.Health = core.MonitorHealth{State: core.MonitorHealthy, LastSuccess: now}
			delay = refreshInterval
		}
		if changed || now.Sub(lastPublish) >= healthHeartbeat {
			s.publishLocked()
			lastPublish = now
		}
		s.mu.Unlock()
		timer.Reset(delay)
	}
}

func (s *SessionSupervisor) publishLocked() {
	select {
	case <-s.updates:
	default:
	}
	update := s.current
	update.Session = core.CloneSession(s.current.Session)
	s.updates <- update
}

func (s *SessionSupervisor) markStopped(epoch uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if epoch == s.activeEpoch && s.rootContext.Err() != nil {
		s.current.Health.State = core.MonitorStopped
		s.publishLocked()
	}
}

func (s *SessionSupervisor) Query() *core.QueryService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.query
}

func (s *SessionSupervisor) Analysis() *core.AnalysisService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.analysis
}
