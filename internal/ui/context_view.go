package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"heimdall/internal/core"
)

type contextTreeItemKind int

const (
	contextItemSystemPrompt contextTreeItemKind = iota
	contextItemPromptSection
	contextItemMCPStatus
	contextItemTools
	contextItemCheckpoint
	contextItemHistory
)

type contextTreeItem struct {
	kind         contextTreeItemKind
	sectionIndex int
}

const (
	contextHistoryListHeaderLines = 4
	contextHistoryListBorderLines = 4
	minimumContextHistoryRows     = 1
	bytesPerKilobyte              = 1024
	bytesPerMegabyte              = bytesPerKilobyte * bytesPerKilobyte
	bytesPerGigabyte              = bytesPerMegabyte * bytesPerKilobyte
)

func (m Model) renderContextView() string {
	width := m.width
	height := m.height - 2 // Minus header and footer
	if height < 8 {
		height = 8
	}
	if width < 40 {
		width = 40
	}

	payload := m.cachedContextPayload()
	payload.IsRawMode = m.isContextRawMode

	// 1. Calculate Split Widths (30% Left Tree, 70% Right Inspector)
	leftWidth := width * 30 / 100
	if leftWidth < 26 {
		leftWidth = 26
	}
	if leftWidth > 38 {
		leftWidth = 38
	}
	rightWidth := width - leftWidth
	if rightWidth < 30 {
		rightWidth = 30
	}

	leftTree := m.renderContextTree(payload, leftWidth, height)
	if m.contextHistoryList {
		leftTree = m.renderCurrentHistoryList(payload, leftWidth, height)
	}
	rightInspector := m.renderContextInspector(payload, rightWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftTree, rightInspector)
}

