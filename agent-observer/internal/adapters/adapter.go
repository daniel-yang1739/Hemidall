package adapters

import (
	"context"

	"agent-observer/internal/core"
)

// AgentAdapter 定義所有外部 Agent 數據來源必須實作的介面
type AgentAdapter interface {
	// Name 回傳該 Adapter 名稱 (例如: "antigravity", "claude_code", "generic_jsonl")
	Name() string

	// Start 啟動監聽，並將標準化後的 UnifiedAgentEvent 寫入 out channel
	// 當 ctx 被取消時，應優雅結束
	Start(ctx context.Context, out chan<- core.UnifiedAgentEvent) error
}
