package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// UI Theme Colors
var (
	ColorPrimary   = lipgloss.Color("#6C5CE7") // Royal Purple
	ColorSecondary = lipgloss.Color("#00CEC9") // Cyan / Teal
	ColorSuccess   = lipgloss.Color("#00B894") // Green
	ColorWarning   = lipgloss.Color("#FDCB6E") // Yellow
	ColorDanger    = lipgloss.Color("#D63031") // Red
	ColorDarkBg    = lipgloss.Color("#2D3436") // Dark Gray
	ColorLightText = lipgloss.Color("#DFE6E9") // Off-White
	ColorMuted     = lipgloss.Color("#636E72") // Gray
	ColorHighlight = lipgloss.Color("#E17055") // Orange
	ColorBorder    = lipgloss.Color("#4A4A68") // Border Indigo
)

// Lipgloss Styles
var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			MarginBottom(0)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				Padding(0, 1)

	SelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorPrimary)

	DimRowStyle = lipgloss.NewStyle().
			Foreground(ColorLightText)

	BadgeSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	BadgeWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning)

	BadgeDanger = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorDanger)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(0, 1)

	KeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)
)