func (m Model) renderContextTree(payload core.AgentContextPayload, width, height int) string {
	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Width(width - 2).
		Height(height - 2)

	if m.contextFocusPane == FocusList {
		boxStyle = boxStyle.BorderForeground(ColorSecondary)
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(" MODEL INPUT SNAPSHOT ")
	snapshotStatus := " [Snapshot unavailable]"
	if payload.SnapshotAvailable {
		snapshotStatus = fmt.Sprintf(" [Total: %s Tok]", formatNumber(payload.TotalTokens))
	}
	totalContext := lipgloss.NewStyle().Foreground(ColorHighlight).Render(snapshotStatus)

	var treeLines []string
	items := contextTreeItems(payload)
	systemTokens, toolsTokens, historyTokens := m.contextTreeTokenCounts(payload)
	toolCount := persistedToolCount(payload)
	treeLines = append(treeLines, title)
	treeLines = append(treeLines, totalContext)
	treeLines = append(treeLines, "")

	treeLines = append(treeLines, renderContextSectionHeader("SYSTEM PROMPT", systemTokens, innerWidth)...)
	treeLines = append(treeLines, m.renderTreeLeaf(0, "  - System Prompt"))
	for sectionIndex, section := range payload.SystemPromptSections {
		treeLines = append(treeLines, m.renderTreeLeaf(sectionIndex+1, "    - "+formatPromptSectionTag(section.Tag)))
	}
	if !hasMCPSystemPromptSection(payload) {
		mcpStatusPosition := contextTreeItemPosition(items, contextItemMCPStatus)
		treeLines = append(treeLines, m.renderTreeLeaf(mcpStatusPosition, "    - MCP (Not Present)"))
	}
	treeLines = append(treeLines, "")

	toolsPosition := contextTreeItemPosition(items, contextItemTools)
	treeLines = append(treeLines, renderContextSectionHeader("TOOL DEFINITIONS", toolsTokens, innerWidth)...)
	treeLines = append(treeLines, m.renderTreeLeaf(toolsPosition, fmt.Sprintf("  - Tools (%d)", toolCount)))
	treeLines = append(treeLines, "")

	checkpointPosition := contextTreeItemPosition(items, contextItemCheckpoint)
	historyPosition := contextTreeItemPosition(items, contextItemHistory)
	treeLines = append(treeLines, renderContextSectionHeader("CONVERSATION MESSAGES", historyTokens, innerWidth)...)
	treeLines = append(treeLines, m.renderTreeLeaf(checkpointPosition, "  - Checkpoint"))
	treeLines = append(treeLines, m.renderTreeLeaf(historyPosition, fmt.Sprintf("  - History Records (%d)", len(currentHistoryRecords(payload)))))

	for i, tl := range treeLines {
		if lipgloss.Width(tl) > innerWidth {
			treeLines[i] = lipgloss.NewStyle().MaxWidth(innerWidth).Render(tl)
		}
	}

	content := strings.Join(treeLines, "\n")
	return boxStyle.Render(content)
}

func contextTreeItems(payload core.AgentContextPayload) []contextTreeItem {
	items := []contextTreeItem{{kind: contextItemSystemPrompt}}
	for sectionIndex := range payload.SystemPromptSections {
		items = append(items, contextTreeItem{kind: contextItemPromptSection, sectionIndex: sectionIndex})
	}
	if !hasMCPSystemPromptSection(payload) {
		items = append(items, contextTreeItem{kind: contextItemMCPStatus})
	}
	return append(items,
		contextTreeItem{kind: contextItemTools},
		contextTreeItem{kind: contextItemCheckpoint},
		contextTreeItem{kind: contextItemHistory},
	)
}

func hasSystemPromptSection(payload core.AgentContextPayload, tag string) bool {
	for _, section := range payload.SystemPromptSections {
		if strings.EqualFold(section.Tag, tag) {
			return true
		}
	}
	return false
}

func hasMCPSystemPromptSection(payload core.AgentContextPayload) bool {
	return hasSystemPromptSection(payload, "mcp") || hasSystemPromptSection(payload, "mcp_servers")
}

func contextTreeItemPosition(items []contextTreeItem, kind contextTreeItemKind) int {
	for position, item := range items {
		if item.kind == kind {
			return position
		}
	}
	return 0
}

func (m Model) selectedContextTreeItem(payload core.AgentContextPayload) contextTreeItem {
	items := contextTreeItems(payload)
	if m.contextSubItemIndex < 0 || m.contextSubItemIndex >= len(items) {
		return items[0]
	}
	return items[m.contextSubItemIndex]
}

func formatPromptSectionTag(tag string) string {
	if strings.EqualFold(tag, "mcp") {
		return "MCP"
	}
	if strings.EqualFold(tag, "mcp_servers") {
		return "MCP Servers"
	}
	words := strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(tag))
	for index, word := range words {
		if word != "" {
			words[index] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func (m Model) contextTreeTokenCounts(payload core.AgentContextPayload) (int, int, int) {
	if !payload.SnapshotAvailable {
		return 0, 0, 0
	}
	if m.contextEstimateHistoryCount == len(m.history) && m.contextEstimate.Available {
		return m.contextEstimate.SystemTokens, m.contextEstimate.ToolsTokens, m.contextEstimate.HistoryTokens
	}
	return 0, 0, 0
}

func currentHistoryRecords(payload core.AgentContextPayload) []core.PersistedContextRecord {
	if len(payload.PersistedRecords) > 0 && payload.PersistedRecords[0].IsCompactedCheckpoint {
		return payload.PersistedRecords[1:]
	}
	return payload.PersistedRecords
}

func compactedCheckpoint(payload core.AgentContextPayload) *core.PersistedContextRecord {
	if len(payload.PersistedRecords) > 0 && payload.PersistedRecords[0].IsCompactedCheckpoint {
		return &payload.PersistedRecords[0]
	}
	return nil
}

func historyListItemCount(payload core.AgentContextPayload) int {
	return len(currentHistoryRecords(payload))
}

func (m Model) renderCurrentHistoryList(payload core.AgentContextPayload, width, height int) string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorSecondary).Width(width - 2).Height(height - 2)
	records := currentHistoryRecords(payload)
	if len(records) == 0 {
		return box.Render(strings.Join([]string{lipgloss.NewStyle().Foreground(ColorMuted).Render("← [Esc] Back to Context"), "", lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("ACTIVE HISTORY (0)"), "", "No decoded persisted records are available."}, "\n"))
	}
	itemCount := historyListItemCount(payload)
	selectedOffset := m.contextHistoryIndex
	if selectedOffset >= itemCount {
		selectedOffset = itemCount - 1
	}
	maxRows := height - contextHistoryListBorderLines - contextHistoryListHeaderLines
	if maxRows < minimumContextHistoryRows {
		maxRows = minimumContextHistoryRows
	}
	if maxRows < minimumContextHistoryRows {
		maxRows = minimumContextHistoryRows
	}
	startOffset := m.contextHistoryScrollOffset
	if startOffset < 0 {
		startOffset = 0
	}
	if startOffset >= itemCount {
		startOffset = itemCount - 1
	}
	endOffset := startOffset + maxRows
	if endOffset > itemCount {
		endOffset = itemCount
	}
	if selectedOffset < startOffset {
		startOffset = selectedOffset
		endOffset = startOffset + maxRows
		if endOffset > itemCount {
			endOffset = itemCount
		}
	}
	if selectedOffset >= endOffset {
		endOffset = selectedOffset + 1
		startOffset = endOffset - maxRows
	}
	lines := []string{lipgloss.NewStyle().Foreground(ColorMuted).Render("← [Esc] Back to Context"), "", lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("ACTIVE HISTORY (%d)", len(records))), ""}
	positionWidth := len(fmt.Sprintf("%d", records[len(records)-1].Position))
	referenceWidth := 0
	sizeWidth := 0
	for _, record := range records {
		reference := formatObservedEventReference(record.ObservedSequence)
		if len(reference) > referenceWidth {
			referenceWidth = len(reference)
		}
		value, _ := formatContextRecordSize(record.ByteSize)
		if len(value) > sizeWidth {
			sizeWidth = len(value)
		}
	}
	for offset := startOffset; offset < endOffset; offset++ {
		r := records[len(records)-1-offset]
		sizeValue, sizeUnit := formatContextRecordSize(r.ByteSize)
		line := fmt.Sprintf("  #%-*d  %-*s  %-*s %s", positionWidth, r.Position, referenceWidth, formatObservedEventReference(r.ObservedSequence), sizeWidth, sizeValue, sizeUnit)
		if offset == selectedOffset {
			line = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render("> " + strings.TrimSpace(line))
		}
		lines = append(lines, line)
	}
	return box.Render(strings.Join(lines, "\n"))
}

