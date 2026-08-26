package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHelpModal() string {
	modalWidth := m.width - 6
	if modalWidth > 92 {
		modalWidth = 92
	}
	if modalWidth < 52 {
		modalWidth = 52
	}
	modalInnerWidth := modalWidth - 4

	modalHeight := m.height - 4
	if modalHeight > 28 {
		modalHeight = 28
	}
	if modalHeight < 14 {
		modalHeight = 14
	}

	var contentLines []string

	// 1. Title
	titleText := "KEYBOARD SHORTCUTS CHEAT SHEET [?]"
	contentLines = append(contentLines, TitleStyle.Render(truncateVisualWidth(titleText, modalInnerWidth)))
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", modalInnerWidth)))

	// 2. Global Navigation
	contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [Global Navigation]"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("1 / d"), "Switch to Live Dashboard View"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("2 / s"), "Switch to Step History Explorer View"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("h"), "Open Terminology & Architecture Glossary"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Ctrl+p"), "Open Session Quick Switcher Modal"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("? / F1"), "Toggle this Keyboard Help Modal"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("q / Ctrl+c"), "Quit agent-observer"))
	contentLines = append(contentLines, "")

	// 3. Dashboard View
	if modalHeight >= 18 {
		contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [Dashboard View (Time Machine Playback)]"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("j / k / ↑ / ↓"), "Playback Previous / Next Historical Step"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Ctrl+d / Ctrl+u"), "Jump 10 Steps Forward / Backward"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("g / G"), "Jump to First Step / Jump to Latest (LIVE)"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Enter"), "Inspect Selected Step in History View"))
		contentLines = append(contentLines, "")
	}

	// 4. History View
	if modalHeight >= 22 {
		contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [Step History Explorer View]"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Tab / Enter / l"), "Switch Focus to Right Pane (Inspector)"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Esc / Left"), "Return Focus to Left Pane (Steps List)"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("j / k / ↑ / ↓"), "Select Step (Left) / Scroll Inspector (Right)"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Ctrl+d / Ctrl+u"), "Scroll Inspector by 10 Lines"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("v / V"), "Visual Selection Mode in Inspector"))
		contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("y"), "Yank (Copy) Selected Lines to Clipboard"))
		contentLines = append(contentLines, "")
	}

	// 5. Session Switcher
	contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [Session Switcher Modal (Vim-First)]"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Type text"), "Fuzzy filter sessions by ID or path"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Ctrl+j/k / ↑/↓"), "Navigate selection up / down"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Enter"), "Attach to highlighted session"))
	contentLines = append(contentLines, fmt.Sprintf("  %-22s : %s", KeyStyle.Render("Esc / Ctrl+p"), "Close switcher modal"))

	// Pad or clamp to modalHeight - 2
	for len(contentLines) < modalHeight-2 {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > modalHeight-2 {
		contentLines = contentLines[:modalHeight-2]
	}

	// Footer
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", modalInnerWidth)))
	footerHint := fmt.Sprintf(" Press %s or %s to close this help panel", KeyStyle.Render("[?]"), KeyStyle.Render("[Esc]"))
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
