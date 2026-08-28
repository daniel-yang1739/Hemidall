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

// TokenBreakdown defines the 5-dimension token distribution and cache metrics
type TokenBreakdown struct {
	SystemTokens     int     `json:"system_tokens"`      // System prompt, identity, rules, and environment
	ToolsDefTokens   int     `json:"tools_def_tokens"`   // MCP / Tool JSON Schema definitions
	ToolResultTokens int     `json:"tool_result_tokens"` // Code reads, terminal outputs, diffs
	HistoryTokens    int     `json:"history_tokens"`     // Prior conversation turns
	ActiveTurnTokens int     `json:"active_turn_tokens"` // Latest user prompt or active assistant output
	ThinkingTokens   int     `json:"thinking_tokens"`    // Chain-of-Thought (CoT) tokens
	TotalTokens      int     `json:"total_tokens"`       // Total context tokens (Active Window)
	
	// Prefix Caching analytical metrics
	CachedTokens     int     `json:"cached_tokens"`      // Prefix cache hit tokens
	NewTokens        int     `json:"new_tokens"`         // New uncached tokens in this step
	CacheHitRate     float64 `json:"cache_hit_rate"`     // Cache hit rate percentage (%)
	
	// Official Google API telemetry fields
	IsOfficialData       bool    `json:"is_official_data"`       // True if fetched directly from SQLite gen_metadata
	OfficialModel        string  `json:"official_model"`         // Actual backend model (e.g. gemini-3.7-flash-high)
	OfficialContextLimit int     `json:"official_context_limit"` // e.g. 256,000
	RawLocalAccumulated  int     `json:"raw_local_accumulated"`  // Raw uncompressed log tokens (e.g. ~400k)
}

// StepScope defines the universal computing origin and billing nature of an agent event
type StepScope string

const (
	ScopeUserInteraction  StepScope = "USER"       // 👤 User Intent / Prompts (Network billed)
	ScopeCloudInference   StepScope = "CLOUD"      // ☁️ Cloud LLM Inference / Tool Calls (GPU billed)
	ScopeLocalExecution   StepScope = "LOCAL"      // 💻 Local Machine Process (Offline, 0 tokens)
	ScopeSubagent         StepScope = "SUBAGENT"   // 👥 Subagent Worker Execution / Parallel Inference
	ScopeSystemCompaction StepScope = "COMPACTION" // ⚙️ Out-of-band context compaction & truncation injection
	ScopeSystemBootstrap  StepScope = "SYSTEM"     // 📜 System Init / Rules / Static configurations
)

// ClassifyCacheStatus provides the single source of truth for cache classification across the entire codebase
func ClassifyCacheStatus(hitRate float64, cachedTokens, totalTokens int, isExpired bool) string {
	if isExpired && cachedTokens == 0 {
		return "EXPIRED"
	}
	if totalTokens == 0 {
		return ""
	}
	if cachedTokens == 0 {
		return "MISS"
	}
	if hitRate >= 80.0 {
		return "HIT"
	}
	if hitRate > 0.0 {
		return "PARTIAL"
	}
	return "MISS"
}

// UnifiedAgentEvent is the standardized domain event model across agent backends
type UnifiedAgentEvent struct {
	SessionID   string           `json:"session_id"`   // Unique session identifier
	StepIndex   int              `json:"step_index"`   // 0-indexed step sequence
	Timestamp   time.Time        `json:"timestamp"`    // Event creation timestamp
	Source      string           `json:"source"`       // USER_EXPLICIT, MODEL, SYSTEM
	Type        StepType         `json:"type"`         // Step category
	Status      string           `json:"status"`       // DONE, RUNNING, ERROR
	Scope       StepScope        `json:"scope"`        // USER, CLOUD, LOCAL, SUBAGENT, COMPACTION, SYSTEM
	AgentRole   string           `json:"agent_role"`   // MAIN, SUBAGENT, INTERNAL
	IsSubagent  bool             `json:"is_subagent"`  // True if executed by a subagent worker

	// Causality & Hierarchy Linkage
	ParentStepIdx       int      `json:"parent_step_idx,omitempty"`        // The triggering parent step index
	PackagedInStepIdx   int      `json:"packaged_in_step_idx,omitempty"`    // Cloud step index where this local step was billed
	ConsumedStepIndices []int    `json:"consumed_step_indices,omitempty"`  // Local step indices consumed by this cloud turn
	
	// Content and summaries
	Summary     string           `json:"summary"`      // Single-line summary for CLI display
	RawContent  string           `json:"raw_content"`  // Full text payload
	Thinking    string           `json:"thinking,omitempty"` // Reasoning chain
	ToolCalls   []ToolCallInfo   `json:"tool_calls,omitempty"`
	ToolResults []ToolResultInfo `json:"tool_results,omitempty"`
	
	// 5-dimension breakdown (injected by Analyzer)
	Tokens      TokenBreakdown   `json:"tokens"`
	CacheStatus string           `json:"cache_status"` // HIT, PARTIAL, WRITE, EXPIRED, MISS, UNKNOWN
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

// ModelTokenStats holds aggregate token metrics and effective pricing for a specific model
type ModelTokenStats struct {
	ModelName         string  `json:"model_name"`
	TurnCount         int     `json:"turn_count"`
	TotalProcessed    int     `json:"total_processed"`
	TotalCached       int     `json:"total_cached"`
	TotalNew          int     `json:"total_new"`
	CacheHitRate      float64 `json:"cache_hit_rate"`
	DiscountRate      float64 `json:"discount_rate"` // e.g. 0.75 for 75% OFF
	PriceFactor       float64 `json:"price_factor"`  // e.g. 0.25
	DiscountLabel     string  `json:"discount_label"` // e.g. "0.25x (75% OFF)"
	EffectiveTokens   int     `json:"effective_tokens"`
	TokensSaved       int     `json:"tokens_saved"`
	SavingsPercentage float64 `json:"savings_percentage"`
}

// SessionAggregateMetrics holds session-wide aggregate stats across all models and per-model
type SessionAggregateMetrics struct {
	TotalStats ModelTokenStats   `json:"total_stats"`
	ModelStats []ModelTokenStats `json:"model_stats"`
}

// TurnTrendPoint holds telemetry metrics for a single cloud turn to be plotted in trend sparklines
type TurnTrendPoint struct {
	StepIndex    int     `json:"step_index"`
	TotalTokens  int     `json:"total_tokens"`
	CachedTokens int     `json:"cached_tokens"`
	NewTokens    int     `json:"new_tokens"`
	CacheHitRate float64 `json:"cache_hit_rate"`
}

// TurnTrendSeries holds the chronological sequence of cloud turns for trend visualization
type TurnTrendSeries struct {
	Points       []TurnTrendPoint `json:"points"`
	MaxContext   int              `json:"max_context"`
	PeakNew      int              `json:"peak_new"`
	AvgHitRate   float64          `json:"avg_hit_rate"`
	LatestCached int              `json:"latest_cached"`
}

