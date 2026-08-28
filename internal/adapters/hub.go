package adapters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"heimdall/internal/adapters/antigravity"
	"heimdall/internal/core"
)

// WatcherHub manages the lifecycle of dynamic session watchers.
// It ensures only the active focused session runs a tailing goroutine,
// and gracefully stops previous watchers when switching sessions.
type WatcherHub struct {
	mu            sync.Mutex
	rootCtx       context.Context
	currentCtx    context.Context
	currentCancel context.CancelFunc
	currentSID    string
	currentType   core.AgentType
	eventChan     chan core.UnifiedAgentEvent
	analyzer      *core.PayloadAnalyzer
}

// NewWatcherHub initializes a new WatcherHub instance
func NewWatcherHub(rootCtx context.Context, eventChan chan core.UnifiedAgentEvent, analyzer *core.PayloadAnalyzer) *WatcherHub {
	return &WatcherHub{
		rootCtx:   rootCtx,
		eventChan: eventChan,
		analyzer:  analyzer,
	}
}

// ActiveSessionID returns the currently monitored session ID
func (h *WatcherHub) ActiveSessionID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.currentSID
}

// ActiveAgentType returns the currently monitored agent type
func (h *WatcherHub) ActiveAgentType() core.AgentType {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.currentType
}

// StartSession initiates watching on a specific session, stopping any previously active session watcher
func (h *WatcherHub) StartSession(sessionID string, agentType core.AgentType, customFile string, customDB string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If already watching this session, no need to restart
	if h.currentSID == sessionID && h.currentCancel != nil {
		return nil
	}

	// 1. Gracefully stop previous watcher if running
	if h.currentCancel != nil {
		h.currentCancel()
		h.currentCancel = nil
	}

	// 2. Create a dedicated child context for the new session watcher
	childCtx, cancel := context.WithCancel(h.rootCtx)
	h.currentCtx = childCtx
	h.currentCancel = cancel
	h.currentSID = sessionID
	h.currentType = agentType

	// 3. Resolve paths based on agent type
	switch agentType {
	case core.AgentTypeAntigravity, "":
		logPath := customFile
		dbPath := customDB
		home, _ := os.UserHomeDir()

		if logPath == "" {
			geminiBrain := filepath.Join(home, ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs")
			fullLog := filepath.Join(geminiBrain, "transcript_full.jsonl")
			compactLog := filepath.Join(geminiBrain, "transcript.jsonl")
			if _, err := os.Stat(fullLog); err == nil {
				logPath = fullLog
			} else {
				logPath = compactLog
			}
		}

		if dbPath == "" {
			dbPath = filepath.Join(home, ".gemini", "antigravity-cli", "conversations", sessionID+".db")
		}

		watcher := antigravity.NewWatcher(logPath, sessionID, h.analyzer, dbPath)

		// 4. Start watcher in a background goroutine
		go func(ctx context.Context, w *antigravity.Watcher, sid string) {
			_ = w.Start(ctx, h.eventChan)
		}(childCtx, watcher, sessionID)

	default:
		return fmt.Errorf("unsupported agent type for live watching: %s", agentType)
	}

	return nil
}

// SwitchSession dynamically hot-reloads watching to the target session ID
func (h *WatcherHub) SwitchSession(sessionID string, agentType core.AgentType) error {
	return h.StartSession(sessionID, agentType, "", "")
}

// Stop terminates the currently active watcher
func (h *WatcherHub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.currentCancel != nil {
		h.currentCancel()
		h.currentCancel = nil
	}
	h.currentSID = ""
}
