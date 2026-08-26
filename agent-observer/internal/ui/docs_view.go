package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DocDefinitionItem represents an architecture, dimension, or token concept definition
type DocDefinitionItem struct {
	Category string
	Key      string
	Desc     string
}

var allDocDefinitions = []DocDefinitionItem{
	// Section 1: 5 Dimensions of Context Anatomy (Track 2)
	{
		Category: "5 Dimensions of Context Anatomy (Track 2)",
		Key:      "1. System Instruction",
		Desc:     "Base system prompt, developer rules, invariant guidelines, and safety constraints that guide agent behavior.",
	},
	{
		Category: "5 Dimensions of Context Anatomy (Track 2)",
		Key:      "2. MCP Tools Schema",
		Desc:     "Function calling JSON schemas defining all available tools, parameter types, descriptions, and required fields.",
	},
	{
		Category: "5 Dimensions of Context Anatomy (Track 2)",
		Key:      "3. Tool Results / Diff",
		Desc:     "Payloads returned from tool executions (bash stdout/stderr, file reads, directory listings, code diffs).",
	},
	{
		Category: "5 Dimensions of Context Anatomy (Track 2)",
		Key:      "4. Conversation Hist",
		Desc:     "Past user-assistant dialogue turns and prior tool call summaries retained within the active context window.",
	},
	{
		Category: "5 Dimensions of Context Anatomy (Track 2)",
		Key:      "5. Active Turn / CoT",
		Desc:     "The latest turn user prompt + Model thinking (Chain of Thought / CoT reasoning) + Active tool call payload.",
	},

	// Section 2: Core Metrics & Token Accounting (Track 1)
	{
		Category: "Core Metrics & Ground Truth Billing (Track 1)",
		Key:      "Total Active Context",
		Desc:     "The exact mathematical token count sent to the LLM in the current HTTP request window (e.g. 130k / 256k limit).",
	},
	{
		Category: "Core Metrics & Ground Truth Billing (Track 1)",
		Key:      "Prefix Cache Hit",
		Desc:     "Identical prefix tokens reused directly from GPU High-Bandwidth Memory (HBM) ($0.00 / 50-80% cost discount).",
	},
	{
		Category: "Core Metrics & Ground Truth Billing (Track 1)",
		Key:      "New Billable Tokens",
		Desc:     "Uncached tokens in the current turn (new prompt + new tool results) requiring full GPU prefill matrix compute.",
	},
	{
		Category: "Core Metrics & Ground Truth Billing (Track 1)",
		Key:      "Raw Log Accumulated",
		Desc:     "Total uncompressed tokens in local append-only history files (e.g. 400k), before cloud sliding-window compaction or truncation.",
	},
	{
		Category: "Core Metrics & Ground Truth Billing (Track 1)",
		Key:      "TTL Cold Start",
		Desc:     "Google GPU memory evicts idle KV cache after ~5 min; subsequent turns incur full cold-start prefill billing.",
	},

	// Section 3: Context Mechanics & Physics
	{
		Category: "Context Mechanics & Memory Physics",
		Key:      "Context Compaction",
		Desc:     "Asynchronous summarization triggered at 95% High Watermark (~245k tokens) to compact history down to 48% Low Watermark (~120k).",
	},
	{
		Category: "Context Mechanics & Memory Physics",
		Key:      "Reverse Sliding Window",
		Desc:     "Algorithm allocating tokens backwards from the newest step to match Google official active budget, pruning ancient steps.",
	},
	{
		Category: "Context Mechanics & Memory Physics",
		Key:      "Longest Common Prefix",
		Desc:     "LCP algorithm comparing sequential steps to determine exact byte-level shared prefix for KV cache reuse.",
	},
}

func (m Model) renderDocsView() string {
	boxInnerWidth := m.width - 2
	if boxInnerWidth < 40 {
		boxInnerWidth = 40
	}
	contentWidth := boxInnerWidth - 2 // Account for Padding(0, 1)

	innerRowsLimit := m.height - 4
	if innerRowsLimit < 4 {
		innerRowsLimit = 4
	}

	var rawLines []string

	// 1. Search / Filter Header
	searchStatus := "[Press / to search, Esc to clear]"
	if m.isDocsSearching {
		searchStatus = "[SEARCHING: Type to filter, Enter/Esc to finish]"
	}
	cursorChar := ""
	if m.isDocsSearching {
		cursorChar = "█"
	}
	searchBar := fmt.Sprintf("Filter: [%s%s]  %s",
		m.docsSearchQuery,
		lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar),
		lipgloss.NewStyle().Foreground(ColorMuted).Render(searchStatus))

	rawLines = append(rawLines, TitleStyle.Render(truncateVisualWidth("ARCHITECTURE & CONTEXT DEFINITIONS", contentWidth)))
	rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(searchBar, contentWidth)))
	rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", contentWidth)))

	// 2. Filter & Render Definition Items
	query := strings.ToLower(strings.TrimSpace(m.docsSearchQuery))
	currentCategory := ""
	matchedCount := 0

	for _, item := range allDocDefinitions {
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

		// Pre-pad key before styling to ensure exact width
		paddedKey := fmt.Sprintf("%-24s", item.Key)
		keyRendered := KeyStyle.Render(paddedKey)
		sepRendered := lipgloss.NewStyle().Foreground(ColorBorder).Render(" : ")
		descRendered := lipgloss.NewStyle().Foreground(ColorLightText).Render(item.Desc)

		entryLine := "  " + keyRendered + sepRendered + descRendered
		rawLines = append(rawLines, wrapVisualLines(entryLine, contentWidth-2)...)
	}

	if matchedCount == 0 {
		rawLines = append(rawLines, "")
		rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorWarning).Render("  No matching architecture definitions found for '"+m.docsSearchQuery+"'."))
		rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  Press [Esc] to clear search query."))
	}

	// 3. Virtual Viewport Slicing
	totalLines := len(rawLines)
	availableLines := innerRowsLimit

	maxScroll := totalLines - availableLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	currentScroll := m.docsScroll
	if currentScroll > maxScroll {
		currentScroll = maxScroll
	}

	endLine := currentScroll + availableLines
	if endLine > totalLines {
		endLine = totalLines
	}

	var visibleLines []string
	for i := currentScroll; i < endLine; i++ {
		visibleLines = append(visibleLines, truncateVisualWidth(rawLines[i], contentWidth))
	}

	for len(visibleLines) < innerRowsLimit {
		visibleLines = append(visibleLines, "")
	}
	if len(visibleLines) > innerRowsLimit {
		visibleLines = visibleLines[:innerRowsLimit]
	}

	return ActivePanelStyle.Width(boxInnerWidth).Render(strings.Join(visibleLines, "\n"))
}
