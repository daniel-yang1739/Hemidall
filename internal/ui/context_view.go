package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"heimdall/internal/core"
)

// Subcategory identifiers in Context Tree (0..11)
const (
	SubcatAll            = core.SubcatAll // 0
	SubcatIdentity       = core.SubcatIdentity
	SubcatAgentsMD       = core.SubcatAgentsMD
	SubcatRuntime        = core.SubcatRuntime
	SubcatTools          = core.SubcatTools
	SubcatSkills         = core.SubcatSkills
	SubcatMCP            = core.SubcatMCP
	SubcatAnchor         = core.SubcatAnchor
	SubcatTurns          = core.SubcatTurns
	SubcatCurrentHistory = core.SubcatCurrentHistory
	SubcatPrompt         = core.SubcatPrompt
	SubcatBuffers        = core.SubcatBuffers
	SubcatLast           = core.SubcatLast
)

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

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(" CONTEXT EVIDENCE TREE ")
	totalContext := lipgloss.NewStyle().Foreground(ColorHighlight).Render(fmt.Sprintf(" [Total: %s Tok]", formatNumber(payload.TotalTokens)))

	var treeLines []string
	systemTokens, toolsTokens, historyTokens, inboundTokens := m.contextTreeTokenCounts(payload)
	treeLines = append(treeLines, title)
	treeLines = append(treeLines, totalContext)
	treeLines = append(treeLines, "")

	// Root Level: FULL OUTBOUND PAYLOAD (ALL)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatAll, "• CONTEXT EVIDENCE"))
	treeLines = append(treeLines, "")

	// Section 1: SYSTEM & RULES
	treeLines = append(treeLines, renderContextSectionHeader("SYSTEM & RULES", systemTokens, innerWidth)...)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatIdentity, "  - Identity Subsection"))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatAgentsMD, "  - Persisted User Rules"))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatRuntime, "  - Heimdall Runtime Metadata"))
	treeLines = append(treeLines, "")

	// Section 2: TOOLS & SCHEMAS
	treeLines = append(treeLines, renderContextSectionHeader("CAPABILITIES", toolsTokens, innerWidth)...)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatTools, fmt.Sprintf("  - Tool Definitions (%d)", len(payload.NativeTools))))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatSkills, "  - Skills Prompt Section"))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatMCP, "  - MCP Attribution"))
	treeLines = append(treeLines, "")

	// Section 3: CONTEXT HIST
	treeLines = append(treeLines, renderContextSectionHeader("TRANSCRIPT HISTORY", historyTokens, innerWidth)...)
	if payload.SnapshotAvailable {
		if compactedCheckpoint(payload) != nil {
			treeLines = append(treeLines, m.renderTreeLeaf(SubcatAnchor, "  - Compacted Checkpoint"))
		}
		treeLines = append(treeLines, m.renderTreeLeaf(SubcatCurrentHistory, fmt.Sprintf("  - Active History (%d)", len(currentHistoryRecords(payload)))))
		treeLines = append(treeLines, m.renderTreeLeaf(SubcatTurns, fmt.Sprintf("  - Active Turns (%d)", len(payload.ActiveHistoryTurns))))
	} else {
		treeLines = append(treeLines, m.renderTreeLeaf(SubcatAnchor, "  - Checkpoint Anchor"))
		treeLines = append(treeLines, m.renderTreeLeaf(SubcatTurns, fmt.Sprintf("  - Active Turns (%d)", len(payload.ActiveHistoryTurns))))
	}
	treeLines = append(treeLines, "")
	treeLines = append(treeLines, renderContextSectionHeader("LATEST TRANSCRIPT INPUT", inboundTokens, innerWidth)...)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatPrompt, "  - Inbound Prompt"))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatBuffers, "  - Staged Tool Buffers"))

	for i, tl := range treeLines {
		if lipgloss.Width(tl) > innerWidth {
			treeLines[i] = lipgloss.NewStyle().MaxWidth(innerWidth).Render(tl)
		}
	}

	content := strings.Join(treeLines, "\n")
	return boxStyle.Render(content)
}

