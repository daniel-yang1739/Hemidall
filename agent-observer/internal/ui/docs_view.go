package ui

import (
	"bufio"
	"embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

//go:embed docs/*.md
var docsFS embed.FS

// DocDefinitionItem represents an architecture, dimension, or token concept definition
type DocDefinitionItem struct {
	Category string
	Key      string
	Desc     string
}

// loadDocDefinitions parses embedded Markdown files by language (en or zh)
func loadDocDefinitions(lang string) []DocDefinitionItem {
	filename := "docs/docs_zh.md"
	if lang == "en" {
		filename = "docs/docs_en.md"
	}

	data, err := docsFS.ReadFile(filename)
	if err != nil {
		return nil
	}

	var items []DocDefinitionItem
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	currentCategory := "General"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "## ") {
			currentCategory = strings.TrimPrefix(line, "## ")
			continue
		}
		if (strings.HasPrefix(line, "* **") || strings.HasPrefix(line, "- **")) && strings.Contains(line, "** : ") {
			line = strings.TrimPrefix(line, "* ")
			line = strings.TrimPrefix(line, "- ")
			parts := strings.SplitN(line, "** : ", 2)
			if len(parts) == 2 {
				key := strings.TrimPrefix(parts[0], "**")
				desc := parts[1]
				items = append(items, DocDefinitionItem{
					Category: currentCategory,
					Key:      strings.TrimSpace(key),
					Desc:     strings.TrimSpace(desc),
				})
			}
		}
	}

	return items
}

// getDocsMaxScroll calculates the maximum allowable scroll offset to prevent over-scrolling
func (m Model) getDocsMaxScroll() int {
	boxInnerWidth := m.width - 2
	if boxInnerWidth < 40 {
		boxInnerWidth = 40
	}
	contentWidth := boxInnerWidth - 2
	innerRowsLimit := m.height - 4
	if innerRowsLimit < 4 {
		innerRowsLimit = 4
	}

	currentLang := m.docsLang
	if currentLang == "" {
		currentLang = "zh"
	}
	defs := loadDocDefinitions(currentLang)
	query := strings.ToLower(strings.TrimSpace(m.docsSearchQuery))

	rawLinesCount := 3 // title, searchBar, divider
	currentCategory := ""
	matchedCount := 0

	for _, item := range defs {
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
				rawLinesCount++
			}
			currentCategory = item.Category
			rawLinesCount++
		}
		paddedKey := fmt.Sprintf("%-24s", item.Key)
		entryLine := "  " + paddedKey + " : " + item.Desc
		wrapped := wrapVisualLines(entryLine, contentWidth-2)
		rawLinesCount += len(wrapped)
	}

	if matchedCount == 0 {
		rawLinesCount += 3
	}

	maxScroll := rawLinesCount - innerRowsLimit
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
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

	// 1. Title with Language Indicator
	currentLang := m.docsLang
	if currentLang == "" {
		currentLang = "zh"
	}
	langBadge := "[L: 繁體中文]"
	if currentLang == "en" {
		langBadge = "[L: English]"
	}

	titleText := "ARCHITECTURE & CONTEXT DEFINITIONS  " + lipgloss.NewStyle().Foreground(ColorHighlight).Render(langBadge)
	if currentLang == "zh" {
		titleText = "架構名詞釋義與上下文辭典  " + lipgloss.NewStyle().Foreground(ColorHighlight).Render(langBadge)
	}

	// 2. Search / Filter Header
	searchStatus := "[Press / to search, L to switch language, Esc to clear]"
	if currentLang == "zh" {
		searchStatus = "[按 / 搜尋，按 L 切換中英，按 Esc 清除]"
	}
	if m.isDocsSearching {
		searchStatus = "[SEARCHING: Type to filter, Enter/Esc to finish]"
		if currentLang == "zh" {
			searchStatus = "[搜尋中: 輸入字元即時過濾，Enter/Esc 結束]"
		}
	}
	cursorChar := ""
	if m.isDocsSearching {
		cursorChar = "█"
	}
	filterLabel := "Filter"
	if currentLang == "zh" {
		filterLabel = "過濾"
	}
	searchBar := fmt.Sprintf("%s: [%s%s]  %s",
		filterLabel,
		m.docsSearchQuery,
		lipgloss.NewStyle().Foreground(ColorHighlight).Render(cursorChar),
		lipgloss.NewStyle().Foreground(ColorMuted).Render(searchStatus))

	rawLines = append(rawLines, TitleStyle.Render(truncateVisualWidth(titleText, contentWidth)))
	rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(truncateVisualWidth(searchBar, contentWidth)))
	rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", contentWidth)))

	// 3. Load Definitions by Language
	defs := loadDocDefinitions(currentLang)

	// 4. Filter & Render Definition Items
	query := strings.ToLower(strings.TrimSpace(m.docsSearchQuery))
	currentCategory := ""
	matchedCount := 0

	for _, item := range defs {
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
		noMatchMsg := "  No matching architecture definitions found for '" + m.docsSearchQuery + "'."
		clearMsg := "  Press [Esc] to clear search query."
		if currentLang == "zh" {
			noMatchMsg = "  查無符合 '" + m.docsSearchQuery + "' 的架構定義。"
			clearMsg = "  按 [Esc] 可清除搜尋條件。"
		}
		rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorWarning).Render(noMatchMsg))
		rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorMuted).Render(clearMsg))
	}

	// 5. Virtual Viewport Slicing
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
