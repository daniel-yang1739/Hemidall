package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderGlossaryModal() string {
	modalWidth := m.width - 6
	if modalWidth > 96 {
		modalWidth = 96
	}
	if modalWidth < 56 {
		modalWidth = 56
	}
	modalInnerWidth := modalWidth - 4

	modalHeight := m.height - 4
	if modalHeight > 30 {
		modalHeight = 30
	}
	if modalHeight < 16 {
		modalHeight = 16
	}

	var contentLines []string

	// 1. Title
	titleText := "TERMINOLOGY & ARCHITECTURE GLOSSARY [h]"
	contentLines = append(contentLines, TitleStyle.Render(truncateVisualWidth(titleText, modalInnerWidth)))
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", modalInnerWidth)))

	// Section 1: Core Metrics & Token Accounting
	contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [Core Metrics & Token Accounting]"))
	
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("Total Active Context"),
		"The actual token payload sent to the LLM in the current HTTP turn."))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("Prefix Cache Hit"),
		"Tokens reused from GPU HBM memory (100% free / 50-80% cost discount)."))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("New Billable Tokens"),
		"Uncached tokens in the current turn that require full prefill compute."))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorWarning).Render("Raw Log Accumulated"),
		"Total uncompressed tokens in local append-only history (e.g. 400k),"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s   %s", "",
		"before cloud sliding-window compaction or tail truncation."))
	contentLines = append(contentLines, "")

	// Section 2: 5 Dimensions of Context Anatomy (Track 2)
	contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [5 Dimensions of Context Anatomy (Track 2)]"))
	
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("1. System Instruction"),
		"Base system prompts, developer rules, and invariant guidelines."))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("2. MCP Tools Schema"),
		"Function calling JSON schemas defining available tools and arguments."))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("3. Tool Results / Diff"),
		"Outputs returned from tool executions (command output, file reads)."))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("4. Conversation Hist"),
		"Past user-assistant dialogue turns retained in the active window."))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorWarning).Render("5. Active Turn / CoT"),
		"Current turn prompt + Model thinking (Chain of Thought) + Tool call."))
	contentLines = append(contentLines, "")

	// Section 3: Mechanism Terms
	if modalHeight >= 24 {
		contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [Context Mechanics & Physics]"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("TTL Cold Start"),
			"GPU memory evicts idle KV cache after ~5 min; next turn incurs cold miss."))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("Context Compaction"),
			"Async summarization triggered at 95% watermark to compact history to 48%."))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("Reverse Sliding Window"),
			"Algorithm allocating tokens backwards from newest step to match cloud budget."))
		contentLines = append(contentLines, "")
	}

	// Pad or clamp
	for len(contentLines) < modalHeight-2 {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > modalHeight-2 {
		contentLines = contentLines[:modalHeight-2]
	}

	// Footer
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", modalInnerWidth)))
	footerHint := fmt.Sprintf(" Press %s or %s to close this glossary panel", KeyStyle.Render("[h]"), KeyStyle.Render("[Esc]"))
	contentLines = append(contentLines, lipgloss.NewStyle().MaxWidth(modalInnerWidth).Render(footerHint))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Background(ColorDarkBg).
		Padding(0, 1).
		Width(modalInnerWidth).
		Render(strings.Join(contentLines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
}