func (m Model) contextTreeTokenCounts(payload core.AgentContextPayload) (int, int, int, int) {
	if m.contextEstimateHistoryCount == len(m.history) && m.contextEstimate.Available {
		return m.contextEstimate.SystemTokens, m.contextEstimate.ToolsTokens, m.contextEstimate.HistoryTokens, m.contextEstimate.InboundTokens + m.contextEstimate.ToolBufferTokens
	}
	return 0, 0, 0, 0
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
	count := len(currentHistoryRecords(payload))
	if compactedCheckpoint(payload) != nil {
		count++
	}
	return count
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
	if compactedCheckpoint(payload) != nil {
		maxRows--
	}
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
		line := ""
		isCompactedCheckpoint := offset == len(records)
		if offset == len(records) {
			line = fmt.Sprintf("  #%-*d  %-*s  CHECKPOINT", positionWidth, 1, referenceWidth, "")
		} else {
			r := records[len(records)-1-offset]
			sizeValue, sizeUnit := formatContextRecordSize(r.ByteSize)
			line = fmt.Sprintf("  #%-*d  %-*s  %-*s %s", positionWidth, r.Position, referenceWidth, formatObservedEventReference(r.ObservedSequence), sizeWidth, sizeValue, sizeUnit)
		}
		if offset == selectedOffset {
			line = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Background(ColorDarkBg).Render("> " + strings.TrimSpace(line))
		}
		if isCompactedCheckpoint {
			line = lipgloss.NewStyle().Foreground(ColorMuted).Render(line)
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
		modeBadge = "[MODE: EVIDENCE JSON [r]]"
		if payload.SnapshotAvailable {
			modeBadge = "[MODE: SNAPSHOT + TRANSCRIPT [r]]"
		}
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
	switch m.contextSubItemIndex {
	case SubcatAll:
		return fmt.Sprintf("CONTEXT EVIDENCE: %s", payload.SourceKind)
	case SubcatIdentity:
		return fmt.Sprintf("IDENTITY SUBSECTION (%s Tok)", formatCompactNumber(core.CountTokens(payload.IdentityPrompt)))
	case SubcatAgentsMD:
		return fmt.Sprintf("PERSISTED USER RULES (%s Tok)", formatCompactNumber(core.CountTokens(payload.ConstitutionDoc)))
	case SubcatRuntime:
		return "RUNTIME ENVIRONMENT METADATA"
	case SubcatTools:
		return fmt.Sprintf("PERSISTED TOOL DEFINITIONS (%d Tools)", len(payload.NativeTools))
	case SubcatSkills:
		return "PERSISTED SKILLS PROMPT SECTION"
	case SubcatMCP:
		return "MCP ATTRIBUTION STATUS"
	case SubcatAnchor:
		if payload.SnapshotAvailable && compactedCheckpoint(payload) != nil {
			return "COMPACTED CHECKPOINT · LATEST PERSISTED SNAPSHOT"
		}
		return fmt.Sprintf("TRANSCRIPT CHECKPOINT ANCHOR · STEP %s", formatNumber(payload.CheckpointStepIndex))
	case SubcatCurrentHistory:
		records := currentHistoryRecords(payload)
		if m.contextHistoryList && compactedCheckpoint(payload) != nil && m.contextHistoryIndex >= len(records) {
			return "COMPACTED CHECKPOINT · LATEST PERSISTED SNAPSHOT"
		}
		if m.contextHistoryList && len(records) > 0 {
			index := len(records) - 1 - m.contextHistoryIndex
			if index < 0 {
				index = 0
			}
			return fmt.Sprintf("RECORD #%d · LATEST PERSISTED SNAPSHOT", records[index].Position)
		}
		return fmt.Sprintf("ACTIVE HISTORY (%d Records)", len(records))
	case SubcatTurns:
		return fmt.Sprintf("ACTIVE CONVERSATION TURNS (%d Turns)", len(payload.ActiveHistoryTurns))
	case SubcatPrompt:
		return fmt.Sprintf("LATEST INBOUND USER PROMPT (%s Tok)", formatCompactNumber(core.CountTokens(payload.LatestPrompt)))
	case SubcatBuffers:
		return fmt.Sprintf("STAGED LOCAL EXECUTION BUFFERS (%s Tok)", formatCompactNumber(core.CountTokens(payload.StagedBuffers)))
	default:
		return "CONTEXT PAYLOAD INSPECTOR"
	}
}

func (m Model) buildRefinedInspectorLines(payload core.AgentContextPayload, width int) []string {
	var lines []string

	switch m.contextSubItemIndex {
	case SubcatAll:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🌐 ANTIGRAVITY CONTEXT OBSERVATION:"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  • Evidence class: %s", payload.SourceKind))
		if payload.SnapshotAvailable {
			lines = append(lines, fmt.Sprintf("  • Source database: %s", payload.SourcePath))
			lines = append(lines, fmt.Sprintf("  • Snapshot record: gen_metadata.idx=%d (%s bytes)", payload.SnapshotGenIndex, formatNumber(payload.SnapshotBytes)))
			lines = append(lines, "  • This page combines snapshot sections with transcript observations and Heimdall runtime metadata.")
		} else {
			lines = append(lines, "  • Persisted context snapshot: unavailable; using transcript observations.")
		}
		lines = append(lines, fmt.Sprintf("  • Observed model: %s", targetModelLabel(payload.TargetModel)))
		if payload.TotalTokens > 0 && payload.ContextLimit > 0 {
			usagePercent := float64(payload.TotalTokens) / float64(payload.ContextLimit) * 100.0
			lines = append(lines, fmt.Sprintf("  • Latest persisted context: %s Tok (%.1f%% of observed %s Tok limit)", formatNumber(payload.TotalTokens), usagePercent, formatNumber(payload.ContextLimit)))
		} else if payload.TotalTokens > 0 {
			lines = append(lines, fmt.Sprintf("  • Latest persisted context: %s Tok (limit unavailable)", formatNumber(payload.TotalTokens)))
		} else {
			lines = append(lines, "  • Latest persisted context: unavailable")
		}
		lines = append(lines, "  • Cache and cost projection: unavailable without a captured provider response.")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("📦 Persisted Snapshot Segments:"))
		lines = append(lines, fmt.Sprintf("  1. SYSTEM PROMPT   : %s Tok [contains identity, rules, skills and host instructions]", formatNumber(core.CountTokens(payload.SystemPrompt))))
		lines = append(lines, fmt.Sprintf("  2. TOOL DEFINITIONS: %d repeated protobuf entries", len(payload.NativeTools)))
		lines = append(lines, fmt.Sprintf("  3. FIELD 2 COUNT  : %d repeated protobuf field-2 occurrences", payload.PersistedContextEntryCount))
		lines = append(lines, "  4. MCP ATTRIBUTION  : unknown without the official protobuf schema")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorHighlight).Render("💡 Action Hints:"))
		lines = append(lines, "  • Press [r] to view Heimdall's evidence JSON with per-field source labels.")
		lines = append(lines, "  • This is not an HTTP body capture or an official GenerateContentRequest serialization.")
		lines = append(lines, "  • Press [y] or [c] to copy the displayed payload directly to clipboard.")
		lines = append(lines, "  • Use [j/k] to navigate to specific subcategories for isolated inspection.")

	case SubcatIdentity:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🎭 AGENT IDENTITY & PAIR-PROGRAMMING DIRECTIVE:"))
		lines = append(lines, "")
		for _, rawL := range strings.Split(payload.IdentityPrompt, "\n") {
			for _, wL := range wrapText("  "+rawL, width) {
				lines = append(lines, wL)
			}
		}
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorLightText).Render("📌 Identity is one subsection of the persisted system prompt; cache reuse is not proven by this snapshot."))

	case SubcatAgentsMD:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("📜 PERSISTED USER RULES (Formatted for Reading):"))
		lines = append(lines, "")
		lines = append(lines, formatPersistedUserRuleLines(payload.ConstitutionDoc, width)...)

	case SubcatRuntime:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("⚙️ RUNTIME HOST & ENVIRONMENT METADATA:"))
		lines = append(lines, "")
		for _, key := range core.SortedRuntimeMetadataKeys(payload.RuntimeMetadata) {
			for _, wL := range wrapText(fmt.Sprintf("  • %-12s: %s", key, payload.RuntimeMetadata[key]), width) {
				lines = append(lines, wL)
			}
		}

	case SubcatTools:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(fmt.Sprintf("🛠️ TOOL SIGNATURES & DESCRIPTIONS (%d Registered Tools):", len(payload.NativeTools))))
		lines = append(lines, "")
		for _, tool := range payload.NativeTools {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("• "+tool.Signature))
			for _, wL := range wrapText("  └ "+tool.Description, width) {
				lines = append(lines, lipgloss.NewStyle().Foreground(ColorLightText).Render(wL))
			}
			lines = append(lines, "")
		}

	case SubcatSkills:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🧠 SKILLS SECTION EMBEDDED IN THE SYSTEM PROMPT:"))
		lines = append(lines, "")
		for _, rawLine := range strings.Split(payload.SkillsSection, "\n") {
			lines = append(lines, wrapText("  "+rawLine, width)...)
		}

	case SubcatMCP:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🔌 MODEL CONTEXT PROTOCOL (MCP) INTEGRATIONS:"))
		lines = append(lines, "")
		lines = append(lines, "  The persisted snapshot contains tool definitions and MCP-related text,")
		lines = append(lines, "  but the reverse-engineered wire paths cannot yet attribute each tool")
		lines = append(lines, "  to native harness code or a specific MCP server.")
		lines = append(lines, "  Status: UNKNOWN, not NONE.")

	case SubcatAnchor:
		if checkpoint := compactedCheckpoint(payload); checkpoint != nil {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("📦 COMPACTED CHECKPOINT:"), "")
			for _, l := range strings.Split(checkpoint.PrimaryText, "\n") {
				lines = append(lines, wrapText("  "+l, width)...)
			}
			lines = append(lines, "", "  • Record #1 is separated from Active History because it is a compacted summary.")
			break
		}
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("⚓ TRANSCRIPT CHECKPOINT ANCHOR:"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  • Observed at transcript step #%d.", payload.CheckpointStepIndex))
		lines = append(lines, "  • The text below is the checkpoint event's captured summary.")
		lines = append(lines, "  • This does not prove the remote context window's exact truncation or cache state.")
		lines = append(lines, "")
		for _, l := range strings.Split(payload.CheckpointSummary, "\n") {
			for _, wL := range wrapText("  "+l, width) {
				lines = append(lines, wL)
			}
		}

	case SubcatCurrentHistory:
		records := currentHistoryRecords(payload)
		if !m.contextHistoryList {
			lines = append(lines, "  • Press [Enter] to browse active events from the latest persisted snapshot.")
			break
		}
		if len(records) == 0 {
			lines = append(lines, "  • No decoded persisted records are available.")
			break
		}
		if checkpoint := compactedCheckpoint(payload); checkpoint != nil && m.contextHistoryIndex >= len(records) {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("📦 COMPACTED CHECKPOINT:"), "")
			for _, l := range strings.Split(checkpoint.PrimaryText, "\n") {
				lines = append(lines, wrapText("  "+l, width)...)
			}
			lines = append(lines, "", "  • This is record #1 in the latest persisted snapshot.")
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

	case SubcatTurns:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🔄 ACTIVE CONVERSATION TURNS (Sliding Window):"))
		lines = append(lines, "")
		for _, t := range payload.ActiveHistoryTurns {
			role := "👤 USER"
			if t.IsCloudStep() {
				role = "🤖 ASSISTANT"
			} else if t.IsLocalStep() {
				role = "🛠️ TOOL_RESULT"
			} else if t.IsCompactionStep() {
				role = "⚙️ SYSTEM"
			}
			summary := t.Summary
			if summary == "" {
				summary = t.RawContent
				if len(summary) > 60 {
					summary = summary[:60] + "..."
				}
			}
			for _, wL := range wrapText(fmt.Sprintf("  • [Step #%d] %s: %s", t.StepIndex, role, summary), width) {
				lines = append(lines, wL)
			}
		}

	case SubcatPrompt:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("📥 LATEST INBOUND PROMPT PAYLOAD:"))
		lines = append(lines, "")
		for _, rawL := range strings.Split(payload.LatestPrompt, "\n") {
			for _, wL := range wrapText("  "+rawL, width) {
				lines = append(lines, wL)
			}
		}

	case SubcatBuffers:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("⚡ STAGED LOCAL TOOL EXECUTION BUFFERS:"))
		lines = append(lines, "")
		for _, rawL := range strings.Split(payload.StagedBuffers, "\n") {
			for _, wL := range wrapText("  "+rawL, width) {
				lines = append(lines, wL)
			}
		}
	}

	return lines
}

