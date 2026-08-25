package core

import (
	"fmt"
	"strings"
)

// FormatTokenBreakdownTable formats the 5-dimension token distribution into an ASCII table
func FormatTokenBreakdownTable(e UnifiedAgentEvent) string {
	t := e.Tokens
	total := t.TotalTokens
	if total == 0 {
		total = 1
	}

	sysPct := float64(t.SystemTokens) / float64(total) * 100.0
	toolsPct := float64(t.ToolsDefTokens) / float64(total) * 100.0
	resPct := float64(t.ToolResultTokens) / float64(total) * 100.0
	histPct := float64(t.HistoryTokens) / float64(total) * 100.0
	activePct := float64(t.ActiveTurnTokens+t.ThinkingTokens) / float64(total) * 100.0

	var cacheBadge string
	switch e.CacheStatus {
	case "HIT":
		cacheBadge = fmt.Sprintf("🟢 [CACHE HIT %.1f%%]", t.CacheHitRate)
	case "PARTIAL":
		cacheBadge = fmt.Sprintf("🟡 [PARTIAL HIT %.1f%%]", t.CacheHitRate)
	case "WRITE":
		cacheBadge = "🔵 [CACHE WRITE / INITIAL]"
	default:
		cacheBadge = "🔴 [CACHE MISS / BROKEN]"
	}

	var sb strings.Builder
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│  📊 Context Token Breakdown (Total: %-7d Tokens) %-21s│\n", t.TotalTokens, cacheBadge))
	sb.WriteString("├───────────────────────┬──────────────┬─────────────┬───────────────────┤\n")
	sb.WriteString("│ Context Dimension     │ Tokens       │ Share (%)   │ Visual Composition│\n")
	sb.WriteString("├───────────────────────┼──────────────┼─────────────┼───────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│ 1. System Instruction │ %-12d │ %5.1f%%      │ %-17s │\n", t.SystemTokens, sysPct, renderProgressBar(sysPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 2. MCP Tools Schema   │ %-12d │ %5.1f%%      │ %-17s │\n", t.ToolsDefTokens, toolsPct, renderProgressBar(toolsPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 3. Tool Results / Diff│ %-12d │ %5.1f%%      │ %-17s │\n", t.ToolResultTokens, resPct, renderProgressBar(resPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 4. Conversation Hist  │ %-12d │ %5.1f%%      │ %-17s │\n", t.HistoryTokens, histPct, renderProgressBar(histPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 5. Active Turn / CoT  │ %-12d │ %5.1f%%      │ %-17s │\n", t.ActiveTurnTokens+t.ThinkingTokens, activePct, renderProgressBar(activePct, 15)))
	sb.WriteString("├───────────────────────┴──────────────┴─────────────┴───────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│  ⚡ Prefix Cache Reuse : %-6d Tokens (%5.1f%%)                             │\n", t.CachedTokens, t.CacheHitRate))
	sb.WriteString(fmt.Sprintf("│  🔥 New Uncached (Cost): %-6d Tokens (%5.1f%%)                             │\n", t.NewTokens, 100.0-t.CacheHitRate))
	sb.WriteString("└────────────────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

func renderProgressBar(pct float64, totalBlocks int) string {
	filled := int((pct / 100.0) * float64(totalBlocks))
	if filled > totalBlocks {
		filled = totalBlocks
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", totalBlocks-filled)
	return bar
}
