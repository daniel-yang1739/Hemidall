package adapters

import (
	"context"
	"fmt"
	"os"
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
	currentCancel context.CancelFunc
	currentSID    string
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

// StartSession initiates watching on a specific session, stopping any previously active session watcher
func (h *WatcherHub) StartSession(sessionID string, customFile string, customDB string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If already watching this session, no need to restart
	if h.currentSID == sessionID && h.currentCancel != nil {
		return nil
	}

	logPath := customFile
	dbPath := customDB
	home, _ := os.UserHomeDir()

	if logPath == "" {
		fullLog, compactLog := antigravity.TranscriptPaths(home, sessionID)
		if _, err := os.Stat(fullLog); err == nil {
			logPath = fullLog
		} else {
			logPath = compactLog
		}
	}

	if dbPath == "" {
		dbPath = antigravity.ConversationDatabasePath(home, sessionID)
	}
	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("resolve session transcript %s: %w", sessionID, err)
	}

	// Stop the previous watcher only after the replacement transcript is valid.
	if h.currentCancel != nil {
		h.currentCancel()
		h.currentCancel = nil
	}

	childCtx, cancel := context.WithCancel(h.rootCtx)
	h.currentCancel = cancel
	h.currentSID = sessionID

	watcher := antigravity.NewWatcher(logPath, sessionID, h.analyzer, dbPath)

	// 4. Start watcher in a background goroutine
	go func(ctx context.Context, w *antigravity.Watcher) {
		_ = w.Start(ctx, h.eventChan)
	}(childCtx, watcher)

	return nil
}

// SwitchSession dynamically hot-reloads watching to the target session ID
func (h *WatcherHub) SwitchSession(sessionID string) error {
	return h.StartSession(sessionID, "", "")
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
