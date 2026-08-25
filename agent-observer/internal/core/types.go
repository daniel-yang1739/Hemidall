package core

import (
	"time"
)

// StepType 表示 Agent 執行的步驟類型
type StepType string

const (
	StepTypeUserInput        StepType = "USER_INPUT"
	StepTypeModelResponse    StepType = "MODEL_RESPONSE"
	StepTypeToolCall         StepType = "TOOL_CALL"
	StepTypeToolResult       StepType = "TOOL_RESULT"
	StepTypeSystemInit       StepType = "SYSTEM_INIT"
	StepTypeRunCommand       StepType = "RUN_COMMAND"
	StepTypeViewFile         StepType = "VIEW_FILE"
	StepTypeCodeAction       StepType = "CODE_ACTION"
	StepTypeListDirectory    StepType = "LIST_DIRECTORY"
	StepTypeAskQuestion      StepType = "ASK_QUESTION"
	StepTypeUnknown          StepType = "UNKNOWN"
)

// ToolCallInfo 代表單一工具呼叫資訊
type ToolCallInfo struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	RawArgs   string                 `json:"raw_args,omitempty"`
}

// ToolResultInfo 代表單一工具執行的返回結果
type ToolResultInfo struct {
	ToolName string `json:"tool_name"`
	Output   string `json:"output"`
	IsError  bool   `json:"is_error,omitempty"`
}

// TokenBreakdown 定義各區塊的 Token 統計
type TokenBreakdown struct {
	SystemTokens     int     `json:"system_tokens"`      // 系統提示詞、環境設定、靜態規則
	ToolsDefTokens   int     `json:"tools_def_tokens"`   // MCP / Tool JSON Schema 定義
	ToolResultTokens int     `json:"tool_result_tokens"` // 程式碼讀取、終端日誌、Diff等工具輸出
	HistoryTokens    int     `json:"history_tokens"`     // 過去對話回合
	ActiveTurnTokens int     `json:"active_turn_tokens"` // 最新一輪輸入或當前輸出
	ThinkingTokens   int     `json:"thinking_tokens"`    // 思考鏈 (CoT) Token
	TotalTokens      int     `json:"total_tokens"`       // 總計
	
	// 前綴快取分析指標
	CachedTokens     int     `json:"cached_tokens"`      // 理論命中前綴的 Token 數
	NewTokens        int     `json:"new_tokens"`         // 本輪新增 (未快取) 的 Token 數
	CacheHitRate     float64 `json:"cache_hit_rate"`     // 快取命中率 (%)
}

// UnifiedAgentEvent 是 Core 統一標準化事件模型
type UnifiedAgentEvent struct {
	SessionID   string          `json:"session_id"`   // 對話識別碼
	StepIndex   int             `json:"step_index"`   // 步驟序號 (0-indexed)
	Timestamp   time.Time       `json:"timestamp"`    // 產生時間
	Source      string          `json:"source"`       // USER_EXPLICIT, MODEL, SYSTEM
	Type        StepType        `json:"type"`         // USER_INPUT, MODEL_RESPONSE, etc.
	Status      string          `json:"status"`       // DONE, RUNNING, ERROR
	
	// 內容與摘要
	Summary     string          `json:"summary"`      // 一行簡短摘要 (終端機輸出用)
	RawContent  string          `json:"raw_content"`  // 原始內容字串
	Thinking    string          `json:"thinking,omitempty"` // 思考鏈字串
	ToolCalls   []ToolCallInfo  `json:"tool_calls,omitempty"`
	ToolResults []ToolResultInfo`json:"tool_results,omitempty"`
	
	// 5 維度 Token 分佈 (由 Analyzer 計算)
	Tokens      TokenBreakdown  `json:"tokens"`
	CacheStatus string          `json:"cache_status"` // HIT, MISS, WRITE, UNKNOWN
}
