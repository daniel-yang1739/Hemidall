package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ShortcutItem represents a single keyboard shortcut entry
type ShortcutItem struct {
	Category string
	Key      string
	Desc     string
}

var allShortcutItems = []ShortcutItem{
	// 1. Global Navigation
	{Category: "Global Navigation", Key: "Tab / Shift+Tab", Desc: "Cyclic cycle views (Dashboard ➔ History ➔ Context ➔ Docs)"},
	{Category: "Global Navigation", Key: "1 / d", Desc: "Switch to [1] Dashboard View"},
	{Category: "Global Navigation", Key: "2 / h", Desc: "Switch to [2] History Explorer View"},
	{Category: "Global Navigation", Key: "3 / c", Desc: "Switch to [3] Context View"},
	{Category: "Global Navigation", Key: "4 / i", Desc: "Switch to [4] Architecture Docs View"},
	{Category: "Global Navigation", Key: "Ctrl+p", Desc: "Open Session Quick Switcher Modal"},
	{Category: "Global Navigation", Key: "? / F1", Desc: "Toggle this Keyboard Shortcuts Float Panel"},
	{Category: "Global Navigation", Key: "q / Ctrl+c", Desc: "Gracefully quit Heimdall"},

	// 2. Context View Controls
	{Category: "Context View", Key: "r / R", Desc: "Toggle Decoded Evidence JSON / Refined Cards Mode"},
	{Category: "Context View", Key: "j / k / ↑ / ↓", Desc: "Select subcategory (Tree) / Scroll (Inspector)"},
	{Category: "Context View", Key: "l / Enter / Right", Desc: "Switch focus to Right Inspector pane"},
	{Category: "Context View", Key: "h / Esc / Left", Desc: "Return focus to Left Context Tree"},
	{Category: "Context View", Key: "m / M", Desc: "Toggle Target Agent (MAIN / SUBAGENT)"},
	{Category: "Context View", Key: "Ctrl+d / Ctrl+u", Desc: "Scroll Inspector by 10 lines"},

	// 3. Dashboard View Controls
	{Category: "Dashboard Playback", Key: "j / k / ↑ / ↓", Desc: "Time machine playback (Previous / Next step)"},
	{Category: "Dashboard Playback", Key: "Ctrl+d / Ctrl+u", Desc: "Fast jump 10 steps forward / backward"},
	{Category: "Dashboard Playback", Key: "g / G", Desc: "Jump to first step / Jump to latest (LIVE)"},
	{Category: "Dashboard Playback", Key: "Enter", Desc: "Inspect selected playback step in History View"},

	// 3. Step History Explorer Controls
	{Category: "History Explorer", Key: "t / T", Desc: "Cycle Step Type filter (All ➔ Tool ➔ Model ➔ User ➔ Code ➔ Generic)"},
	{Category: "History Explorer", Key: "c / C", Desc: "Cycle Cache Status filter (All ➔ Hit ➔ Partial ➔ Write ➔ Expired ➔ Miss)"},
	{Category: "History Explorer", Key: "/", Desc: "Search / filter by Step Number (#)"},
	{Category: "History Explorer", Key: "l / Enter / Right", Desc: "Switch focus to Right Pane (Inspector)"},
	{Category: "History Explorer", Key: "h / Esc / Left", Desc: "Return focus to Left Pane / Clear active filters"},
	{Category: "History Explorer", Key: "j / k / ↑ / ↓", Desc: "Select step (Left) / Scroll Inspector (Right)"},
	{Category: "History Explorer", Key: "Ctrl+d / Ctrl+u", Desc: "Scroll Inspector by 10 lines"},
	{Category: "History Explorer", Key: "Ctrl+f / Ctrl+b", Desc: "Page Down / Page Up in Inspector buffer"},
	{Category: "History Explorer", Key: "v / V", Desc: "Enter Visual selection mode in Inspector"},
	{Category: "History Explorer", Key: "y", Desc: "Yank (copy) selected lines to system clipboard"},

	// 4. Docs View Controls
	{Category: "Docs View", Key: "/", Desc: "Activate real-time keyword search filter"},
	{Category: "Docs View", Key: "t / T", Desc: "Toggle language (Traditional Chinese / English)"},
	{Category: "Docs View", Key: "j / k / ↑ / ↓", Desc: "Scroll documentation topics"},
	{Category: "Docs View", Key: "Ctrl+d / Ctrl+u", Desc: "Scroll documentation by 10 lines"},
	{Category: "Docs View", Key: "Esc", Desc: "Clear search query or return to Dashboard"},

	// 5. Session Switcher Modal
	{Category: "Session Switcher", Key: "Type text", Desc: "Filter by directory, hash, or prompt keywords"},
	{Category: "Session Switcher", Key: "Ctrl+j/k / ↑/↓", Desc: "Navigate session selection (Vim-first)"},
	{Category: "Session Switcher", Key: "Enter", Desc: "Attach and dynamically load session history"},
	{Category: "Session Switcher", Key: "Esc / Ctrl+p", Desc: "Close switcher modal and return"},
}

func (m Model) renderShortcutsModal() string {
	modalWidth := m.width - 8
	if modalWidth > 86 {
		modalWidth = 86
	}
	if modalWidth < 48 {
		modalWidth = 48
	}
	modalInnerWidth := modalWidth - 4
	contentWidth := modalInnerWidth - 2 // Account for Padding(0, 1)

	modalHeight := m.height - 4
	if modalHeight > 24 {
		modalHeight = 24
	}
	if modalHeight < 12 {
		modalHeight = 12
	}

	var contentLines []string

	// 1. Title
	titleText := "KEYBOARD SHORTCUTS [?]"
	contentLines = append(contentLines, TitleStyle.Render(truncateVisualWidth(titleText, contentWidth)))
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", contentWidth)))

	// 2. Body Items grouped by category
	currentCat := ""
	for _, item := range allShortcutItems {
		if item.Category != currentCat {
			if currentCat != "" {
				contentLines = append(contentLines, "")
			}
			currentCat = item.Category
			contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" ["+item.Category+"]"))
		}

		paddedKey := fmt.Sprintf("%-20s", item.Key)
		line := fmt.Sprintf("  %s : %s",
			KeyStyle.Render(paddedKey),
			lipgloss.NewStyle().Foreground(ColorLightText).Render(item.Desc))
		contentLines = append(contentLines, truncateVisualWidth(line, contentWidth))
	}

	// Pad or clamp to modalHeight - 2
	for len(contentLines) < modalHeight-2 {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > modalHeight-2 {
		contentLines = contentLines[:modalHeight-2]
	}

	// 3. Footer
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", contentWidth)))
	footerHint := fmt.Sprintf(" Press %s or %s to close this shortcuts panel", KeyStyle.Render("[?]"), KeyStyle.Render("[Esc]"))
	contentLines = append(contentLines, lipgloss.NewStyle().MaxWidth(contentWidth).Render(footerHint))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1).
		Width(modalInnerWidth).
		Render(strings.Join(contentLines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
}