func formatObservedEventReference(sequence uint64) string {
	if sequence == 0 {
		return ""
	}
	return fmt.Sprintf("[%d]", sequence)
}

func formatContextRecordSize(bytes int) (string, string) {
	switch {
	case bytes >= bytesPerGigabyte:
		return fmt.Sprintf("%.1f", float64(bytes)/float64(bytesPerGigabyte)), "GB"
	case bytes >= bytesPerMegabyte:
		return fmt.Sprintf("%.1f", float64(bytes)/float64(bytesPerMegabyte)), "MB"
	case bytes >= bytesPerKilobyte:
		return fmt.Sprintf("%.1f", float64(bytes)/float64(bytesPerKilobyte)), "KB"
	default:
		return fmt.Sprintf("%d", bytes), "Bytes"
	}
}

func (m Model) renderTreeLeaf(idx int, label string) string {
	isSelected := (m.contextSubItemIndex == idx)
	if isSelected {
		trimmed := strings.TrimPrefix(label, "  ")
		prefix := "> "
		styled := lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render(prefix + trimmed)
		return styled
	}
	return lipgloss.NewStyle().Foreground(ColorLightText).Render(label)
}

func renderContextSectionHeader(title string, tokens, width int) []string {
	tokenLabel := fmt.Sprintf("%s Tok", formatCompactNumber(tokens))
	combined := fmt.Sprintf("%s (%s)", title, tokenLabel)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText)
	if lipgloss.Width(combined) <= width {
		return []string{headerStyle.Render(combined)}
	}
	return []string{
		headerStyle.Render(title),
		lipgloss.NewStyle().Foreground(ColorMuted).Render("  " + tokenLabel),
	}
}

