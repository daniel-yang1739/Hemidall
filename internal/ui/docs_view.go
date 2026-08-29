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

// DocSubBullet represents a sub-item under a major documentation topic
type DocSubBullet struct {
	Key  string
	Text string
}

// DocItem represents a major topic definition with optional sub-bullets
type DocItem struct {
	Category string
	Title    string
	Desc     string
	Bullets  []DocSubBullet
}

// loadDocDefinitions parses embedded Markdown files with full support for hierarchical bullets
func loadDocDefinitions(lang string) []DocItem {
	filename := "docs/docs_en.md"
	if lang == "zh" {
		filename = "docs/docs_zh.md"
	}

	data, err := docsFS.ReadFile(filename)
	if err != nil {
		return nil
	}

	var items []DocItem
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	currentCategory := "General"
	var currentItem *DocItem

	for scanner.Scan() {
		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "## ") {
			if currentItem != nil {
				items = append(items, *currentItem)
				currentItem = nil
			}
			currentCategory = strings.TrimPrefix(trimmed, "## ")
			continue
		}

		// Check if line is indented sub-bullet (2+ leading spaces or tab)
		isIndented := strings.HasPrefix(rawLine, "  ") || strings.HasPrefix(rawLine, "\t")

		if (strings.HasPrefix(trimmed, "* **") || strings.HasPrefix(trimmed, "- **")) && isIndented && currentItem != nil {
			line := strings.TrimPrefix(trimmed, "* ")
			line = strings.TrimPrefix(line, "- ")
			parts := strings.SplitN(line, "** : ", 2)
			if len(parts) == 2 {
				k := strings.Trim(parts[0], "* ")
				currentItem.Bullets = append(currentItem.Bullets, DocSubBullet{
					Key:  strings.TrimSpace(k),
					Text: strings.TrimSpace(parts[1]),
				})
			} else {
				k := strings.Trim(line, "* ")
				currentItem.Bullets = append(currentItem.Bullets, DocSubBullet{
					Text: strings.TrimSpace(k),
				})
			}
			continue
		}

		if (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) && isIndented && currentItem != nil {
			line := strings.TrimPrefix(trimmed, "* ")
			line = strings.TrimPrefix(line, "- ")
			parts := strings.SplitN(line, " : ", 2)
			if len(parts) == 2 {
				currentItem.Bullets = append(currentItem.Bullets, DocSubBullet{
					Key:  strings.TrimSpace(parts[0]),
					Text: strings.TrimSpace(parts[1]),
				})
			} else {
				currentItem.Bullets = append(currentItem.Bullets, DocSubBullet{
					Text: strings.TrimSpace(line),
				})
			}
			continue
		}

		// Top-level Item
		if strings.HasPrefix(trimmed, "* **") || strings.HasPrefix(trimmed, "- **") {
			if currentItem != nil {
				items = append(items, *currentItem)
				currentItem = nil
			}

			line := strings.TrimPrefix(trimmed, "* ")
			line = strings.TrimPrefix(line, "- ")
			parts := strings.SplitN(line, "** : ", 2)
			if len(parts) == 2 {
				k := strings.Trim(parts[0], "* ")
				currentItem = &DocItem{
					Category: currentCategory,
					Title:    strings.TrimSpace(k),
					Desc:     strings.TrimSpace(parts[1]),
				}
			} else {
				k := strings.Trim(line, "* ")
				currentItem = &DocItem{
					Category: currentCategory,
					Title:    strings.TrimSpace(k),
				}
			}
			continue
		}

		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			if currentItem != nil {
				items = append(items, *currentItem)
				currentItem = nil
			}
			line := strings.TrimPrefix(trimmed, "* ")
			line = strings.TrimPrefix(line, "- ")
			parts := strings.SplitN(line, " : ", 2)
			if len(parts) == 2 {
				currentItem = &DocItem{
					Category: currentCategory,
					Title:    strings.TrimSpace(parts[0]),
					Desc:     strings.TrimSpace(parts[1]),
				}
			} else {
				currentItem = &DocItem{
					Category: currentCategory,
					Title:    strings.TrimSpace(line),
				}
			}
			continue
		}
	}

	if currentItem != nil {
		items = append(items, *currentItem)
	}

	return items
}

