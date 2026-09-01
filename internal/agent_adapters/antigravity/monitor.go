package antigravity

import (
	"context"
	"time"

	"heimdall/internal/core"
)

const monitorRefreshInterval = 250 * time.Millisecond

// Monitor owns the single refresh loop for one active SessionStore.
type Monitor struct {
	query    *core.QueryService
	interval time.Duration
}

// NewMonitor creates the single active-session monitor.
func NewMonitor(query *core.QueryService) *Monitor {
	return &Monitor{query: query, interval: monitorRefreshInterval}
}

// Run refreshes the store and emits deltas only after source state changes.
func (monitor *Monitor) Run(ctx context.Context, deltas chan<- core.SessionDelta) error {
	ticker := time.NewTicker(monitor.interval)
	defer ticker.Stop()
	var lastRevision uint64
	for {
		delta, err := monitor.query.Refresh()
		if err != nil {
			return err
		}
		if delta.Revision > lastRevision {
			select {
			case deltas <- delta:
				lastRevision = delta.Revision
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
