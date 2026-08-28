package core

import "time"

// AgentType defines the type of AI Agent
type AgentType string

const (
	AgentTypeAntigravity AgentType = "antigravity"
	AgentTypeClaudeCode  AgentType = "claudecode"
	AgentTypeOpenCode    AgentType = "opencode"
)

// SessionInfo encapsulates universal session metadata across different agent engines
type SessionInfo struct {
	AgentType    AgentType
	SessionID    string
	WorkspaceDir string
	ShortPath    string
	InitialGoal  string
	LastPrompt   string
	StepCount    int
	LastModified time.Time
	SizeMB       float64
	ModelName    string
	DBPath       string
	LogPath      string
}
