package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DocItem represents a single help or glossary entry
type DocItem struct {
	Category string
	Key      string
	Desc     string
}

var allDocItems = []DocItem{
	// 1. Global Navigation
	{Category: "Global Navigation", Key: "1 / d", Desc: "Switch to Live Dashboard View"},
	{Category: "Global Navigation", Key: "2 / s", Desc: "Switch to Step History Explorer View"},
	{Category: "Global Navigation", Key: "3 / ? / h", Desc: "Switch to this Help & Documentation View"},
	{Category: "Global Navigation", Key: "Ctrl+p", Desc: "Open Session Quick Switcher Modal"},
	{Category: "Global Navigation", Key: "/", Desc: "Focus search bar to filter help & glossary topics"},
	{Category: "Global Navigation", Key: "q / Ctrl+c", Desc: "Gracefully quit agent-observer"},

	// 2. Dashboard View
	{Category: "Dashboard Controls", Key: "j / k / ↑ / ↓", Desc: "Time machine step-by-step playback (Prev / Next)"},
	{Category: "Dashboard Controls", Key: "Ctrl+d / Ctrl+u", Desc: "Fast jump 10 steps forward / backward"},
	{Category: "Dashboard Controls", Key: "g / G", Desc: "Jump to first step / Jump to latest (LIVE)"},
	{Category: "Dashboard Controls", Key: "Enter", Desc: "Inspect selected playback step in History View"},

	// 3. History Explorer View
	{Category: "History Explorer", Key: "Tab / Enter / l", Desc: "Switch focus to Right Pane (Inspector)"},
	{Category: "History Explorer", Key: "Esc / Left", Desc: "Return focus to Left Pane (Steps List)"},
	{Category: "History Explorer", Key: "j / k / ↑ / ↓", Desc: "Select step (Left) / Scroll Inspector (Right)"},
	{Category: "History Explorer", Key: "Ctrl+d / Ctrl+u", Desc: "Scroll Inspector by 10 lines"},
	{Category: "History Explorer", Key: "Ctrl+f / Ctrl+b", Desc: "Page Down / Page Up in Inspector buffer"},
	{Category: "History Explorer", Key: "v / V", Desc: "Enter Visual selection mode in Inspector"},
	{Category: "History Explorer", Key: "y", Desc: "Yank (copy) selected lines to system clipboard"},

	// 4. Session Switcher
	{Category: "Session Switcher", Key: "Type text", Desc: "Real-time fuzzy filter sessions by ID or path"},
	{Category: "Session Switcher", Key: "Ctrl+j/k / ↑/↓", Desc: "Navigate session selection (Vim-first)"},
	{Category: "Session Switcher", Key: "Tab / Shift+Tab", Desc: "Navigate session selection forward / backward"},
	{Category: "Session Switcher", Key: "Enter", Desc: "Attach and dynamically load session history"},
	{Category: "Session Switcher", Key: "Esc / Ctrl+p", Desc: "Close switcher modal and return to previous view"},

	// 5. 5 Dimensions of Context Anatomy
	{Category: "5 Dimensions of Context", Key: "1. System Instruction", Desc: "Base system prompt, developer rules, and invariant instructions."},
	{Category: "5 Dimensions of Context", Key: "2. MCP Tools Schema", Desc: "Function calling JSON schemas defining available tools and arguments."},
	{Category: "5 Dimensions of Context", Key: "3. Tool Results / Diff", Desc: "Outputs returned from tool executions (bash stdout, file reads, diffs)."},
	{Category: "5 Dimensions of Context", Key: "4. Conversation Hist", Desc: "Past user-assistant dialogue turns retained in the active context window."},
	{Category: "5 Dimensions of Context", Key: "5. Active Turn / CoT", Desc: "Current turn user prompt + Model thinking (CoT) + Tool call."},

	// 6. Architecture & Token Concepts
	{Category: "Architecture Concepts", Key: "Total Active Context", Desc: "The actual token payload sent to the LLM in the current HTTP turn."},
	{Category: "Architecture Concepts", Key: "Prefix Cache Hit", Desc: "KV cache tokens reused from GPU HBM memory ($0.00 / 50-80% cost discount)."},
	{Category: "Architecture Concepts", Key: "New Billable Tokens", Desc: "Uncached tokens in turn requiring full prefill computation."},
	{Category: "Architecture Concepts", Key: "Raw Log Accumulated", Desc: "Total uncompressed tokens in local append-only history (e.g. 400k), before cloud sliding-window compaction or tail truncation."},
	{Category: "Architecture Concepts", Key: "Active Turn / CoT", Desc: "Tokens generated in current step including thinking (Chain of Thought) & tool calls."},
	{Category: "Architecture Concepts", Key: "TTL Cold Start", Desc: "GPU memory evicts idle KV cache after ~5 min; next turn incurs full prefill cost."},
	{Category: "Architecture Concepts", Key: "Context Compaction", Desc: "Async summarization triggered at 95% watermark to compact history to 48%."},
	{Category: "Architecture Concepts", Key: "Reverse Sliding Window", Desc: "Algorithm allocating tokens backwards from newest step to match cloud budget."},
}

