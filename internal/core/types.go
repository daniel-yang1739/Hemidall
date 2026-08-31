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
	StepTypeGeneric       StepType = "GENERIC"
	StepTypeError         StepType = "ERROR_MESSAGE"
	StepTypeCheckpoint    StepType = "CHECKPOINT"
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

// TokenBreakdown is Heimdall's local content estimate.
type TokenBreakdown struct {
	SystemTokens        int `json:"system_tokens"`         // System prompt, identity, rules, and environment
	ToolsDefTokens      int `json:"tools_def_tokens"`      // MCP / Tool JSON Schema definitions
	ToolResultTokens    int `json:"tool_result_tokens"`    // Code reads, terminal outputs, diffs
	HistoryTokens       int `json:"history_tokens"`        // Prior conversation turns
	ActiveTurnTokens    int `json:"active_turn_tokens"`    // Latest user prompt or active assistant output
	ThinkingTokens      int `json:"thinking_tokens"`       // Chain-of-Thought (CoT) tokens
	TotalTokens         int `json:"total_tokens"`          // Locally estimated content total
	RawLocalAccumulated int `json:"raw_local_accumulated"` // Locally estimated append-only total
	StepDelta           int `json:"step_delta"`            // Locally estimated current event contribution

}

// PersistedUsageObservation is a decoded observation from a local Antigravity
// generation record. Its schema is inferred from the persisted wire format, so
// it is not an HTTP request capture or vendor billing guarantee.
type PersistedUsageObservation struct {
	Available       bool   `json:"available"`
	Source          string `json:"source"`
	GenerationIndex int    `json:"generation_index"`
	StepIndex       int    `json:"step_index"`
	ModelName       string `json:"model_name"`

	HasTotalTokens  bool `json:"has_total_tokens"`
	TotalTokens     int  `json:"total_tokens"`
	HasCachedTokens bool `json:"has_cached_tokens"`
	CachedTokens    int  `json:"cached_tokens"`
	HasContextLimit bool `json:"has_context_limit"`
	ContextLimit    int  `json:"context_limit"`
}

// UncachedTokens returns a derived value only when both persisted operands were
// decoded. It never manufactures a cache value from earlier turns or TTL rules.
func (o PersistedUsageObservation) UncachedTokens() (int, bool) {
	if !o.HasTotalTokens || !o.HasCachedTokens || o.CachedTokens > o.TotalTokens {
		return 0, false
	}
	return o.TotalTokens - o.CachedTokens, true
}

// CacheHitRate returns a derived ratio only when both persisted operands were
// decoded and the observed total is positive.
func (o PersistedUsageObservation) CacheHitRate() (float64, bool) {
	if !o.HasTotalTokens || !o.HasCachedTokens || o.TotalTokens <= 0 || o.CachedTokens > o.TotalTokens {
		return 0, false
	}
	return float64(o.CachedTokens) / float64(o.TotalTokens) * 100.0, true
}

// StepScope defines the observed origin category of an agent event.
type StepScope string

const (
	ScopeUserInteraction  StepScope = "USER"
	ScopeCloudInference   StepScope = "CLOUD"
	ScopeLocalExecution   StepScope = "LOCAL"
	ScopeSubagent         StepScope = "SUBAGENT"
	ScopeSystemCompaction StepScope = "COMPACTION"
	ScopeSystemBootstrap  StepScope = "SYSTEM"
)

// ClassifyCacheStatus provides the single source of truth for cache classification across the entire codebase
func ClassifyCacheStatus(hitRate float64, cachedTokens, totalTokens int) string {
	if totalTokens == 0 {
		return ""
	}
	if cachedTokens == 0 {
		return "MISS"
	}
	if hitRate >= CacheHitRateThresholdHit {
		return "HIT"
	}
	if hitRate >= CacheHitRateThresholdPartial {
		return "PARTIAL"
	}
	return "MISS"
}