func (m Model) renderContextInspector(payload core.AgentContextPayload, width, height int) string {
	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}
	innerHeight := height - 4
	if innerHeight < 2 {
		innerHeight = 2
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Width(width - 2).
		Height(height - 2)

	if m.contextFocusPane == FocusDetail {
		boxStyle = boxStyle.BorderForeground(ColorSecondary)
	}

	modeBadge := "[MODE: REFINED [r]]"
	if m.isContextRawMode {
		modeBadge = "[MODE: SNAPSHOT JSON [r]]"
	}
	modeStyled := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(modeBadge)

	headerTitle := m.getInspectorHeaderTitle(payload)
	headerLine := fmt.Sprintf("%s %s", lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(headerTitle), modeStyled)
	headerLine = lipgloss.NewStyle().MaxWidth(innerWidth).Render(headerLine)

	sepWidth := innerWidth
	if sepWidth < 1 {
		sepWidth = 1
	}
	sepLine := strings.Repeat("─", sepWidth)

	bodyLines := m.contextInspectorLines
	if !m.hasContextInspectorCache(innerWidth, innerHeight) {
		bodyLines = m.buildRefinedInspectorLines(payload, innerWidth)
		if m.isContextRawMode {
			bodyLines = m.buildRawWireLines(payload, innerWidth)
		}
	}

	// Apply scrolling
	maxScroll := len(bodyLines) - innerHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.contextDetailScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	visibleLines := bodyLines
	if len(visibleLines) > scroll {
		visibleLines = visibleLines[scroll:]
	}
	if len(visibleLines) > innerHeight {
		visibleLines = visibleLines[:innerHeight]
	}

	// Ensure every visible line strictly respects innerWidth
	for i, l := range visibleLines {
		if lipgloss.Width(l) > innerWidth {
			visibleLines[i] = lipgloss.NewStyle().MaxWidth(innerWidth).Render(l)
		}
	}

	content := headerLine + "\n" + sepLine + "\n" + strings.Join(visibleLines, "\n")
	return boxStyle.Render(content)
}

func (m Model) contextInspectorMaxScroll() int {
	return m.contextInspectorMax
}

func (m Model) contextInspectorDimensions() (int, int) {
	width := m.width
	height := m.height - 2
	if height < 8 {
		height = 8
	}
	if width < 40 {
		width = 40
	}
	leftWidth := width * 30 / 100
	if leftWidth < 26 {
		leftWidth = 26
	}
	if leftWidth > 38 {
		leftWidth = 38
	}
	rightWidth := width - leftWidth
	if rightWidth < 30 {
		rightWidth = 30
	}
	innerWidth := rightWidth - 4
	innerHeight := height - 4
	if innerWidth < 10 {
		innerWidth = 10
	}
	if innerHeight < 2 {
		innerHeight = 2
	}
	return innerWidth, innerHeight
}