func (m Model) renderHelpView() string {
	boxInnerWidth := m.width - 4
	if boxInnerWidth < 40 {
		boxInnerWidth = 40
	}

	innerRowsLimit := m.height - 4
	if innerRowsLimit < 4 {
		innerRowsLimit = 4
	}

	var rawLines []string

	// 1. Search / Filter Header
	searchStatus := "[Press / to search, Esc to clear]"
	if m.isHelpSearching {
		searchStatus = "[SEARCHING: Type to filter, Enter/Esc to finish]"
	}
	cursorChar := ""
	if m.isHelpSearching {
		cursorChar = "█"
	}
	searchBar := fmt.Sprintf("Search: [%s%s]  %s",
		m.helpSearchQuery,
		lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar),
		lipgloss.NewStyle().Foreground(ColorMuted).Render(searchStatus))

	rawLines = append(rawLines, TitleStyle.Render(truncateVisualWidth("HELP & ARCHITECTURE GLOSSARY", boxInnerWidth-2)))
	rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(searchBar, boxInnerWidth-2)))
	rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", boxInnerWidth-2)))

	// 2. Filter & Render Documentation Items
	query := strings.ToLower(strings.TrimSpace(m.helpSearchQuery))
	currentCategory := ""
	matchedCount := 0

	for _, item := range allDocItems {
		if query != "" {
			if !strings.Contains(strings.ToLower(item.Category), query) &&
				!strings.Contains(strings.ToLower(item.Key), query) &&
				!strings.Contains(strings.ToLower(item.Desc), query) {
				continue
			}
		}

		matchedCount++
		if item.Category != currentCategory {
			if currentCategory != "" {
				rawLines = append(rawLines, "")
			}
			currentCategory = item.Category
			rawLines = append(rawLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" ["+item.Category+"]"))
		}

		// Pre-pad plain key before styling to prevent ANSI width calculation corruption
		paddedKey := fmt.Sprintf("%-24s", item.Key)
		keyRendered := KeyStyle.Render(paddedKey)
		sepRendered := lipgloss.NewStyle().Foreground(ColorBorder).Render(" : ")
		descRendered := lipgloss.NewStyle().Foreground(ColorLightText).Render(item.Desc)

		entryLine := "  " + keyRendered + sepRendered + descRendered
		rawLines = append(rawLines, wrapVisualLines(entryLine, boxInnerWidth-4)...)
	}

	if matchedCount == 0 {
		rawLines = append(rawLines, "")
		rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorWarning).Render("  No matching topics found for '"+m.helpSearchQuery+"'."))
		rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  Press [Esc] to clear search query."))
	}

	// 3. Virtual Viewport Slicing
	totalLines := len(rawLines)
	availableLines := innerRowsLimit

	maxScroll := totalLines - availableLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	currentScroll := m.helpScroll
	if currentScroll > maxScroll {
		currentScroll = maxScroll
	}

	endLine := currentScroll + availableLines
	if endLine > totalLines {
		endLine = totalLines
	}

	var visibleLines []string
	for i := currentScroll; i < endLine; i++ {
		visibleLines = append(visibleLines, truncateVisualWidth(rawLines[i], boxInnerWidth-2))
	}

	for len(visibleLines) < innerRowsLimit {
		visibleLines = append(visibleLines, "")
	}
	if len(visibleLines) > innerRowsLimit {
		visibleLines = visibleLines[:innerRowsLimit]
	}

	return ActivePanelStyle.Width(boxInnerWidth).Render(strings.Join(visibleLines, "\n"))
}
