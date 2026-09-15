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

// PersistedUsageObservation contains schema-inferred observations decoded from
// a local Antigravity generation record. UncachedInputTokens and
// CachedInputTokens share one schema-inferred usage-message path; they support
// the cache-adjusted input projection. ObservedContextTokens comes from a
// separate context-state path and is kept separate from input usage.
type PersistedUsageObservation struct {
	HasThinkingOutputTokens bool         `json:"has_thinking_output_tokens"`
	HasOutputContentTokens  bool         `json:"has_output_content_tokens"`
	Available               bool         `json:"available"`
	Source                  string       `json:"source"`
	GenerationIndex         int          `json:"generation_index"`
	StepIndex               int          `json:"step_index"`
	Provider                ProviderName `json:"provider,omitempty"`
	ModelName               string       `json:"model_name"`
	Model                   ModelID      `json:"model,omitempty"`

	HasObservedContextTokens bool `json:"has_observed_context_tokens"`
	ObservedContextTokens    int  `json:"observed_context_tokens"`
	HasUncachedInputTokens   bool `json:"has_uncached_input_tokens"`
	UncachedInputTokens      int  `json:"uncached_input_tokens"`
	HasCachedInputTokens     bool `json:"has_cached_input_tokens"`
	CachedInputTokens        int  `json:"cached_input_tokens"`
	HasContextLimit          bool `json:"has_context_limit"`
	ContextLimit             int  `json:"context_limit"`

	ThinkingOutputTokens int    `json:"thinking_output_tokens,omitempty"`
	OutputContentTokens  int    `json:"output_content_tokens,omitempty"`
	TotalTokens          int    `json:"total_tokens,omitempty"`
	TimeToFirstTokenMs   int64  `json:"time_to_first_token_ms,omitempty"`
	StreamingDurationMs  int64  `json:"streaming_duration_ms,omitempty"`
	UpstreamRequestID    string `json:"upstream_request_id,omitempty"`
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
	Tokens TokenBreakdown            `json:"tokens"`
	Usage  PersistedUsageObservation `json:"usage"`
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

// GetMessageRole returns the authoritative conversational role (USER, ASSISTANT, TOOL_RESULT, or SYSTEM)
func (e UnifiedAgentEvent) GetMessageRole() string {
	if e.IsCloudStep() {
		return "ASSISTANT"
	}
	if e.IsLocalStep() {
		return "TOOL_RESULT"
	}
	if e.Type == StepTypeUserInput || e.Scope == ScopeUserInteraction {
		return "USER"
	}
	if e.IsCompactionStep() {
		return "SYSTEM (CHECKPOINT)"
	}
	return "SYSTEM"
}

// GetMessageRoleDescription returns a human-friendly description of the role in the LLM conversation lifecycle.
func (e UnifiedAgentEvent) GetMessageRoleDescription() string {
	switch e.GetMessageRole() {
	case "ASSISTANT":
		return "Model Turn / Tool Call"
	case "TOOL_RESULT":
		return "Tool Execution Output"
	case "USER":
		return "Inbound User Prompt"
	case "SYSTEM (CHECKPOINT)":
		return "Compacted Context Checkpoint"
	default:
		return "System Guidelines / Bootstrap"
	}
}

// ModelTokenStats holds aggregates from one schema-inferred input-usage message.
// CachedInputTokenSum is a cached-input counter, not a provider invoice.
type ModelTokenStats struct {
	Provider                      ProviderName `json:"provider,omitempty"`
	ModelName                     string       `json:"model_name"`
	Model                         ModelID      `json:"model,omitempty"`
	TurnCount                     int          `json:"turn_count"`
	UncachedInputTokenSum         int          `json:"uncached_input_token_sum"`
	CachedInputTokenSum           int          `json:"cached_input_token_sum"`
	TotalProcessedTokenSum        int          `json:"total_processed_token_sum"`
	ExplicitCacheValueTurnCount   int          `json:"explicit_cache_value_turn_count"`
	InferredZeroCacheTurnCount    int          `json:"inferred_zero_cache_turn_count"`
	CacheInputSharePercent        float64      `json:"cache_input_share_percent"`
	ObservedContextTokenSum       int          `json:"observed_context_token_sum"`
	ObservedContextValueTurnCount int          `json:"observed_context_value_turn_count"`
	TotalOutputTokenSum           int          `json:"total_output_token_sum"`
	ThinkingOutputTokenSum        int          `json:"thinking_output_token_sum"`
	ContentOutputTokenSum         int          `json:"content_output_token_sum"`
	EstimatedCostUSD              float64      `json:"estimated_cost_usd"`
	HasEstimatedCost              bool         `json:"has_estimated_cost"`
}

// SessionAggregateMetrics holds session-wide aggregate stats across all models and per-model
type SessionAggregateMetrics struct {
	TotalStats ModelTokenStats   `json:"total_stats"`
	ModelStats []ModelTokenStats `json:"model_stats"`
}