func (m Model) getInspectorHeaderTitle(payload core.AgentContextPayload) string {
	item := m.selectedContextTreeItem(payload)
	switch item.kind {
	case contextItemSystemPrompt:
		return fmt.Sprintf("COMPLETE SYSTEM PROMPT (%s Tok)", formatCompactNumber(persistedSnapshotTokenCount(payload, payload.SystemPrompt)))
	case contextItemPromptSection:
		section := payload.SystemPromptSections[item.sectionIndex]
		return fmt.Sprintf("%s (%s Tok)", strings.ToUpper(formatPromptSectionTag(section.Tag)), formatCompactNumber(persistedSnapshotTokenCount(payload, section.Content)))
	case contextItemMCPStatus:
		return "MCP (NOT PRESENT)"
	case contextItemTools:
		return fmt.Sprintf("TOOL DEFINITIONS (%d Tools)", persistedToolCount(payload))
	case contextItemCheckpoint:
		return "COMPACTED CHECKPOINT"
	case contextItemHistory:
		records := currentHistoryRecords(payload)
		if m.contextHistoryList && len(records) > 0 {
			index := len(records) - 1 - m.contextHistoryIndex
			if index < 0 {
				index = 0
			}
			return fmt.Sprintf("RECORD #%d · LATEST PERSISTED SNAPSHOT", records[index].Position)
		}
		return fmt.Sprintf("ACTIVE HISTORY RECORDS (%d)", len(records))
	default:
		return "MODEL INPUT SNAPSHOT"
	}
}

func persistedSnapshotTokenCount(payload core.AgentContextPayload, text string) int {
	if !payload.SnapshotAvailable {
		return 0
	}
	return core.CountTokens(text)
}

func persistedToolCount(payload core.AgentContextPayload) int {
	if !payload.SnapshotAvailable {
		return 0
	}
	return len(payload.NativeTools)
}

func (m Model) buildRefinedInspectorLines(payload core.AgentContextPayload, width int) []string {
	var lines []string
	item := m.selectedContextTreeItem(payload)

	switch item.kind {
	case contextItemSystemPrompt:
		lines = appendPersistedSnapshotText(lines, payload, "COMPLETE SYSTEM PROMPT", payload.SystemPrompt, width)

	case contextItemPromptSection:
		if !payload.SnapshotAvailable {
			lines = append(lines, "  Persisted model-input snapshot unavailable.")
			break
		}
		section := payload.SystemPromptSections[item.sectionIndex]
		sectionTitle := strings.ToUpper(formatPromptSectionTag(section.Tag))
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(sectionTitle+":"), "")
		if strings.TrimSpace(section.Content) == "" {
			lines = append(lines, "  (Empty)")
			break
		}
		if section.Tag == "user_rules" {
			lines = append(lines, formatPersistedUserRuleLines(section.Content, width)...)
			break
		}
		for _, rawLine := range strings.Split(section.Content, "\n") {
			lines = append(lines, wrapText("  "+rawLine, width)...)
		}

	case contextItemMCPStatus:
		if !payload.SnapshotAvailable {
			lines = append(lines, "  Persisted model-input snapshot unavailable.")
			break
		}
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("MCP:"),
			"",
			"  No <mcp> section is present in the persisted system prompt.",
			"  Local MCP configuration is not part of the model-input snapshot.",
		)

	case contextItemTools:
		if !payload.SnapshotAvailable {
			lines = append(lines, "  Persisted model-input snapshot unavailable.")
			break
		}
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(fmt.Sprintf("🛠️ TOOL SIGNATURES & DESCRIPTIONS (%d Registered Tools):", len(payload.NativeTools))))
		lines = append(lines, "")
		for _, tool := range payload.NativeTools {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("• "+tool.Signature))
			for _, wL := range wrapText("  └ "+tool.Description, width) {
				lines = append(lines, lipgloss.NewStyle().Foreground(ColorLightText).Render(wL))
			}
			lines = append(lines, "")
		}

	case contextItemCheckpoint:
		if !payload.SnapshotAvailable {
			lines = append(lines, "  Persisted model-input snapshot unavailable.")
			break
		}
		if checkpoint := compactedCheckpoint(payload); checkpoint != nil {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("📦 COMPACTED CHECKPOINT:"), "")
			for _, l := range strings.Split(checkpoint.PrimaryText, "\n") {
				lines = append(lines, wrapText("  "+l, width)...)
			}
			lines = append(lines, "", "  • Record #1 is separated from Active History because it is a compacted summary.")
			break
		}
		lines = append(lines, "  No compacted checkpoint is present in the persisted snapshot.")

	case contextItemHistory:
		if !payload.SnapshotAvailable {
			lines = append(lines, "  Persisted model-input snapshot unavailable.")
			break
		}
		records := currentHistoryRecords(payload)
		if !m.contextHistoryList {
			lines = append(lines, "  • Press [Enter] to browse active events from the latest persisted snapshot.")
			break
		}
		if len(records) == 0 {
			lines = append(lines, "  • No decoded persisted records are available.")
			break
		}
		index := len(records) - 1 - m.contextHistoryIndex
		if index < 0 {
			index = 0
		}
		record := records[index]
		roleDisplay := record.RoleName
		if roleDisplay == "" {
			roleDisplay = fmt.Sprintf("KIND_%d", record.ObservedKind)
		}
		if record.RoleDescription != "" {
			roleDisplay = fmt.Sprintf("%s (%s)", roleDisplay, record.RoleDescription)
		}
		lines = append(lines,
			fmt.Sprintf("  • Snapshot position: #%d", record.Position),
			fmt.Sprintf("  • Record size: %s bytes", formatNumber(record.ByteSize)),
			fmt.Sprintf("  • Message role: %s", roleDisplay),
			"",
		)
		for _, l := range strings.Split(record.PrimaryText, "\n") {
			lines = append(lines, wrapText("  "+l, width)...)
		}
		if record.HasPrivateContent {
			lines = append(lines, "", "  • Private content is present and intentionally not displayed.")
		}

	}

	return lines
}

