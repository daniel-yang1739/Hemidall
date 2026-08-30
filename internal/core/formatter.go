package core

import (
	"fmt"
	"strings"
)

// FormatTokenBreakdownTable formats the dual-track telemetry into two clear, distinct panels
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
	case "EXPIRED":
		cacheBadge = "🔴 [TTL EXPIRED / COLD START]"
	default:
		cacheBadge = "🔴 [CACHE MISS / BROKEN]"
	}

	ctxLimit := t.OfficialContextLimit
	if ctxLimit == 0 {
		ctxLimit = 256000
	}
	ctxUsagePct := float64(total) / float64(ctxLimit) * 100.0

	var sb strings.Builder

	// ==================== PANEL 1: OFFICIAL TELEMETRY ====================
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────┐\n")
	if t.IsOfficialData {
		sb.WriteString(fmt.Sprintf("│  👑 TRACK 1: Official Gemini Physics Telemetry (Billing Ground Truth)  │\n"))
		sb.WriteString("├──────────────────────────┬─────────────────────────────────────────────┤\n")
		modelName := t.OfficialModel
		if modelName == "" {
			modelName = "gemini-3.7-flash-high"
		}
		sb.WriteString(fmt.Sprintf("│ 🎯 Backend Model         │ %-43s │\n", modelName))
		sb.WriteString(fmt.Sprintf("│ 📊 Total Active Context  │ %-7d Tokens (%5.1f%% of %dk Window)      │\n", total, ctxUsagePct, ctxLimit/1000))
		sb.WriteString(fmt.Sprintf("│    Context Progress Bar  │ [%-25s]           │\n", renderProgressBar(ctxUsagePct, 25)))
		sb.WriteString(fmt.Sprintf("│ ⚡ Prefix Cache Hit      │ %-7d Tokens (%5.1f%%) %-21s│\n", t.CachedTokens, t.CacheHitRate, cacheBadge))
		sb.WriteString(fmt.Sprintf("│ 🔥 New Billable Tokens   │ %-7d Tokens (%5.1f%%)                           │\n", t.NewTokens, 100.0-t.CacheHitRate))
		sb.WriteString(fmt.Sprintf("│ ⏱️ Response Timestamp    │ %-43s │\n", e.Timestamp.Format("2006-01-02 15:04:05")))
	} else {
		sb.WriteString(fmt.Sprintf("│  🔍 TRACK 1: Local BPE Estimation (Awaiting Google API Response...)    │\n"))
		sb.WriteString("├──────────────────────────┬─────────────────────────────────────────────┤\n")
		sb.WriteString(fmt.Sprintf("│ 📊 Estimated Context     │ %-7d Tokens %-32s│\n", total, cacheBadge))
		sb.WriteString(fmt.Sprintf("│ ⚡ Estimated Cache Hit   │ %-7d Tokens (%5.1f%%)                           │\n", t.CachedTokens, t.CacheHitRate))
		sb.WriteString(fmt.Sprintf("│ 🔥 Estimated New Tokens  │ %-7d Tokens (%5.1f%%)                           │\n", t.NewTokens, 100.0-t.CacheHitRate))
		sb.WriteString(fmt.Sprintf("│ ⏱️ Response Timestamp    │ %-43s │\n", e.Timestamp.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString("└──────────────────────────┴─────────────────────────────────────────────┘\n")

	// ==================== PANEL 2: 5-DIMENSION CONTEXT ANATOMY ====================
	sb.WriteString("┌────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  🔬 TRACK 2: Local 5-Dimension Context Anatomy (Payload Analysis)      │\n")
	sb.WriteString("├───────────────────────┬──────────────┬─────────────┬───────────────────┤\n")
	sb.WriteString("│ Context Dimension     │ Tokens       │ Share (%)   │ Visual Breakdown  │\n")
	sb.WriteString("├───────────────────────┼──────────────┼─────────────┼───────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│ 1. System Instruction │ %-12d │ %5.1f%%      │ %-17s │\n", t.SystemTokens, sysPct, renderProgressBar(sysPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 2. MCP Tools Schema   │ %-12d │ %5.1f%%      │ %-17s │\n", t.ToolsDefTokens, toolsPct, renderProgressBar(toolsPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 3. Tool Results / Diff│ %-12d │ %5.1f%%      │ %-17s │\n", t.ToolResultTokens, resPct, renderProgressBar(resPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 4. Conversation Hist  │ %-12d │ %5.1f%%      │ %-17s │\n", t.HistoryTokens, histPct, renderProgressBar(histPct, 15)))
	sb.WriteString(fmt.Sprintf("│ 5. Active Turn / CoT  │ %-12d │ %5.1f%%      │ %-17s │\n", t.ActiveTurnTokens+t.ThinkingTokens, activePct, renderProgressBar(activePct, 15)))
	sb.WriteString("├───────────────────────┴──────────────┴─────────────┴───────────────────┤\n")
	if t.RawLocalAccumulated > total {
		truncatedTokens := t.RawLocalAccumulated - total
		sb.WriteString(fmt.Sprintf("│  💡 Raw Uncompressed Log: %-7d Tokens (✂️ %d Tokens Truncated)     │\n", t.RawLocalAccumulated, truncatedTokens))
	} else {
		sb.WriteString("│  💡 Estimated dimensions calibrated to the observed total token count │\n")
	}
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