// UnifiedAgentEvent is the standardized domain event model across agent backends
type UnifiedAgentEvent struct {
	SessionID  string    `json:"session_id"`  // Unique session identifier
	StepIndex  int       `json:"step_index"`  // 0-indexed step sequence
	Timestamp  time.Time `json:"timestamp"`   // Event creation timestamp
	Source     string    `json:"source"`      // USER_EXPLICIT, MODEL, SYSTEM
	Type       StepType  `json:"type"`        // Step category
	Status     string    `json:"status"`      // DONE, RUNNING, ERROR
	Scope      StepScope `json:"scope"`       // USER, CLOUD, LOCAL, SUBAGENT, COMPACTION, SYSTEM
	AgentRole  string    `json:"agent_role"`  // MAIN, SUBAGENT, INTERNAL
	IsSubagent bool      `json:"is_subagent"` // True if executed by a subagent worker

	// Causality & Hierarchy Linkage
	ParentStepIdx       int   `json:"parent_step_idx,omitempty"`       // The triggering parent step index
	PackagedInStepIdx   int   `json:"packaged_in_step_idx,omitempty"`  // Cloud step index where this local step was billed
	ConsumedStepIndices []int `json:"consumed_step_indices,omitempty"` // Local step indices consumed by this cloud turn

	// Content and summaries
	Summary     string           `json:"summary"`            // Single-line summary for CLI display
	RawContent  string           `json:"raw_content"`        // Full text payload
	Thinking    string           `json:"thinking,omitempty"` // Reasoning chain
	ToolCalls   []ToolCallInfo   `json:"tool_calls,omitempty"`
	ToolResults []ToolResultInfo `json:"tool_results,omitempty"`

	// Tokens is a local cl100k_base estimate injected by Analyzer. Usage is the
	// separately persisted generation metadata observation, when available.
	Tokens      TokenBreakdown            `json:"tokens"`
	Usage       PersistedUsageObservation `json:"usage"`
	CacheStatus string                    `json:"cache_status"` // HIT, PARTIAL, MISS, UNKNOWN
}

// GetAgentRole returns the authoritative role of the agent executing this step (MAIN, SUBAGENT, or INTERNAL)
func (e UnifiedAgentEvent) GetAgentRole() string {
	if e.AgentRole != "" {
		return e.AgentRole
	}
	if e.IsSubagent || e.Scope == ScopeSubagent {
		return "SUBAGENT"
	}
	if e.Source == "SYSTEM" || e.Scope == ScopeSystemBootstrap || e.Scope == ScopeSystemCompaction {
		return "INTERNAL"
	}
	return "MAIN"
}

// IsLocalStep returns true if the step is an offline execution on the local host machine
func (e UnifiedAgentEvent) IsLocalStep() bool {
	return e.Scope == ScopeLocalExecution ||
		e.Type == StepTypeRunCommand ||
		e.Type == StepTypeViewFile ||
		e.Type == StepTypeCodeAction ||
		e.Type == StepTypeListDirectory ||
		e.Type == StepTypeToolResult ||
		e.Type == StepTypeAskQuestion ||
		e.Type == StepTypeGeneric ||
		e.Type == StepTypeError ||
		e.Type == StepTypeUnknown
}

// IsCompactionStep returns true if the step is a system-injected context compaction/checkpoint event
func (e UnifiedAgentEvent) IsCompactionStep() bool {
	return e.Scope == ScopeSystemCompaction || e.Type == StepTypeCheckpoint || e.Type == "CHECKPOINT"
}

// IsCloudStep returns true if the step is a remote LLM generation / decision turn
func (e UnifiedAgentEvent) IsCloudStep() bool {
	return !e.IsLocalStep() && !e.IsCompactionStep() && e.Type != StepTypeUserInput &&
		(e.Scope == ScopeCloudInference || e.Type == StepTypeModelResponse || e.Type == StepTypeToolCall)
}

// ModelTokenStats holds only aggregates derived from persisted usage observations.
type ModelTokenStats struct {
	ModelName        string  `json:"model_name"`
	TurnCount        int     `json:"turn_count"`
	CachedTurnCount  int     `json:"cached_turn_count"`
	TotalProcessed   int     `json:"total_processed"`
	TotalCached      int     `json:"total_cached"`
	TotalNew         int     `json:"total_new"`
	ComparableTokens int     `json:"comparable_tokens"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	CompleteUsage    bool    `json:"complete_usage"`
}

// SessionAggregateMetrics holds session-wide aggregate stats across all models and per-model
type SessionAggregateMetrics struct {
	TotalStats ModelTokenStats   `json:"total_stats"`
	ModelStats []ModelTokenStats `json:"model_stats"`
}