func appendPersistedSnapshotText(lines []string, payload core.AgentContextPayload, title, content string, width int) []string {
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(title+":"), "")
	if !payload.SnapshotAvailable {
		return append(lines, "  Persisted model-input snapshot unavailable.")
	}
	if strings.TrimSpace(content) == "" {
		return append(lines, "  (Empty)")
	}
	for _, rawLine := range strings.Split(content, "\n") {
		lines = append(lines, wrapText("  "+rawLine, width)...)
	}
	return lines
}

func (m Model) buildRawWireLines(payload core.AgentContextPayload, width int) []string {
	wireJSON, err := m.serializeContextRaw(payload)
	if err != nil {
		return []string{"Error serializing wire payload: " + err.Error()}
	}

	subcatName := m.getSelectedContextItemName(payload)
	var formattedLines []string
	formattedLines = append(formattedLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(fmt.Sprintf("// Persisted model-input snapshot: %s", subcatName)))
	formattedLines = append(formattedLines, "")

	for _, line := range strings.Split(wireJSON, "\n") {
		// Highlight JSON keys
		if strings.Contains(line, `": `) {
			parts := strings.SplitN(line, `": `, 2)
			keyPart := lipgloss.NewStyle().Foreground(ColorSecondary).Render(parts[0] + `": `)
			valPart := lipgloss.NewStyle().Foreground(ColorLightText).Render(parts[1])
			fullLine := keyPart + valPart
			if lipgloss.Width(fullLine) > width && width > 10 {
				formattedLines = append(formattedLines, wrapText(fullLine, width)...)
			} else {
				formattedLines = append(formattedLines, fullLine)
			}
		} else {
			if lipgloss.Width(line) > width && width > 10 {
				formattedLines = append(formattedLines, wrapText(line, width)...)
			} else {
				formattedLines = append(formattedLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(line))
			}
		}
	}

	return formattedLines
}

