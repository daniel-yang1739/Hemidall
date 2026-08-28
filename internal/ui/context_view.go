package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"heimdall/internal/core"
)

// Subcategory identifiers in Context Tree (0..11)
const (
	SubcatAll       = core.SubcatAll // 0
	SubcatIdentity  = core.SubcatIdentity
	SubcatAgentsMD  = core.SubcatAgentsMD
	SubcatRuntime   = core.SubcatRuntime
	SubcatTools     = core.SubcatTools
	SubcatSkills    = core.SubcatSkills
	SubcatMCP       = core.SubcatMCP
	SubcatAnchor    = core.SubcatAnchor
	SubcatCompacted = core.SubcatCompacted
	SubcatTurns     = core.SubcatTurns
	SubcatPrompt    = core.SubcatPrompt
	SubcatBuffers   = core.SubcatBuffers
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

	payload := core.ExtractAgentContextPayload(m.history, m.sessionID, "Gemini 3.7 Flash")
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

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(" CONTEXT PAYLOAD TREE ")
	target := lipgloss.NewStyle().Foreground(ColorLightText).Render(" [Target: " + payload.TargetModel + "]")

	var treeLines []string
	treeLines = append(treeLines, title)
	treeLines = append(treeLines, target)
	treeLines = append(treeLines, "")

	// Root Level: FULL OUTBOUND PAYLOAD (ALL)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatAll, "• FULL OUTBOUND PAYLOAD (ALL)"))
	treeLines = append(treeLines, "")

	// Section 1: SYSTEM & RULES
	sec1Header := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("SYSTEM & RULES (3.8k | 2%)")
	treeLines = append(treeLines, sec1Header)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatIdentity, "  - Identity & System"))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatAgentsMD, "  - AGENTS.md Rules"))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatRuntime, "  - Runtime Metadata"))
	treeLines = append(treeLines, "")

	// Section 2: TOOLS & SCHEMAS
	sec2Header := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("TOOLS & SCHEMAS (1.4k | 1%)")
	treeLines = append(treeLines, sec2Header)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatTools, fmt.Sprintf("  - Native Tools (%d)", len(payload.NativeTools))))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatSkills, fmt.Sprintf("  - Active Skills (%d)", len(payload.ActiveSkills))))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatMCP, fmt.Sprintf("  - MCP Servers (%d)", len(payload.MCPServers))))
	treeLines = append(treeLines, "")

	// Section 3: CONTEXT HIST
	sec3Header := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("CONTEXT HIST (158k | 86%)")
	treeLines = append(treeLines, sec3Header)
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatAnchor, "  - Checkpoint Anchor"))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatCompacted, fmt.Sprintf("  - %s Compacted Steps", formatNumber(payload.CompactedStepsCount))))
	treeLines = append(treeLines, m.renderTreeLeaf(SubcatTurns, "  - Active Turns (146k)"))
	treeLines = append(treeLines, "")

	// Section 4: ACTIVE INBOUND
	sec4Header := lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("ACTIVE INBOUND (21k | 11%)")
	treeLines = append(treeLines, sec4Header)
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
		modeBadge = "[MODE: RAW WIRE [r]]"
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

	var bodyLines []string
	if m.isContextRawMode {
		bodyLines = m.buildRawWireLines(payload, innerWidth)
	} else {
		bodyLines = m.buildRefinedInspectorLines(payload, innerWidth)
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

func (m Model) getInspectorHeaderTitle(payload core.AgentContextPayload) string {
	switch m.contextSubItemIndex {
	case SubcatAll:
		return "FULL OUTBOUND PAYLOAD (185.2k Tok)"
	case SubcatIdentity:
		return "IDENTITY & SYSTEM INSTRUCTIONS (~1,200 Tok)"
	case SubcatAgentsMD:
		return "AGENTS.MD CONSTITUTION RULES (~2,600 Tok)"
	case SubcatRuntime:
		return "RUNTIME ENVIRONMENT METADATA (~280 Tok)"
	case SubcatTools:
		return fmt.Sprintf("NATIVE HARNESS TOOLS SCHEMAS (%d Tools | ~1,120 Tok)", len(payload.NativeTools))
	case SubcatSkills:
		return fmt.Sprintf("INSTALLED AGENT SKILLS (%d Skills | ~4,200 Tok)", len(payload.ActiveSkills))
	case SubcatMCP:
		return "MODEL CONTEXT PROTOCOL (MCP) SERVERS (0 Servers)"
	case SubcatAnchor:
		return "TRUNCATION CHECKPOINT BASE ANCHOR (12.5k Tok)"
	case SubcatCompacted:
		return fmt.Sprintf("%s COMPACTED CONVERSATION STEPS", formatNumber(payload.CompactedStepsCount))
	case SubcatTurns:
		return "ACTIVE CONVERSATION TURNS (5 Active Turns | 146k Tok)"
	case SubcatPrompt:
		return "LATEST INBOUND USER PROMPT (~340 Tok)"
	case SubcatBuffers:
		return "STAGED LOCAL EXECUTION BUFFERS (110 Tok)"
	default:
		return "CONTEXT PAYLOAD INSPECTOR"
	}
}

func (m Model) buildRefinedInspectorLines(payload core.AgentContextPayload, width int) []string {
	var lines []string

	switch m.contextSubItemIndex {
	case SubcatAll:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🌐 COMPREHENSIVE OUTBOUND PAYLOAD OVERVIEW:"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  • Target API Endpoint: Google Gemini GenerateContentRequest"))
		lines = append(lines, fmt.Sprintf("  • Target Architecture: %s (Context Limit: %s Tok)", payload.TargetModel, formatNumber(payload.ContextLimit)))
		lines = append(lines, fmt.Sprintf("  • Total Outbound Tokens: %s Tok (%.1f%% of limit)", formatNumber(payload.TotalTokens), float64(payload.TotalTokens)/float64(payload.ContextLimit)*100.0))
		lines = append(lines, fmt.Sprintf("  • Projected KV Cache Hit: %.1f%% (~%s cached, ~%s new)", payload.ProjectedHitRate, formatNumber(payload.ProjectedCached), formatNumber(payload.ProjectedNew)))
		lines = append(lines, fmt.Sprintf("  • Estimated Cloud Turn Cost: $%.4f USD (~NT$ %.2f TWD)", payload.ProjectedCostUSD, payload.ProjectedCostTWD))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render("📦 4 Core Payload Segments:"))
		lines = append(lines, "  1. SYSTEM & RULES :   3,806 Tok ( 2.1%)  [Identity, AGENTS.md Constitution, Runtime]")
		lines = append(lines, fmt.Sprintf("  2. TOOLS & SCHEMAS:   1,377 Tok ( 0.7%%)  [%d Native Tools, %d Active Skills, %d MCP]", len(payload.NativeTools), len(payload.ActiveSkills), len(payload.MCPServers)))
		lines = append(lines, fmt.Sprintf("  3. CONTEXT HISTORY: 158,537 Tok (85.6%%)  [Base Anchor, %s Compacted, 5 Active Turns]", formatNumber(payload.CompactedStepsCount)))
		lines = append(lines, "  4. ACTIVE INBOUND :  21,480 Tok (11.6%)  [User Prompt + Staged Tool Buffers]")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorHighlight).Render("💡 Action Hints:"))
		lines = append(lines, "  • Press [r] to view the COMPLETE official Gemini JSON Wire Request.")
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
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorLightText).Render("📌 Static Prefix Invariant: KV Cache locked at Step 0, shared across 100% of turns."))

	case SubcatAgentsMD:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("📜 REPOSITORY CONSTITUTION (Immutable Laws):"))
		lines = append(lines, "")
		for _, rawL := range strings.Split(payload.ConstitutionDoc, "\n") {
			for _, wL := range wrapText("  "+rawL, width) {
				lines = append(lines, wL)
			}
		}

	case SubcatRuntime:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("⚙️ RUNTIME HOST & ENVIRONMENT METADATA:"))
		lines = append(lines, "")
		for k, v := range payload.RuntimeMetadata {
			for _, wL := range wrapText(fmt.Sprintf("  • %-12s: %s", k, v), width) {
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
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(fmt.Sprintf("🧠 INSTALLED AGENT SKILLS & CAPABILITIES (%d Skills):", len(payload.ActiveSkills))))
		lines = append(lines, "")
		for _, skill := range payload.ActiveSkills {
			statusTag := lipgloss.NewStyle().Foreground(ColorSuccess).Render("[" + skill.Status + "]")
			lines = append(lines, fmt.Sprintf("• %s %s", lipgloss.NewStyle().Bold(true).Render(skill.Name), statusTag))
			for _, wL := range wrapText("  └ "+skill.Description, width) {
				lines = append(lines, wL)
			}
			for _, wL := range wrapText("  └ Guidelines: "+skill.Guidelines, width) {
				lines = append(lines, wL)
			}
			lines = append(lines, "")
		}

	case SubcatMCP:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🔌 MODEL CONTEXT PROTOCOL (MCP) INTEGRATIONS:"))
		lines = append(lines, "")
		lines = append(lines, "  No external MCP servers currently active.")
		lines = append(lines, "  All tool calls serviced natively by built-in Antigravity harness.")

	case SubcatAnchor:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("⚓ COMPACTION RE-ANCHORING BASELINE:"))
		lines = append(lines, "")
		for _, l := range strings.Split(payload.CheckpointSummary, "\n") {
			for _, wL := range wrapText("  "+l, width) {
				lines = append(lines, wL)
			}
		}

	case SubcatCompacted:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("📦 SLICED & COMPACTED HISTORY WINDOW:"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  • %d Historical Steps truncated from active GPU window.", payload.CompactedStepsCount))
		lines = append(lines, "  • Retained structured summary preserves critical decisions and test constraints.")
		lines = append(lines, "  • Prevents context window explosion while maintaining 88.4% cache hit rates.")

	case SubcatTurns:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("🔄 ACTIVE CONVERSATION TURNS (Sliding Window):"))
		lines = append(lines, "")
		for _, t := range payload.ActiveHistoryTurns {
			role := "👤 User"
			if t.IsCloudStep() {
				role = "🤖 Model"
			} else if t.IsLocalStep() {
				role = "💻 Local"
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
	wireJSON, err := core.SerializeSubcategoryRaw(payload, m.contextSubItemIndex)
	if err != nil {
		return []string{"Error serializing wire payload: " + err.Error()}
	}

	subcatName := m.getSubcategoryName(m.contextSubItemIndex)
	var formattedLines []string
	formattedLines = append(formattedLines, lipgloss.NewStyle().Foreground(ColorLightText).Render(fmt.Sprintf("// Google Gemini Wire Payload Part: %s", subcatName)))
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

func (m Model) getSubcategoryName(subcat int) string {
	switch subcat {
	case SubcatAll:
		return "FULL OUTBOUND PAYLOAD (ALL)"
	case SubcatIdentity:
		return "Identity & System"
	case SubcatAgentsMD:
		return "AGENTS.md Rules"
	case SubcatRuntime:
		return "Runtime Metadata"
	case SubcatTools:
		return "Native Tools (8)"
	case SubcatSkills:
		return "Active Skills (3)"
	case SubcatMCP:
		return "MCP Servers (0)"
	case SubcatAnchor:
		return "Checkpoint Anchor"
	case SubcatCompacted:
		return "Compacted Steps"
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

// GetContextInspectorContent returns the full text of the current context inspector for clipboard copy
func (m Model) GetContextInspectorContent(payload core.AgentContextPayload) string {
	if m.isContextRawMode {
		wireJSON, err := core.SerializeSubcategoryRaw(payload, m.contextSubItemIndex)
		if err != nil {
			return err.Error()
		}
		return wireJSON
	}

	lines := m.buildRefinedInspectorLines(payload, 120)
	return strings.Join(lines, "\n")
}
