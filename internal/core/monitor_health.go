package core

import "time"

type MonitorState string

const (
	MonitorHealthy  MonitorState = "healthy"
	MonitorDegraded MonitorState = "degraded"
	MonitorStopped  MonitorState = "stopped"
	MonitorSnapshot MonitorState = "snapshot"
)

// MonitorHealth distinguishes source availability from agent activity.
type MonitorHealth struct {
	State       MonitorState `json:"state"`
	LastSuccess time.Time    `json:"last_success"`
	Error       string       `json:"error,omitempty"`
}

// SessionUpdate is a complete immutable snapshot of one activation epoch.
type SessionUpdate struct {
	Epoch              uint64
	Ready              bool
	Session            Session
	Health             MonitorHealth
	Loading            bool
	RequestedSessionID string
	SwitchError        string
}