func (m Model) serializeContextRaw(payload core.AgentContextPayload) (string, error) {
	if !payload.SnapshotAvailable {
		return "{\n  \"status\": \"PERSISTED_MODEL_INPUT_SNAPSHOT_UNAVAILABLE\"\n}", nil
	}
	item := m.selectedContextTreeItem(payload)
	switch item.kind {
	case contextItemSystemPrompt:
		return core.SerializeSubcategoryRaw(payload, core.SubcatSystemPrompt)
	case contextItemPromptSection:
		return core.SerializeSystemPromptSectionRaw(payload, item.sectionIndex)
	case contextItemMCPStatus:
		return "{\n  \"tag\": \"mcp\",\n  \"status\": \"NOT_PRESENT\",\n  \"reason\": \"No <mcp> section is present in the persisted system prompt.\"\n}", nil
	case contextItemTools:
		return core.SerializeSubcategoryRaw(payload, core.SubcatTools)
	case contextItemCheckpoint:
		return core.SerializeSubcategoryRaw(payload, core.SubcatAnchor)
	}
	if !m.contextHistoryList {
		return core.SerializePersistedActiveEventRaw(payload, 0)
	}
	records := currentHistoryRecords(payload)
	if len(records) == 0 {
		return core.SerializePersistedActiveEventRaw(payload, 0)
	}
	index := len(records) - 1 - m.contextHistoryIndex
	if index < 0 {
		index = 0
	}
	return core.SerializePersistedActiveEventRaw(payload, records[index].Position)
}

func (m Model) getSelectedContextItemName(payload core.AgentContextPayload) string {
	item := m.selectedContextTreeItem(payload)
	switch item.kind {
	case contextItemSystemPrompt:
		return "Complete System Prompt"
	case contextItemPromptSection:
		return formatPromptSectionTag(payload.SystemPromptSections[item.sectionIndex].Tag)
	case contextItemMCPStatus:
		return "MCP (Not Present)"
	case contextItemTools:
		return "Tool Definitions"
	case contextItemCheckpoint:
		return "Compacted Checkpoint"
	case contextItemHistory:
		return "Active History Records"
	default:
		return "Model Input Snapshot"
	}
}

func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{""}
	}
	if lipgloss.Width(text) <= maxWidth {
		return []string{text}
	}
	wrapped := lipgloss.NewStyle().Width(maxWidth).Render(text)
	return strings.Split(wrapped, "\n")
}

func formatPersistedUserRuleLines(rules string, width int) []string {
	var lines []string
	rawLines := strings.Split(rules, "\n")
	for index, rawLine := range rawLines {
		if isRuleTagLine(rawLine) {
			lines = append(lines, renderRuleTag(rawLine))
			if isClosingRuleTag(rawLine) && index < len(rawLines)-1 {
				lines = append(lines, "")
			}
			continue
		}
		lines = append(lines, wrapText("  "+rawLine, width)...)
	}
	return lines
}

func isRuleTagLine(line string) bool {
	tag := strings.TrimSpace(line)
	return (strings.HasPrefix(tag, "<RULE[") || strings.HasPrefix(tag, "</RULE[")) && strings.HasSuffix(tag, "]>")
}

func isClosingRuleTag(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "</RULE[")
}

func renderRuleTag(line string) string {
	tag := strings.TrimSpace(line)
	return lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("  " + tag)
}

func formatNumber(n int) string {
	in := fmt.Sprintf("%d", n)
	var out []byte
	l := len(in)
	for i := 0; i < l; i++ {
		if (l-i)%3 == 0 && i > 0 {
			out = append(out, ',')
		}
		out = append(out, in[i])
	}
	return string(out)
}

func targetModelLabel(model string) string {
	if model == "" {
		return "unknown"
	}
	return model
}

// GetContextInspectorContent returns the full text of the current context inspector for clipboard copy
func (m Model) GetContextInspectorContent(payload core.AgentContextPayload) string {
	if m.isContextRawMode {
		wireJSON, err := m.serializeContextRaw(payload)
		if err != nil {
			return err.Error()
		}
		return wireJSON
	}

	lines := m.buildRefinedInspectorLines(payload, 120)
	return strings.Join(lines, "\n")
}