func (m Model) buildRawWireLines(payload core.AgentContextPayload, width int) []string {
	wireJSON, err := m.serializeContextRaw(payload)
	if err != nil {
		return []string{"Error serializing wire payload: " + err.Error()}
	}

	subcatName := m.getSubcategoryName(m.contextSubItemIndex)
	var formattedLines []string
	formattedLines = append(formattedLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(fmt.Sprintf("// Heimdall evidence view [%s]: %s", payload.SourceKind, subcatName)))
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
	if m.contextSubItemIndex != SubcatCurrentHistory {
		return core.SerializeSubcategoryRaw(payload, m.contextSubItemIndex)
	}
	if !m.contextHistoryList {
		return core.SerializePersistedActiveEventRaw(payload, 0)
	}
	records := currentHistoryRecords(payload)
	if m.contextHistoryIndex >= len(records) {
		return core.SerializePersistedActiveEventRaw(payload, 1)
	}
	index := len(records) - 1 - m.contextHistoryIndex
	if index < 0 {
		index = 0
	}
	return core.SerializePersistedActiveEventRaw(payload, records[index].Position)
}

func (m Model) getSubcategoryName(subcat int) string {
	switch subcat {
	case SubcatAll:
		return "Context Evidence"
	case SubcatIdentity:
		return "Identity Subsection"
	case SubcatAgentsMD:
		return "Persisted User Rules"
	case SubcatRuntime:
		return "Runtime Metadata"
	case SubcatTools:
		return "Persisted Tool Definitions"
	case SubcatSkills:
		return "Persisted Skills Section"
	case SubcatMCP:
		return "MCP Attribution"
	case SubcatAnchor:
		return "Checkpoint Anchor"
	case SubcatTurns:
		return "Active Turns"
	case SubcatPrompt:
		return "Inbound Prompt"
	case SubcatBuffers:
		return "Staged Tool Buffers"
	default:
		return "Context Payload"
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
