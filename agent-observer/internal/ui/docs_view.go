package ui

import (
	"bufio"
	"embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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
	filename := "docs/docs_en.md"
	if lang == "zh" {
		filename = "docs/docs_zh.md"
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

// formatDocItem renders a glossary definition with hanging indent across the full terminal width
func formatDocItem(key, desc string, contentWidth int) []string {
	paddedKey := fmt.Sprintf("%-24s", key)
	keyStyled := KeyStyle.Render(paddedKey)
	sepStyled := lipgloss.NewStyle().Foreground(ColorBorder).Render(" : ")
	prefixVisualWidth := 2 + 24 + 3 // 2 spaces + 24 chars + " : " (3 chars) = 29

	descMaxWidth := contentWidth - prefixVisualWidth
	if descMaxWidth < 20 {
		descMaxWidth = 20
	}

	// Wrap plain description text cleanly by visual terminal column width (CJK & ASCII aware)
	descChunks := wrapPlainText(desc, descMaxWidth)
	if len(descChunks) == 0 {
		return []string{"  " + keyStyled + sepStyled}
	}

	var lines []string
	// Line 1: "  " + Key + " : " + first chunk
	firstLine := "  " + keyStyled + sepStyled + lipgloss.NewStyle().Foreground(ColorLightText).Render(descChunks[0])
	lines = append(lines, firstLine)

	// Line 2+: 29 spaces hanging indent + subsequent chunks
	indentSpaces := strings.Repeat(" ", prefixVisualWidth)
	for i := 1; i < len(descChunks); i++ {
		lines = append(lines, indentSpaces+lipgloss.NewStyle().Foreground(ColorLightText).Render(descChunks[i]))
	}

	return lines
}

func wrapPlainText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	var chunks []string
	runes := []rune(text)
	var current []rune
	currW := 0

	for _, r := range runes {
		rw := runewidth.RuneWidth(r)
		if currW+rw > maxWidth {
			if len(current) > 0 {
				chunks = append(chunks, string(current))
				current = nil
				currW = 0
			}
		}
		current = append(current, r)
		currW += rw
	}
	if len(current) > 0 {
		chunks = append(chunks, string(current))
	}
	return chunks
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
		currentLang = "en"
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
		lines := formatDocItem(item.Key, item.Desc, contentWidth)
		rawLinesCount += len(lines)
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
		currentLang = "en"
	}
	langBadge := "[l: English]"
	if currentLang == "zh" {
		langBadge = "[l: 繁體中文]"
	}

	titleText := "ARCHITECTURE & CONTEXT DEFINITIONS  " + lipgloss.NewStyle().Foreground(ColorHighlight).Render(langBadge)
	if currentLang == "zh" {
		titleText = "架構名詞釋義與上下文辭典  " + lipgloss.NewStyle().Foreground(ColorHighlight).Render(langBadge)
	}

	// 2. Search / Filter Header
	searchStatus := "[Press / to search, l to switch language, Esc to clear]"
	if currentLang == "zh" {
		searchStatus = "[按 / 搜尋，按 l 切換中英，按 Esc 清除]"
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

		// Render with hanging indent and full terminal width wrapping
		formattedLines := formatDocItem(item.Key, item.Desc, contentWidth)
		rawLines = append(rawLines, formattedLines...)
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
