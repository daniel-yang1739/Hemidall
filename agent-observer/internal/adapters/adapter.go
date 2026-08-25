package adapters

import (
	"context"

	"agent-observer/internal/core"
)

// AgentAdapter defines the interface for all external agent data source ingestion
type AgentAdapter interface {
	// Name returns the identifier of the adapter (e.g., "antigravity", "claude_code", "generic_jsonl")
	Name() string

	// Start begins streaming UnifiedAgentEvents to the output channel.
	// It must terminate gracefully when ctx is canceled.
	Start(ctx context.Context, out chan<- core.UnifiedAgentEvent) error
}
