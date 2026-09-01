package core

import (
	"fmt"
	"strings"
)

const (
	plainTableWidth = 72
)

// FormatTokenBreakdownTable renders persisted usage observations and local
// estimates as separate panels for the legacy plain-text mode.
func FormatTokenBreakdownTable(event UnifiedAgentEvent) string {
	estimate := event.Tokens

	var builder strings.Builder
	builder.WriteString("┌" + strings.Repeat("─", plainTableWidth) + "┐\n")
	builder.WriteString("│  TRACK 1: PERSISTED CLOUD USAGE (SCHEMA-INFERRED LOCAL METADATA)        │\n")
	builder.WriteString("├──────────────────────────┬─────────────────────────────────────────────┤\n")
	writePersistedUsageRows(&builder, event.Usage)
	builder.WriteString(fmt.Sprintf("│ Response timestamp       │ %-43s │\n", event.Timestamp.Format("2006-01-02 15:04:05")))
	builder.WriteString("└──────────────────────────┴─────────────────────────────────────────────┘\n")

	builder.WriteString("┌────────────────────────────────────────────────────────────────────────┐\n")
	builder.WriteString("│  TRACK 2: LOCAL TRANSCRIPT DIAGNOSTICS (NOT REQUEST ANATOMY)           │\n")
	builder.WriteString("├──────────────────────────┬─────────────────────────────────────────────┤\n")
	builder.WriteString(fmt.Sprintf("│ Current event text       │ %-43d │\n", estimate.StepDelta))
	builder.WriteString(fmt.Sprintf("│ Transcript accumulated   │ %-43d │\n", estimate.RawLocalAccumulated))
	builder.WriteString("│ Persisted snapshot split │ unavailable in plain event output           │\n")
	builder.WriteString("│ Interpretation           │ local cl100k_base diagnostic only           │\n")
	builder.WriteString("└────────────────────────────────────────────────────────────────────────┘\n")

	return builder.String()
}

func writePersistedUsageRows(builder *strings.Builder, usage PersistedUsageObservation) {
	if !usage.Available {
		builder.WriteString("│ Persisted usage         │ unavailable                                 │\n")
		return
	}
	modelName := usage.ModelName
	if modelName == "" {
		modelName = "unknown"
	}
	builder.WriteString(fmt.Sprintf("│ Recorded model          │ %-43s │\n", modelName))
	if usage.HasObservedContextTokens {
		builder.WriteString(fmt.Sprintf("│ Observed context tokens │ %-43d │\n", usage.ObservedContextTokens))
	} else {
		builder.WriteString("│ Observed context tokens │ unavailable                                 │\n")
	}
	if usage.HasContextLimit {
		builder.WriteString(fmt.Sprintf("│ Observed context limit  │ %-43d │\n", usage.ContextLimit))
	} else {
		builder.WriteString("│ Observed context limit  │ unavailable                                 │\n")
	}
	if usage.HasUncachedInputTokens {
		builder.WriteString(fmt.Sprintf("│ Uncached input tokens   │ %-43d │\n", usage.UncachedInputTokens))
	} else {
		builder.WriteString("│ Uncached input tokens   │ unavailable                                 │\n")
	}
	if usage.HasCachedInputTokens {
		builder.WriteString(fmt.Sprintf("│ Cached input tokens     │ %-43d │\n", usage.CachedInputTokens))
	} else if usage.HasUncachedInputTokens {
		builder.WriteString("│ Cached input tokens     │ 0 (inferred from omitted scalar)             │\n")
	} else {
		builder.WriteString("│ Cached input tokens     │ unavailable                                 │\n")
	}
}
