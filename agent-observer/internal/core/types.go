package core

import (
	"time"
)

// StepType represents the category of an agent execution step
type StepType string

const (
	StepTypeUserInput     StepType = "USER_INPUT"
	StepTypeModelResponse StepType = "MODEL_RESPONSE"
	StepTypeToolCall      StepType = "TOOL_CALL"
	StepTypeToolResult    StepType = "TOOL_RESULT"
	StepTypeSystemInit    StepType = "SYSTEM_INIT"
	StepTypeRunCommand    StepType = "RUN_COMMAND"
	StepTypeViewFile      StepType = "VIEW_FILE"
	StepTypeCodeAction    StepType = "CODE_ACTION"
	StepTypeListDirectory StepType = "LIST_DIRECTORY"
	StepTypeAskQuestion   StepType = "ASK_QUESTION"
	StepTypeUnknown       StepType = "UNKNOWN"
)

// ToolCallInfo represents metadata for a single tool call invocation
type ToolCallInfo struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	RawArgs   string                 `json:"raw_args,omitempty"`
}

// ToolResultInfo represents the execution output of a tool
type ToolResultInfo struct {
	ToolName string `json:"tool_name"`
	Output   string `json:"output"`
	IsError  bool   `json:"is_error,omitempty"`
}

// TokenBreakdown defines the 5-dimension token distribution and cache metrics
type TokenBreakdown struct {
	SystemTokens     int     `json:"system_tokens"`      // System prompt, identity, rules, and environment
	ToolsDefTokens   int     `json:"tools_def_tokens"`   // MCP / Tool JSON Schema definitions
	ToolResultTokens int     `json:"tool_result_tokens"` // Code reads, terminal outputs, diffs
	HistoryTokens    int     `json:"history_tokens"`     // Prior conversation turns
	ActiveTurnTokens int     `json:"active_turn_tokens"` // Latest user prompt or active assistant output
	ThinkingTokens   int     `json:"thinking_tokens"`    // Chain-of-Thought (CoT) tokens
	TotalTokens      int     `json:"total_tokens"`       // Total context tokens
	
	// Prefix Caching analytical metrics
	CachedTokens     int     `json:"cached_tokens"`      // Theoretical prefix cache hit tokens
	NewTokens        int     `json:"new_tokens"`         // New uncached tokens in this step
	CacheHitRate     float64 `json:"cache_hit_rate"`     // Cache hit rate percentage (%)
}

// UnifiedAgentEvent is the standardized domain event model across agent backends
type UnifiedAgentEvent struct {
	SessionID   string           `json:"session_id"`   // Unique session identifier
	StepIndex   int              `json:"step_index"`   // 0-indexed step sequence
	Timestamp   time.Time        `json:"timestamp"`    // Event creation timestamp
	Source      string           `json:"source"`       // USER_EXPLICIT, MODEL, SYSTEM
	Type        StepType         `json:"type"`         // Step category
	Status      string           `json:"status"`       // DONE, RUNNING, ERROR
	
	// Content and summaries
	Summary     string           `json:"summary"`      // Single-line summary for CLI display
	RawContent  string           `json:"raw_content"`  // Full text payload
	Thinking    string           `json:"thinking,omitempty"` // Reasoning chain
	ToolCalls   []ToolCallInfo   `json:"tool_calls,omitempty"`
	ToolResults []ToolResultInfo `json:"tool_results,omitempty"`
	
	// 5-dimension breakdown (injected by Analyzer)
	Tokens      TokenBreakdown   `json:"tokens"`
	CacheStatus string           `json:"cache_status"` // HIT, MISS, WRITE, UNKNOWN
}
