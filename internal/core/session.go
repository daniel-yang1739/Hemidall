package core

import "time"

// SessionInfo contains Antigravity session metadata for the switcher catalog.
type SessionInfo struct {
	SessionID          string
	WorkspaceDir       string
	ShortPath          string
	InitialGoal        string
	LastPrompt         string
	StepCount          int
	StepCountAvailable bool
	LastModified       time.Time
	SizeMB             float64
	ModelName          string
	DBPath             string
	LogPath            string
}