// formatDocItem renders a structured DocItem with colors, hierarchy, bullets, and word wrapping
func formatDocItem(item DocItem, contentWidth int) []string {
	var lines []string

	bulletIcon := lipgloss.NewStyle().Foreground(ColorHighlight).Render("• ")
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render(item.Title)

	if item.Desc != "" {
		sepStyled := lipgloss.NewStyle().Foreground(ColorBorder).Render(" : ")
		prefixW := 2 + 2 + lipgloss.Width(item.Title) + 3 // "  • " + title + " : "
		descMaxWidth := contentWidth - prefixW

		if descMaxWidth < 25 {
			// Compact stacked rendering on narrow width
			lines = append(lines, "  "+bulletIcon+titleStyled)
			descChunks := wrapPlainText(item.Desc, contentWidth-6)
			for _, chunk := range descChunks {
				lines = append(lines, "    "+lipgloss.NewStyle().Foreground(ColorLightText).Render(chunk))
			}
		} else {
			// Hanging indent rendering
			descChunks := wrapPlainText(item.Desc, descMaxWidth)
			if len(descChunks) > 0 {
				lines = append(lines, "  "+bulletIcon+titleStyled+sepStyled+lipgloss.NewStyle().Foreground(ColorLightText).Render(descChunks[0]))
				indent := strings.Repeat(" ", prefixW)
				for i := 1; i < len(descChunks); i++ {
					lines = append(lines, indent+lipgloss.NewStyle().Foreground(ColorLightText).Render(descChunks[i]))
				}
			}
		}
	} else {
		lines = append(lines, "  "+bulletIcon+titleStyled)
	}

	// Render hierarchical sub-bullets with '▸'
	for _, b := range item.Bullets {
		subIcon := lipgloss.NewStyle().Foreground(ColorPrimary).Render("    ▸ ")
		if b.Key != "" {
			keyStyled := lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(b.Key)
			sep := lipgloss.NewStyle().Foreground(ColorBorder).Render(" : ")
			prefixW := 6 + lipgloss.Width(b.Key) + 3 // "    ▸ " (6) + key + " : " (3)
			descMaxWidth := contentWidth - prefixW

			if descMaxWidth < 20 {
				lines = append(lines, subIcon+keyStyled+sep)
				chunks := wrapPlainText(b.Text, contentWidth-8)
				for _, chunk := range chunks {
					lines = append(lines, "      "+lipgloss.NewStyle().Foreground(ColorLightText).Render(chunk))
				}
			} else {
				chunks := wrapPlainText(b.Text, descMaxWidth)
				if len(chunks) > 0 {
					lines = append(lines, subIcon+keyStyled+sep+lipgloss.NewStyle().Foreground(ColorLightText).Render(chunks[0]))
					indent := strings.Repeat(" ", prefixW)
					for i := 1; i < len(chunks); i++ {
						lines = append(lines, indent+lipgloss.NewStyle().Foreground(ColorLightText).Render(chunks[i]))
					}
				}
			}
		} else {
			chunks := wrapPlainText(b.Text, contentWidth-8)
			for i, chunk := range chunks {
				if i == 0 {
					lines = append(lines, subIcon+lipgloss.NewStyle().Foreground(ColorLightText).Render(chunk))
				} else {
					lines = append(lines, "      "+lipgloss.NewStyle().Foreground(ColorLightText).Render(chunk))
				}
			}
		}
	}

	// Spacing line between major items for comfortable human reading
	lines = append(lines, "")
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

// matchesQuery checks if a DocItem matches the search query across all fields
func (item DocItem) matchesQuery(query string) bool {
	if query == "" {
		return true
	}
	if strings.Contains(strings.ToLower(item.Category), query) ||
		strings.Contains(strings.ToLower(item.Title), query) ||
		strings.Contains(strings.ToLower(item.Desc), query) {
		return true
	}
	for _, b := range item.Bullets {
		if strings.Contains(strings.ToLower(b.Key), query) ||
			strings.Contains(strings.ToLower(b.Text), query) {
			return true
		}
	}
	return false
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
		if !item.matchesQuery(query) {
			continue
		}
		matchedCount++
		if item.Category != currentCategory {
			if currentCategory != "" {
				rawLinesCount++
			}
			currentCategory = item.Category
			rawLinesCount += 2 // Category header + divider
		}
		lines := formatDocItem(item, contentWidth)
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
	langBadge := "[t: English]"
	if currentLang == "zh" {
		langBadge = "[t: 繁體中文]"
	}

	titleText := "ARCHITECTURE & CONTEXT DEFINITIONS  " + lipgloss.NewStyle().Foreground(ColorHighlight).Render(langBadge)
	if currentLang == "zh" {
		titleText = "架構名詞釋義與上下文辭典  " + lipgloss.NewStyle().Foreground(ColorHighlight).Render(langBadge)
	}

	// 2. Search / Filter Header
	searchStatus := "[Press / to search, t to switch language, Esc to clear]"
	if currentLang == "zh" {
		searchStatus = "[按 / 搜尋，按 t 切換中英，按 Esc 清除]"
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
		if !item.matchesQuery(query) {
			continue
		}

		matchedCount++
		if item.Category != currentCategory {
			if currentCategory != "" {
				rawLines = append(rawLines, "")
			}
			currentCategory = item.Category
			rawLines = append(rawLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🔷 "+item.Category))
			rawLines = append(rawLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("┈", contentWidth)))
		}

		// Render item with hierarchical styling
		formattedLines := formatDocItem(item, contentWidth)
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
