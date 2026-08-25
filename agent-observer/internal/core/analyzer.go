package core

import (
	"strings"
	"sync"
)

// SessionContextState 維護單一 Session 的上下文累積狀態
type SessionContextState struct {
	SessionID         string
	SystemTokens      int
	ToolsDefTokens    int
	ToolResultTokens  int
	HistoryTokens     int
	TotalTokens       int
	HasInitialized    bool  // 是否已完成初始快取寫入
	PrevTotalTokens   int   // 上一輪的總 Context Token 數
}

// PayloadAnalyzer 是 Core 核心 Context 分析器
type PayloadAnalyzer struct {
	mu       sync.Mutex
	sessions map[string]*SessionContextState
}

// NewPayloadAnalyzer 建立分析器實例
func NewPayloadAnalyzer() *PayloadAnalyzer {
	return &PayloadAnalyzer{
		sessions: make(map[string]*SessionContextState),
	}
}

// AnalyzeStep 分析單一事件並注入 Token 拆解與快取指標
func (a *PayloadAnalyzer) AnalyzeStep(event *UnifiedAgentEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	state, exists := a.sessions[event.SessionID]
	if !exists {
		state = &SessionContextState{
			SessionID:      event.SessionID,
			SystemTokens:   4618, // 基準靜態 System Prompt (包含 identity, rules, skills 宣告)
			ToolsDefTokens: 3200, // 基準 MCP Tools Schema 總宣告
			HasInitialized: false,
		}
		a.sessions[event.SessionID] = state
	}

	// 1. 計算該 Step 的各維度 Token
	contentTokens := CountTokens(event.RawContent)
	thinkingTokens := CountTokens(event.Thinking)

	// 計算工具呼叫參數 Tokens
	toolCallArgsTokens := 0
	for _, tc := range event.ToolCalls {
		if tc.RawArgs != "" {
			toolCallArgsTokens += CountTokens(tc.RawArgs)
		} else if len(tc.Arguments) > 0 {
			toolCallArgsTokens += 50
		}
	}

	// 2. 根據 StepType 歸類累加
	switch event.Type {
	case StepTypeUserInput:
		event.Tokens.ActiveTurnTokens = contentTokens
	case StepTypeModelResponse:
		event.Tokens.ActiveTurnTokens = contentTokens
		event.Tokens.ThinkingTokens = thinkingTokens
	case StepTypeToolCall:
		event.Tokens.ActiveTurnTokens = toolCallArgsTokens
		event.Tokens.ThinkingTokens = thinkingTokens
	case StepTypeRunCommand, StepTypeViewFile, StepTypeCodeAction, StepTypeListDirectory, StepTypeToolResult:
		state.ToolResultTokens += contentTokens
		event.Tokens.ToolResultTokens = contentTokens
	default:
		if strings.Contains(string(event.Type), "SYSTEM") {
			state.SystemTokens += contentTokens
		} else if contentTokens > 0 {
			state.HistoryTokens += contentTokens
		}
	}

	// 3. 計算當前整個 Context 總量
	currentDelta := event.Tokens.ActiveTurnTokens + event.Tokens.ThinkingTokens
	totalContext := state.SystemTokens + state.ToolsDefTokens + state.ToolResultTokens + state.HistoryTokens + currentDelta

	event.Tokens.SystemTokens = state.SystemTokens
	event.Tokens.ToolsDefTokens = state.ToolsDefTokens
	event.Tokens.ToolResultTokens = state.ToolResultTokens
	event.Tokens.HistoryTokens = state.HistoryTokens
	event.Tokens.TotalTokens = totalContext

	// 4. 前綴快取 (Prefix Caching) 物理命中計算
	if !state.HasInitialized {
		// 第一次寫入 (Cache Write)
		event.Tokens.CachedTokens = 0
		event.Tokens.NewTokens = totalContext
		event.Tokens.CacheHitRate = 0.0
		event.CacheStatus = "WRITE"
		state.HasInitialized = true
	} else {
		// 前綴快取重用：基底 (System + Tools) + 過去所有已固化歷史均為可重用前綴
		cachedTokens := state.PrevTotalTokens
		if cachedTokens > totalContext {
			cachedTokens = totalContext
		}

		newTokens := totalContext - cachedTokens
		if newTokens < 0 {
			newTokens = 0
		}

		hitRate := 0.0
		if totalContext > 0 {
			hitRate = float64(cachedTokens) / float64(totalContext) * 100.0
		}

		event.Tokens.CachedTokens = cachedTokens
		event.Tokens.NewTokens = newTokens
		event.Tokens.CacheHitRate = hitRate

		if hitRate >= 80.0 {
			event.CacheStatus = "HIT"
		} else if hitRate > 0.0 {
			event.CacheStatus = "PARTIAL"
		} else {
			event.CacheStatus = "MISS"
		}
	}

	// 5. 更新狀態
	state.PrevTotalTokens = totalContext
	if event.Type == StepTypeUserInput || event.Type == StepTypeModelResponse {
		state.HistoryTokens += contentTokens
	}
}
