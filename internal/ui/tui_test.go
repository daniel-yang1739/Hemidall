package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"heimdall/internal/adapters/antigravity"
	"heimdall/internal/core"
)

func TestCJKAndLongPayloadZeroHeightVariation(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}, {140, 40}} {
		m := NewModel("test-session", false)
		m.width = size.w
		m.height = size.h

		// Create steps with heavy Traditional Chinese text, emojis, and long payloads
		for i := 0; i < 25; i++ {
			chineseSummary := "【實測連線】這是一個非常長的繁體中文步驟說明文字，包含特殊符號與長度測試"
			m.history = append(m.history, core.UnifiedAgentEvent{
				StepIndex:  i,
				Type:       core.StepTypeModelResponse,
				Summary:    chineseSummary,
				Timestamp:  time.Now(),
				RawContent: strings.Repeat("繁體中文長文本內容一行接一行，測試 Overflow: hidden 絕對不換行也不長高。\n", 50),
			})
		}
		m.latestEvent = m.history[len(m.history)-1]
		m.selectedIdx = 0

		m.activeView = ViewHistory
		fullView := m.View()
		lines := strings.Split(fullView, "\n")

		if len(lines) != size.h {
			t.Fatalf("[%dx%d] Expected exactly %d lines, got %d", size.w, size.h, size.h, len(lines))
		}

		// Verify bottom border is strictly on line h-2
		bottomBorderLine := lines[size.h-2]
		if !strings.Contains(bottomBorderLine, "╰") || !strings.Contains(bottomBorderLine, "╯") {
			t.Errorf("[%dx%d] Expected bottom border on line %d, got: %s", size.w, size.h, size.h-2, bottomBorderLine)
		}

		// Verify footer shortcuts are strictly on line h-1
		footerLine := lines[size.h-1]
		if !strings.Contains(footerLine, "Shortcuts") && !strings.Contains(footerLine, "Quit") && !strings.Contains(footerLine, "Docs") {
			t.Errorf("[%dx%d] Expected footer on line %d, got: %s", size.w, size.h, size.h-1, footerLine)
		}
	}
}

func TestShortcutsFloatModalToggle(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30

	// 1. Press '?' to open shortcuts modal
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)
	if !m.isShortcutsModalOpen {
		t.Fatal("Expected isShortcutsModalOpen=true after pressing '?'")
	}

	view := m.View()
	if !strings.Contains(view, "KEYBOARD SHORTCUTS") {
		t.Errorf("Expected 'KEYBOARD SHORTCUTS' in view, got: %s", view)
	}

	// 2. Press 'Esc' or '?' to close modal
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.isShortcutsModalOpen {
		t.Error("Expected isShortcutsModalOpen=false after pressing Esc")
	}
}

func TestView3DocsPageRenderingAndSearch(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30

	// 1. Press '4' or 'i' to switch to View 4 Docs (Default: English)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = updated.(Model)
	if m.activeView != ViewDocs {
		t.Fatalf("Expected activeView=ViewDocs, got %v", m.activeView)
	}

	view := m.View()
	if !strings.Contains(view, "ARCHITECTURE & CONTEXT DEFINITIONS") {
		t.Errorf("Expected 'ARCHITECTURE & CONTEXT DEFINITIONS' in view, got: %s", view)
	}

	// 2. Press 't' to toggle to Traditional Chinese
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = updated.(Model)
	if m.docsLang != "zh" {
		t.Errorf("Expected docsLang='zh' after pressing 't', got '%s'", m.docsLang)
	}
	zhView := m.View()
	if !strings.Contains(zhView, "架構名詞釋義與上下文辭典") {
		t.Errorf("Expected Chinese title in zhView, got: %s", zhView)
	}

	// 3. Press 't' again to toggle back to English
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = updated.(Model)
	if m.docsLang != "en" {
		t.Errorf("Expected docsLang='en' after pressing 't' again, got '%s'", m.docsLang)
	}

	// 4. Press '/' to activate search mode and search for persisted snapshots.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	if !m.isDocsSearching {
		t.Error("Expected isDocsSearching=true after pressing '/'")
	}

	m = typeDocsQuery(m, "snapshot")
	snapshotFilteredView := m.View()
	requireViewContains(t, snapshotFilteredView, "Persisted context snapshot")

	// 5. Press Esc to clear search, then search for the local transcript estimate.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	m = typeDocsQuery(m, "local transcript")
	if m.docsSearchQuery != "local transcript" {
		t.Errorf("Expected docsSearchQuery='local transcript', got '%s'", m.docsSearchQuery)
	}

	filteredView := m.View()
	requireViewContains(t, filteredView, "Track 2 — Playback Context Evidence")

	// 6. Press Esc to clear search
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.docsSearchQuery != "" {
		t.Errorf("Expected docsSearchQuery to be cleared, got '%s'", m.docsSearchQuery)
	}

	// 7. Test jump to bottom on 'G' and overscroll prevention
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updated.(Model)
	maxScroll := m.getDocsMaxScroll()
	if m.docsScroll != maxScroll {
		t.Errorf("Expected docsScroll to be clamped to maxScroll %d, got %d", maxScroll, m.docsScroll)
	}

	// Immediate response on pressing 'k' once
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(Model)
	if m.docsScroll != maxScroll-1 {
		t.Errorf("Expected docsScroll to immediately decrement to %d on first 'k', got %d", maxScroll-1, m.docsScroll)
	}
}

func typeDocsQuery(model Model, query string) Model {
	for _, character := range query {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{character}})
		model = updated.(Model)
	}
	return model
}

func requireViewContains(t *testing.T, view, expected string) {
	t.Helper()
	if !strings.Contains(view, expected) {
		t.Fatalf("expected view to contain %q, got: %s", expected, view)
	}
}

func TestView2ContextPageRenderingAndDualModeToggle(t *testing.T) {
	m := NewModel("test-session", false)
	m.history = []core.UnifiedAgentEvent{{
		SessionID: "test-session",
		Type:      core.StepTypeToolCall,
		ToolCalls: []core.ToolCallInfo{{
			ToolName:  "write_to_file",
			Arguments: map[string]interface{}{"TargetFile": "main.go"},
		}},
	}}
	m.width = 100
	m.height = 30

	// 1. Press '3' or 'c' to switch to [3] Context View
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = updated.(Model)
	if m.activeView != ViewContext {
		t.Fatalf("Expected activeView=ViewContext, got %v", m.activeView)
	}

	view := m.View()
	if !strings.Contains(view, "CONTEXT EVIDENCE TREE") {
		t.Fatalf("Expected 'CONTEXT EVIDENCE TREE' in context view, got: %s", view)
	}
	if !strings.Contains(view, "[MODE: REFINED [r]]") {
		t.Errorf("Expected default mode '[MODE: REFINED [r]]' in view, got: %s", view)
	}

	// 2. Press 'r' to toggle decoded evidence JSON mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = updated.(Model)
	if !m.isContextRawMode {
		t.Fatal("Expected isContextRawMode=true after pressing 'r'")
	}

	rawView := m.View()
	if !strings.Contains(rawView, "[MODE: EVIDENCE JSON [r]]") {
		t.Errorf("Expected evidence JSON mode after 'r' toggle, got: %s", rawView)
	}
	if !strings.Contains(rawView, "evidence") && !strings.Contains(rawView, "transcript_observations") {
		t.Errorf("Expected evidence representation in rawView, got: %s", rawView)
	}

	// 3. Navigate down to Native Tools (SubcatTools = 4) and check specific raw tools schema
	m.contextSubItemIndex = SubcatTools
	toolsRawView := m.View()
	if !strings.Contains(toolsRawView, "decoded_tool_entries") || !strings.Contains(toolsRawView, "write_to_file") {
		t.Errorf("Expected decoded tool entries in toolsRawView, got: %s", toolsRawView)
	}

	// 4. Press 'r' again to return to Refined Cards mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = updated.(Model)
	if m.isContextRawMode {
		t.Fatal("Expected isContextRawMode=false after second 'r'")
	}
}

func TestFormatPersistedUserRuleLines_SeparatesAndPreservesRuleSources(t *testing.T) {
	rules := `<RULE[user_global]>
Global rule text
</RULE[user_global]>
<RULE[/workspace/AGENTS.md]>
Project rule text
</RULE[/workspace/AGENTS.md]>`

	lines := formatPersistedUserRuleLines(rules, 120)
	formatted := strings.Join(lines, "\n")
	if !strings.Contains(formatted, "<RULE[user_global]>") || !strings.Contains(formatted, "<RULE[/workspace/AGENTS.md]>") {
		t.Fatalf("expected both original rule tags to be preserved: %q", formatted)
	}
	if !strings.Contains(formatted, "</RULE[user_global]>\n\n  <RULE[/workspace/AGENTS.md]>") {
		t.Errorf("expected a blank line between rule blocks: %q", formatted)
	}
	if !strings.Contains(formatted, "  Global rule text") || !strings.Contains(formatted, "  Project rule text") {
		t.Errorf("expected rule text to remain unchanged except for display indentation: %q", formatted)
	}
}

func TestCheckpointAnchorInspector_CombinesSummaryAndTranscriptLocation(t *testing.T) {
	m := NewModel("test-session", false)
	m.contextSubItemIndex = SubcatAnchor
	payload := core.AgentContextPayload{
		CheckpointStepIndex: 2255,
		CheckpointSummary:   "<CONTEXT_SUMMARY>Saved conversation summary</CONTEXT_SUMMARY>",
	}

	inspector := strings.Join(m.buildRefinedInspectorLines(payload, 120), "\n")
	if !strings.Contains(inspector, "Observed at transcript step #2255.") {
		t.Errorf("expected transcript location in combined checkpoint inspector: %q", inspector)
	}
	if !strings.Contains(inspector, "Saved conversation summary") {
		t.Errorf("expected checkpoint summary in combined checkpoint inspector: %q", inspector)
	}
	if strings.Contains(inspector, "SLICED & COMPACTED HISTORY WINDOW") {
		t.Errorf("expected obsolete standalone compaction section to be absent: %q", inspector)
	}
}

func TestContextTree_SeparatesSectionTokensAndShowsTotal(t *testing.T) {
	sectionLines := renderContextSectionHeader("TRANSCRIPT HISTORY", 123456, 20)
	if len(sectionLines) != 2 {
		t.Fatalf("expected a narrow section header to use two lines, got %d", len(sectionLines))
	}
	if !strings.Contains(sectionLines[1], "123.5k Tok") {
		t.Errorf("expected compact token label on the second line: %q", sectionLines[1])
	}

	m := NewModel("test-session", false)
	payload := core.AgentContextPayload{TotalTokens: 123456, CheckpointStepIndex: 2255}
	tree := m.renderContextTree(payload, 38, 30)
	if !strings.Contains(tree, "Total: 123,456 Tok") {
		t.Errorf("expected latest total context tokens near the tree title: %q", tree)
	}
	if strings.Contains(tree, "Checkpoint Anchor · Step") {
		t.Errorf("expected checkpoint step to be absent from the left tree: %q", tree)
	}
	if strings.Contains(tree, "Observed model") {
		t.Errorf("expected model identity to be absent from the context tree: %q", tree)
	}
}

func TestContextTreeKeyboardNavigationJK(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewContext
	m.contextFocusPane = FocusList
	m.contextSubItemIndex = 0

	// 1. Press 'j' to navigate down through tree
	for i := 1; i <= 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(Model)
		if m.contextSubItemIndex != i {
			t.Errorf("Expected subitem index %d after %d 'j' presses, got %d", i, i, m.contextSubItemIndex)
		}
	}

	// 2. Press 'k' to navigate up
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(Model)
	if m.contextSubItemIndex != 4 {
		t.Errorf("Expected subitem index 4 after 'k', got %d", m.contextSubItemIndex)
	}
}

func TestContextVimPaneSwitchingLH(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewContext
	m.contextFocusPane = FocusList

	// 1. Press 'l' or 'Enter' to focus Inspector
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updated.(Model)
	if m.contextFocusPane != FocusDetail {
		t.Fatalf("Expected contextFocusPane=FocusDetail after 'l', got %v", m.contextFocusPane)
	}

	// 2. Press 'h' or 'Esc' to return focus to Context Tree
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = updated.(Model)
	if m.contextFocusPane != FocusList {
		t.Fatalf("Expected contextFocusPane=FocusList after 'h', got %v", m.contextFocusPane)
	}
}

func TestContextTree_NavigationMatchesRenderedHistoryOrder(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewContext
	m.contextFocusPane = FocusList
	m.contextSubItemIndex = SubcatAnchor

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)
	if m.contextSubItemIndex != SubcatCurrentHistory {
		t.Fatalf("expected Current History after Compacted Checkpoint, got subcategory %d", m.contextSubItemIndex)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)
	if m.contextSubItemIndex != SubcatTurns {
		t.Fatalf("expected Active Turns after Current History, got subcategory %d", m.contextSubItemIndex)
	}
}

func TestCurrentHistoryList_NavigationLeavesCompactedCheckpoint(t *testing.T) {
	m := NewModel("test-session", false)
	m.activeView = ViewContext
	m.contextFocusPane = FocusList
	m.contextHistoryList = true
	m.contextHistoryItemCount = 3
	m.contextHistoryIndex = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(Model)
	if m.contextHistoryIndex != 1 {
		t.Fatalf("expected up navigation to leave the last history item, got index %d", m.contextHistoryIndex)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updated.(Model)
	if m.contextHistoryIndex != 2 {
		t.Fatalf("expected G to select the last history item, got index %d", m.contextHistoryIndex)
	}
}

func TestCurrentHistoryList_CheckpointStaysOnOneMutedLine(t *testing.T) {
	m := NewModel("test-session", false)
	payload := core.AgentContextPayload{PersistedRecords: []core.PersistedContextRecord{
		{Position: 1, IsCompactedCheckpoint: true},
		{Position: 2, ByteSize: 1024},
		{Position: 3, ByteSize: 2048},
	}}
	m.contextHistoryList = true
	m.contextHistoryIndex = 2
	rendered := m.renderCurrentHistoryList(payload, 38, 30)
	if !strings.Contains(rendered, "CHECKPOINT") || strings.Contains(rendered, "CHECK\n") {
		t.Fatalf("checkpoint must remain a single line: %q", rendered)
	}
}

func TestCyclicTabAndShiftTabViewSwitching(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30

	if m.activeView != ViewDashboard {
		t.Fatalf("Initial view should be ViewDashboard, got %v", m.activeView)
	}

	// 1. Press Tab -> ViewHistory
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.activeView != ViewHistory {
		t.Fatalf("After 1st Tab, expected ViewHistory, got %v", m.activeView)
	}

	// 2. Press Tab -> ViewContext
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.activeView != ViewContext {
		t.Fatalf("After 2nd Tab, expected ViewContext, got %v", m.activeView)
	}

	// 3. Press Tab -> ViewDocs
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.activeView != ViewDocs {
		t.Fatalf("After 3rd Tab, expected ViewDocs, got %v", m.activeView)
	}

	// 4. Press Tab -> ViewDashboard (Clockwise wrap)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.activeView != ViewDashboard {
		t.Fatalf("After 4th Tab, expected ViewDashboard, got %v", m.activeView)
	}

	// 5. Press Shift+Tab -> ViewDocs (Counter-clockwise wrap)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.activeView != ViewDocs {
		t.Fatalf("After Shift+Tab, expected ViewDocs, got %v", m.activeView)
	}

	// 6. Press Shift+Tab -> ViewContext
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.activeView != ViewContext {
		t.Fatalf("After 2nd Shift+Tab, expected ViewContext, got %v", m.activeView)
	}

	// 7. Press Shift+Tab -> ViewHistory
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.activeView != ViewHistory {
		t.Fatalf("After 3rd Shift+Tab, expected ViewHistory, got %v", m.activeView)
	}
}

func TestHistoryVimPaneSwitchingHL(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusList

	// 1. Press 'l' to enter Right Pane (Inspector)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updated.(Model)
	if m.focusPane != FocusDetail {
		t.Fatalf("Expected focusPane=FocusDetail after pressing 'l', got %v", m.focusPane)
	}

	// 2. Press 'h' to return to Left Pane (Steps List)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = updated.(Model)
	if m.focusPane != FocusList {
		t.Fatalf("Expected focusPane=FocusList after pressing 'h', got %v", m.focusPane)
	}
}

func TestHistoryStepListScrollIndicators(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 24
	m.activeView = ViewHistory
	m.focusPane = FocusList

	// Seed with 20 historical steps
	for i := 0; i < 20; i++ {
		m.history = append(m.history, core.UnifiedAgentEvent{
			StepIndex: i,
			Summary:   fmt.Sprintf("Step %d execution details", i),
			Timestamp: time.Now(),
		})
	}

	// Case 1: At top (historyOffset = 0) -> NO top '...', YES bottom '...'
	m.historyOffset = 0
	m.selectedIdx = 0
	vTop := m.renderHistoryView()
	if strings.Contains(vTop, "STEPS (Newest First <)\n│   ...") {
		t.Error("Did not expect top '...' when at the newest step (historyOffset = 0)")
	}
	if !strings.Contains(vTop, "...") {
		t.Error("Expected bottom '...' when there are older steps below")
	}

	// Case 2: Scrolled down (historyOffset = 5) -> YES top '...', YES bottom '...'
	m.historyOffset = 5
	m.selectedIdx = 5
	vMid := m.renderHistoryView()
	if !strings.Contains(vMid, "...") {
		t.Error("Expected '...' scroll indicators when scrolled into the middle")
	}

	// Case 3: Step-by-step navigation down (j) through all 20 steps
	// Verified that selected step is ALWAYS visible on screen (never cut off below)
	m.historyOffset = 0
	m.selectedIdx = 0
	for step := 0; step < 20; step++ {
		v := m.renderHistoryView()
		expectedStepIdx := 19 - step
		expectedBadge := fmt.Sprintf("[%04d]", expectedStepIdx)
		if !strings.Contains(v, "> ") || !strings.Contains(v, expectedBadge) {
			t.Fatalf("Step %d (selectedIdx=%d, offset=%d, badge=%s) was not visible on screen! Rendered view:\n%s", expectedStepIdx, m.selectedIdx, m.historyOffset, expectedBadge, v)
		}
		// Press 'j'
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(Model)
	}
}

func TestHistoryFilteringAndStepSearch(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusList

	// Seed with distinct types and cache field observations.
	m.history = []core.UnifiedAgentEvent{
		{StepIndex: 1, Type: core.StepTypeUserInput, Summary: "User question"},
		{StepIndex: 2, Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, HasObservedContextTokens: true, ObservedContextTokens: 100, HasCachedContentTokens: true, CachedContentTokens: 85}, Summary: "Model plan"},
		{StepIndex: 3, Type: core.StepTypeToolCall, Usage: core.PersistedUsageObservation{Available: true, HasObservedContextTokens: true, ObservedContextTokens: 100, HasCachedContentTokens: true, CachedContentTokens: 90}, Summary: "Run command"},
		{StepIndex: 4, Type: core.StepTypeToolCall, Usage: core.PersistedUsageObservation{Available: true, HasObservedContextTokens: true, ObservedContextTokens: 100}, Summary: "Command result"},
		{StepIndex: 14, Type: core.StepTypeToolCall, Usage: core.PersistedUsageObservation{Available: true, HasObservedContextTokens: true, ObservedContextTokens: 100, HasCachedContentTokens: true, CachedContentTokens: 95}, Summary: "Write file"},
	}

	// 1. Initial State: All 5 events and [T:All] [C:All] rendered
	filtered := m.getFilteredHistory()
	if len(filtered) != 5 {
		t.Fatalf("Expected 5 events, got %d", len(filtered))
	}
	renderedInitial := m.renderHistoryView()
	if !strings.Contains(renderedInitial, "[T:All]") || !strings.Contains(renderedInitial, "[C:All]") {
		t.Fatalf("Expected '[T:All]' and '[C:All]' badges on header line, got:\n%s", renderedInitial)
	}

	// 2. Cycle Type Filter: press 't' -> Tool (should match Step 3, 4, 14)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = updated.(Model)
	if m.historyTypeFilter != TypeFilterTool {
		t.Fatalf("Expected TypeFilterTool, got %v", m.historyTypeFilter)
	}
	filtered = m.getFilteredHistory()
	if len(filtered) != 3 {
		t.Fatalf("Expected 3 tool events, got %d", len(filtered))
	}

	// 3. Cycle Cache Filter: press 'c' -> Observed (should match Step 3, 14)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if m.historyCacheFilter != CacheFilterObserved {
		t.Fatalf("Expected CacheFilterObserved, got %v", m.historyCacheFilter)
	}
	filtered = m.getFilteredHistory()
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 hit events, got %d", len(filtered))
	}

	// Cycle Cache Filter: press 'c' -> Omitted (should match Step 4)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if m.historyCacheFilter != CacheFilterOmitted {
		t.Fatalf("Expected CacheFilterOmitted, got %v", m.historyCacheFilter)
	}
	filtered = m.getFilteredHistory()
	if len(filtered) != 1 || filtered[0].StepIndex != 4 {
		t.Fatalf("Expected 1 omitted-cache event (step 4), got %v", filtered)
	}

	// 4. Press Esc to reset filters
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(Model)
	if m.historyTypeFilter != TypeFilterAll || m.historyCacheFilter != CacheFilterAll {
		t.Fatalf("Expected reset to All filters, got T:%v C:%v", m.historyTypeFilter, m.historyCacheFilter)
	}

	// 5. Jump-to-Step Search: press '/' then type '14'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	if !m.isHistorySearching {
		t.Fatal("Expected isHistorySearching=true after pressing '/'")
	}

	// Type '1'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = updated.(Model)
	// Type '4'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = updated.(Model)
	if m.historyStepQuery != "14" {
		t.Fatalf("Expected historyStepQuery='14', got '%s'", m.historyStepQuery)
	}

	// Step list remains full during typing (not destructive filter)
	filtered = m.getFilteredHistory()
	if len(filtered) == 0 {
		t.Fatal("Expected full history preserved during typing")
	}

	// Press Enter to confirm jump
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.isHistorySearching {
		t.Fatal("Expected isHistorySearching=false after pressing Enter")
	}

	// Verify cursor jumped to step 14
	currentStep := filtered[len(filtered)-1-m.selectedIdx]
	if currentStep.StepIndex != 14 {
		t.Fatalf("Expected cursor to jump to step 14, got step %d (selectedIdx=%d)", currentStep.StepIndex, m.selectedIdx)
	}

	// 6. Test Non-existent Step Jump Error
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.historySearchErr == "" {
		t.Fatal("Expected historySearchErr for non-existent step 999")
	}

	// Moving cursor clears error
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)
	if m.historySearchErr != "" {
		t.Fatalf("Expected historySearchErr cleared after navigation, got '%s'", m.historySearchErr)
	}
}

func TestHistoryTreeAndDistinctiveLabels(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusList

	m.history = []core.UnifiedAgentEvent{
		{StepIndex: 1, Type: core.StepTypeUserInput, Scope: core.ScopeUserInteraction, Summary: "User question"},
		{StepIndex: 2, Type: core.StepTypeToolCall, Scope: core.ScopeCloudInference, Summary: "Run command", ToolCalls: []core.ToolCallInfo{{ToolName: "run_command"}}},
		{StepIndex: 3, Type: core.StepTypeRunCommand, Scope: core.ScopeLocalExecution, Summary: "Command output"},
		{StepIndex: 4, Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, Summary: "Final answer"},
		{StepIndex: 5, Type: core.StepTypeGeneric, Scope: core.ScopeLocalExecution, Summary: "Generic output"},
	}

	rendered := m.renderHistoryView()
	if !strings.Contains(rendered, "┌") {
		t.Fatalf("Expected '┌' bracket top connector in rendered history view, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "└") {
		t.Fatalf("Expected '└' bracket bottom connector in rendered history view, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "Model: gemini-3.7-flash") {
		t.Fatalf("History must not invent an unobserved model name, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Tool: run_cmd") {
		t.Fatalf("Expected 'Tool: run_cmd' in rendered history view, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "💻 OUTPUT") {
		t.Fatalf("Expected '💻 OUTPUT' badge in rendered history view, got:\n%s", rendered)
	}
}

func TestHistoryThreePanelSplitAndZeroTruncation(t *testing.T) {
	m := NewModel("test-session", false)
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusList

	m.history = []core.UnifiedAgentEvent{
		{
			StepIndex:  1,
			Type:       core.StepTypeUserInput,
			Scope:      core.ScopeUserInteraction,
			Summary:    "How do I optimize Gemini cache?",
			RawContent: "How do I optimize Gemini cache in Antigravity?",
		},
		{
			StepIndex:  2,
			Type:       core.StepTypeModelResponse,
			Scope:      core.ScopeCloudInference,
			Summary:    "Model response with tool call",
			RawContent: "Let me check the tools.",
			Usage: core.PersistedUsageObservation{
				Available: true, ModelName: "gemini-3.7-flash", HasObservedContextTokens: true, ObservedContextTokens: 151479, HasCachedContentTokens: true, CachedContentTokens: 150866,
			},
		},
	}
	m.selectedIdx = 0

	// 1. Full-Width (width >= 100)
	m.width = 120
	renderedFull := m.renderHistoryView()
	if !strings.Contains(renderedFull, "STEP TELEMETRY & METRICS") {
		t.Fatalf("Expected 'STEP TELEMETRY & METRICS' in full-width history view, got:\n%s", renderedFull)
	}
	if !strings.Contains(renderedFull, "CONTENT PAYLOAD") {
		t.Fatalf("Expected 'CONTENT PAYLOAD' in full-width history view, got:\n%s", renderedFull)
	}
	if !strings.Contains(renderedFull, "Filters:") {
		t.Fatalf("Expected 'Filters:' line in full-width history view, got:\n%s", renderedFull)
	}

	// 2. Half-Width (width < 100)
	m.width = 80
	renderedHalf := m.renderHistoryView()
	if !strings.Contains(renderedHalf, "STEP TELEMETRY") {
		t.Fatalf("Expected 'STEP TELEMETRY' in half-width history view, got:\n%s", renderedHalf)
	}
	if !strings.Contains(renderedHalf, "CONTENT PAYLOAD") {
		t.Fatalf("Expected 'CONTENT PAYLOAD' in half-width history view, got:\n%s", renderedHalf)
	}
	if !strings.Contains(renderedHalf, "Filters:") {
		t.Fatalf("Expected 'Filters:' line in half-width history view, got:\n%s", renderedHalf)
	}
}

func TestAllViewsZeroHeightVariationAcrossSizes(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}, {140, 40}} {
		for _, view := range []ActiveView{ViewDashboard, ViewContext, ViewHistory, ViewDocs} {
			m := NewModel("test-session", false)
			m.width = size.w
			m.height = size.h
			m.activeView = view

			v := m.View()
			lines := strings.Split(v, "\n")

			if len(lines) != size.h {
				t.Fatalf("[%dx%d] View %v expected exactly %d lines, got %d", size.w, size.h, view, size.h, len(lines))
			}
		}
	}
}

func TestSessionSwitcherModalRenderingAndFilter(t *testing.T) {
	m := NewModel("session-test", true)
	m.width = 100
	m.height = 30

	if !m.isSessionSwitcherOpen {
		t.Fatal("Expected Session Switcher to be open by default")
	}

	view := m.View()
	if !strings.Contains(view, "ANTIGRAVITY SESSIONS") {
		t.Errorf("Expected Antigravity session title in view, got: %s", view)
	}
	if strings.Contains(view, "Claude Code") || strings.Contains(view, "OpenCode") {
		t.Errorf("Unsupported adapter tabs must not appear in view: %s", view)
	}

	// Test Ctrl+P toggle
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updatedModel.(Model)
	if m.isSessionSwitcherOpen {
		t.Error("Expected Ctrl+P to close session switcher modal")
	}

	// Re-open with Ctrl+P
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updatedModel.(Model)
	if !m.isSessionSwitcherOpen {
		t.Error("Expected Ctrl+P to re-open session switcher modal")
	}

	// Test typing in search box
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("aa72")})
	m = updatedModel.(Model)
	if m.sessionSearchQuery != "aa72" {
		t.Errorf("Expected query 'aa72', got '%s'", m.sessionSearchQuery)
	}
}

func TestSessionSwitcherHalfScreenVerticalStacking(t *testing.T) {
	m := NewModel("test-session-1", true)
	// Half-screen / narrow terminal width (60 cols)
	m.width = 60
	m.height = 30

	var mockSessions []core.SessionInfo
	for i := 0; i < 6; i++ {
		mockSessions = append(mockSessions, core.SessionInfo{
			SessionID:    fmt.Sprintf("session-narrow-%d", i),
			WorkspaceDir: fmt.Sprintf("/home/example/projects/proj-%d", i),
			ShortPath:    fmt.Sprintf("projects/proj-%d", i),
			InitialGoal:  fmt.Sprintf("Initial goal for session %d", i),
			LastPrompt:   fmt.Sprintf("Last prompt for session %d", i),
			StepCount:    10 * (i + 1),
			LastModified: time.Now().Add(time.Duration(-i) * time.Hour),
		})
	}
	m.availableSessions = mockSessions
	m.filteredSessions = filterSessions(m.availableSessions, "")

	// Case 1: At top (index 0) -> NO top '...', YES bottom '...'
	m.switcherSelectedIdx = 0
	viewTop := m.View()
	if strings.Contains(viewTop, "Filter: [█]\n│ ──────────────────────────────────────────────────────── │\n│   ...") {
		t.Error("Did not expect top '...' when at the very first session")
	}
	if !strings.Contains(viewTop, "...") {
		t.Error("Expected bottom '...' when there are more sessions below")
	}

	// Case 2: In middle (index 3) -> YES top '...', YES bottom '...'
	m.switcherSelectedIdx = 3
	viewMid := m.View()
	if !strings.Contains(viewMid, "...") {
		t.Error("Expected '...' indicators in middle of list")
	}

	// Case 3: At bottom (index 5) -> YES top '...', NO bottom '...'
	m.switcherSelectedIdx = 5
	viewBot := m.View()
	if !strings.Contains(viewBot, "projects/proj-5") {
		t.Errorf("Expected last session 'projects/proj-5' to be visible, got: %s", viewBot)
	}
}

func TestHistoryInspectionAntiJitterLock(t *testing.T) {
	m := NewModel("test-session", false)
	m.activeView = ViewHistory
	m.focusPane = FocusList

	// Seed with 5 historical steps
	for i := 0; i < 5; i++ {
		m.history = append(m.history, core.UnifiedAgentEvent{
			StepIndex: i,
			Summary:   "Step " + string(rune('0'+i)),
			Timestamp: time.Now(),
		})
	}
	m.selectedIdx = 2 // Pointing to Step 2 (realIdx = 5 - 1 - 2 = 2)

	selectedEv, _ := m.getSelectedEvent()
	if selectedEv.StepIndex != 2 {
		t.Fatalf("Expected selected step to be 2, got %d", selectedEv.StepIndex)
	}

	// New event arrives while user is inspecting past step
	newEvent := core.UnifiedAgentEvent{
		StepIndex: 5,
		Summary:   "New incoming Step 5",
		Timestamp: time.Now(),
	}
	updated, _ := m.Update(AgentEventMsg(newEvent))
	m = updated.(Model)

	// Verify that the inspected step REMAINS step 2!
	selectedEvAfter, _ := m.getSelectedEvent()
	if selectedEvAfter.StepIndex != 2 {
		t.Fatalf("Expected inspected step to remain locked at 2, got %d", selectedEvAfter.StepIndex)
	}
}

func TestSessionSwitcherKeyboardNavigationAndActions(t *testing.T) {
	m := NewModel("test-session-1", true)
	m.width = 100
	m.height = 30

	// Mock available sessions
	m.availableSessions = []core.SessionInfo{
		{
			SessionID:    "session-alpha-12345678",
			WorkspaceDir: "/home/example/projects/project-a",
			ShortPath:    "projects/project-a",
			InitialGoal:  "Create a new microservice",
			LastPrompt:   "Add unit tests",
			StepCount:    100,
			LastModified: time.Now(),
		},
		{
			SessionID:    "session-beta-87654321",
			WorkspaceDir: "/home/example/projects/project-b",
			ShortPath:    "projects/project-b",
			InitialGoal:  "Fix database deadlock bug",
			LastPrompt:   "Verify WAL mode",
			StepCount:    50,
			LastModified: time.Now().Add(-1 * time.Hour),
		},
	}
	m.filteredSessions = filterSessions(m.availableSessions, "")
	m.switcherSelectedIdx = 0

	// Initial catalog contains the two supported Antigravity sessions.
	if len(m.filteredSessions) != 2 {
		t.Fatalf("Expected 2 AGY sessions, got %d", len(m.filteredSessions))
	}

	// 1. Test Navigation Down with Ctrl+j (Vim)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m = updated.(Model)
	if m.switcherSelectedIdx != 1 {
		t.Errorf("Expected switcherSelectedIdx=1 after Ctrl+j, got %d", m.switcherSelectedIdx)
	}

	// 2. Test Navigation Up with Ctrl+k (Vim)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)
	if m.switcherSelectedIdx != 0 {
		t.Errorf("Expected switcherSelectedIdx=0 after Ctrl+k, got %d", m.switcherSelectedIdx)
	}

	// 3. Test typing filter for path or prompt keywords.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("deadlock")})
	m = updated.(Model)
	if len(m.filteredSessions) != 1 || m.filteredSessions[0].SessionID != "session-beta-87654321" {
		t.Fatalf("Expected 1 filtered session matching 'deadlock', got %d", len(m.filteredSessions))
	}

	// 4. Test backspace.
	for i := 0; i < 8; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = updated.(Model)
	}
	if len(m.filteredSessions) != 2 {
		t.Errorf("Expected 2 sessions restored after backspace, got %d", len(m.filteredSessions))
	}

	// 5. Test dual-pane rendering output (zero emojis and inspector content).
	view := m.View()
	if strings.Contains(view, "📂") || strings.Contains(view, "🏷️") || strings.Contains(view, "🎯") {
		t.Errorf("Expected Zero Emojis in switcher view, but found emoji")
	}
	if !strings.Contains(view, "INITIAL GOAL / FIRST PROMPT") {
		t.Errorf("Expected 'INITIAL GOAL / FIRST PROMPT' inspector block, got: %s", view)
	}
	if !strings.Contains(view, "LATEST PROGRESS / LAST ACTION") {
		t.Errorf("Expected 'LATEST PROGRESS / LAST ACTION' inspector block, got: %s", view)
	}
	if !strings.Contains(view, "projects/project-a") {
		t.Errorf("Expected 'projects/project-a' short path in view, got: %s", view)
	}

	// 8. Test Escape Key (Cancel without switching)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.isSessionSwitcherOpen {
		t.Error("Expected Esc key to close switcher modal")
	}
}

func TestFormatShortPath(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"/home/example/projects/ithome2026", "projects/ithome2026"},
		{"/home/example/projects/bookkeeper", "projects/bookkeeper"},
		{"/home/example/.gemini/antigravity-cli", ".gemini/antigravity-cli"},
		{"/project", "project"},
		{"", "workspace"},
	}

	for _, c := range cases {
		got := antigravity.FormatShortPath(c.input)
		if got != c.expected {
			t.Errorf("FormatShortPath(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}

func TestDirectSessionSwitchMsgState(t *testing.T) {
	m := NewModel("initial-session", false)

	// Simulate switching session with mock events
	mockEvents := []core.UnifiedAgentEvent{
		{
			StepIndex: 1,
			Type:      core.StepTypeUserInput,
			Summary:   "Hello Agent",
			Timestamp: time.Now(),
		},
		{
			StepIndex: 2,
			Type:      core.StepTypeModelResponse,
			Summary:   "Hello! How can I help you?",
			Timestamp: time.Now(),
			Usage: core.PersistedUsageObservation{
				Available: true, HasObservedContextTokens: true, ObservedContextTokens: 1500, HasCachedContentTokens: true, CachedContentTokens: 1200,
			},
		},
	}

	updated, _ := m.Update(SessionSwitchedMsg{
		SessionID: "target-session-123",
		Events:    mockEvents,
	})
	m = updated.(Model)

	if m.sessionID != "target-session-123" {
		t.Errorf("Expected sessionID 'target-session-123', got '%s'", m.sessionID)
	}
	if len(m.history) != 2 {
		t.Errorf("Expected 2 history events, got %d", len(m.history))
	}
	if m.dashboardIdx != 1 {
		t.Errorf("Expected dashboardIdx=1 (latest event), got %d", m.dashboardIdx)
	}
	if m.isSessionSwitcherOpen {
		t.Error("Expected session switcher to be closed after switch")
	}
	if !strings.Contains(m.clipboardStatus, "Switched to session") {
		t.Errorf("Expected status message in clipboardStatus, got: %s", m.clipboardStatus)
	}
}

func TestWidthMeasurement(t *testing.T) {
	m := NewModel("session-test", false)
	m.width = 120
	m.height = 30

	header := m.renderHeader()
	if lipgloss.Width(header) > m.width {
		t.Errorf("Header width %d exceeds max width %d", lipgloss.Width(header), m.width)
	}

	m.activeView = ViewDashboard
	dash := m.renderDashboardView()
	dashLines := strings.Split(dash, "\n")
	for i, l := range dashLines {
		if lipgloss.Width(l) > m.width {
			t.Errorf("Dashboard line %d width %d exceeds max width %d", i, lipgloss.Width(l), m.width)
		}
	}

	m.activeView = ViewHistory
	hist := m.renderHistoryView()
	histLines := strings.Split(hist, "\n")
	for i, l := range histLines {
		if lipgloss.Width(l) > m.width {
			t.Errorf("History line %d width %d exceeds max width %d", i, lipgloss.Width(l), m.width)
		}
	}
}

func TestDashboardKpiRendering(t *testing.T) {
	m := NewModel("test-session-multi-model", false)
	m.width = 120
	m.height = 35

	// Add multi-model events to history
	m.history = append(m.history, core.UnifiedAgentEvent{
		StepIndex: 1,
		Scope:     core.ScopeCloudInference,
		Type:      core.StepTypeModelResponse,
		Usage: core.PersistedUsageObservation{
			Available: true, ModelName: "gemini-3.7-flash", HasObservedContextTokens: true, ObservedContextTokens: 120000, HasMeteredInputTokens: true, MeteredInputTokens: 20000, HasCachedContentTokens: true, CachedContentTokens: 100000,
		},
	})
	m.history = append(m.history, core.UnifiedAgentEvent{
		StepIndex: 2,
		Scope:     core.ScopeCloudInference,
		Type:      core.StepTypeModelResponse,
		Usage: core.PersistedUsageObservation{
			Available: true, ModelName: "claude-3.7-sonnet", HasObservedContextTokens: true, ObservedContextTokens: 80000, HasMeteredInputTokens: true, MeteredInputTokens: 10000, HasCachedContentTokens: true, CachedContentTokens: 70000,
		},
	})
	m.latestEvent = m.history[1]

	m.activeView = ViewDashboard
	viewStr := m.View()

	// 1. Verify metered-usage KPI cards are present.
	requireViewContains(t, viewStr, "TOTAL PROCESSED")
	requireViewContains(t, viewStr, "CACHE HIT VOLUME")
	requireViewContains(t, viewStr, "UNCACHED INBOUND")
	requireViewContains(t, viewStr, "EFFECTIVE TOKENS")

	// 2. Verify Multi-Model breakdown table shows both models.
	requireViewContains(t, viewStr, "gemini-3.7-flash")
	requireViewContains(t, viewStr, "claude-3.7-sonnet")
	requireViewContains(t, viewStr, "TOTAL SUMMARY")

}

func TestDashboardHalfWidthResponsiveRendering(t *testing.T) {
	m := NewModel("test-session-half-width", false)
	m.width = 80
	m.height = 30

	m.history = append(m.history, core.UnifiedAgentEvent{
		StepIndex: 1,
		Scope:     core.ScopeCloudInference,
		Type:      core.StepTypeModelResponse,
		Usage: core.PersistedUsageObservation{
			Available: true, ModelName: "gemini-3.7-flash", HasObservedContextTokens: true, ObservedContextTokens: 120000, HasMeteredInputTokens: true, MeteredInputTokens: 20000, HasCachedContentTokens: true, CachedContentTokens: 100000,
		},
	})
	m.latestEvent = m.history[0]

	m.activeView = ViewDashboard
	viewStr := m.View()

	// In half-width (80 cols), ensure 2-column KPI labels are complete without truncation
	requireViewContains(t, viewStr, "TOTAL PROCESSED")
	requireViewContains(t, viewStr, "CACHE HIT VOLUME")
	requireViewContains(t, viewStr, "UNCACHED INBOUND")
	requireRenderedLinesWithinWidth(t, viewStr, 80)
}

func TestSpaceBetweenRowDistribution(t *testing.T) {
	cols := []string{"Model Name", "Turns", "Processed", "Cached (Hit %)", "Cached Saved (%)"}
	minWidths := []int{18, 5, 10, 16, 16}
	leftAligns := []bool{true, false, false, false, false}

	for _, targetW := range []int{76, 90, 116, 150, 180} {
		res := formatSpaceBetweenRow(cols, minWidths, leftAligns, targetW)
		actualW := lipgloss.Width(res)
		if actualW != targetW {
			t.Errorf("Target width %d, but got %d: '%s'", targetW, actualW, res)
		}
	}
}

func TestCacheFilterStrictCloudIsolation(t *testing.T) {
	m := NewModel("test-session", true)
	m.width = 120
	m.height = 30
	m.activeView = ViewHistory

	m.history = []core.UnifiedAgentEvent{
		{StepIndex: 0, Type: core.StepTypeUserInput, Summary: "User prompt"},
		{StepIndex: 1, Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, HasObservedContextTokens: true, ObservedContextTokens: 1000, HasCachedContentTokens: true, CachedContentTokens: 900}, Summary: "Model Response 1"},
		{StepIndex: 2, Type: core.StepTypeRunCommand, Summary: "cat file.go"},
		{StepIndex: 3, Type: core.StepTypeCheckpoint, Summary: "Checkpoint 1"},
		{StepIndex: 4, Type: core.StepTypeModelResponse, Usage: core.PersistedUsageObservation{Available: true, HasObservedContextTokens: true, ObservedContextTokens: 1000}, Summary: "Model Response 2"},
	}

	// 1. Omitted Filter: should only match Step 4.
	m.historyCacheFilter = CacheFilterOmitted
	filtered := m.getFilteredHistory()
	if len(filtered) != 1 {
		t.Fatalf("Expected exactly 1 omitted-cache event (Step 4), got %d: %v", len(filtered), filtered)
	}
	if filtered[0].StepIndex != 4 {
		t.Errorf("Expected Step 4, got Step %d", filtered[0].StepIndex)
	}

	// 2. Observed Filter: should only match Step 1.
	m.historyCacheFilter = CacheFilterObserved
	filtered = m.getFilteredHistory()
	if len(filtered) != 1 {
		t.Fatalf("Expected exactly 1 observed-cache event (Step 1), got %d: %v", len(filtered), filtered)
	}
	if filtered[0].StepIndex != 1 {
		t.Errorf("Expected Step 1, got Step %d", filtered[0].StepIndex)
	}
}

func TestCompactionInspectorPanel(t *testing.T) {
	m := NewModel("test-session", true)
	cpEvent := core.UnifiedAgentEvent{
		StepIndex:  779,
		Type:       core.StepTypeCheckpoint,
		Scope:      core.ScopeSystemCompaction,
		Status:     "DONE",
		RawContent: "{{ CHECKPOINT 4 }} Summary of truncated context...",
	}

	linesCompact := m.buildTelemetryPanelLines(cpEvent, 40, true)
	compactStr := strings.Join(linesCompact, "\n")
	if !strings.Contains(compactStr, "Context Compaction") {
		t.Errorf("Compact panel missing 'Context Compaction':\n%s", compactStr)
	}

	linesFull := m.buildTelemetryPanelLines(cpEvent, 80, false)
	fullStr := strings.Join(linesFull, "\n")
	if !strings.Contains(fullStr, "CONTEXT COMPACTION (CHECKPOINT)") {
		t.Errorf("Full panel missing 'CONTEXT COMPACTION (CHECKPOINT)':\n%s", fullStr)
	}
}

func TestFooterStrictlyPinnedAtBottom(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}, {140, 40}} {
		for _, view := range []ActiveView{ViewDashboard, ViewContext, ViewHistory, ViewDocs} {
			m := NewModel("test-session", false)
			m.width = size.w
			m.height = size.h
			m.activeView = view

			v := m.View()
			lines := strings.Split(v, "\n")

			if len(lines) != size.h {
				t.Fatalf("[%dx%d] View %v expected %d lines, got %d", size.w, size.h, view, size.h, len(lines))
			}

			lastLine := lines[len(lines)-1]
			if !strings.Contains(lastLine, "Shortcuts") && !strings.Contains(lastLine, "Command") && !strings.Contains(lastLine, "Quit") {
				t.Fatalf("[%dx%d] View %v footer is not on bottom line! Last line: %q", size.w, size.h, view, lastLine)
			}
		}
	}
}

func TestRawWireModeAntiOverflowGuards(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}} {
		m := NewModel("test-session", false)
		m.width = size.w
		m.height = size.h
		m.activeView = ViewContext
		m.isContextRawMode = true

		v := m.View()
		lines := strings.Split(v, "\n")

		if len(lines) != size.h {
			t.Fatalf("[%dx%d] Raw mode expected %d lines, got %d", size.w, size.h, size.h, len(lines))
		}

		for i, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth > size.w {
				t.Fatalf("[%dx%d] Line %d width %d exceeds terminal width %d:\n%s", size.w, size.h, i, lineWidth, size.w, line)
			}
		}
	}
}

func TestCommandModeColonQuitAndSave(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewContext

	// 1. Bare 'q' key does not quit immediately
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updatedModel.(Model)
	if cmd != nil {
		t.Errorf("Expected bare 'q' not to issue tea.Quit, but got cmd")
	}
	if !strings.Contains(m.statusMessage, ":q") {
		t.Errorf("Expected status message prompting :q, got %q", m.statusMessage)
	}

	// 2. Pressing ':' enters command mode
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = updatedModel.(Model)
	if !m.isCommandMode {
		t.Fatalf("Expected isCommandMode to be true")
	}

	// 3. Typing 'q' into command buffer
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updatedModel.(Model)
	if m.commandInput != "q" {
		t.Fatalf("Expected commandInput 'q', got %q", m.commandInput)
	}

	// 4. Pressing Enter with ':q' issues tea.Quit
	_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if quitCmd == nil {
		t.Fatalf("Expected :q + Enter to issue tea.Quit")
	}

	// 5. Esc cancels command mode
	m.isCommandMode = true
	m.commandInput = "something"
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedModel.(Model)
	if m.isCommandMode || m.commandInput != "" {
		t.Fatalf("Expected Esc to cancel command mode")
	}
}

func TestContextSubcatAllAndNavigation(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 120
	m.height = 35
	m.activeView = ViewContext

	// Starts at SubcatAll (0)
	if m.contextSubItemIndex != 0 {
		t.Errorf("Expected default subcat index 0 (SubcatAll), got %d", m.contextSubItemIndex)
	}

	viewStr := m.View()
	if !strings.Contains(viewStr, "CONTEXT EVIDENCE") {
		t.Errorf("Expected view to render evidence-oriented context root")
	}

	// Navigate down to the last tree item (SubcatBuffers).
	for i := SubcatAll; i < SubcatLast; i++ {
		updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updatedModel.(Model)
	}

	if m.contextSubItemIndex != SubcatLast {
		t.Errorf("Expected contextSubItemIndex %d after navigation, got %d", SubcatLast, m.contextSubItemIndex)
	}

	// Navigate back up with 'g'
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = updatedModel.(Model)
	if m.contextSubItemIndex != 0 {
		t.Errorf("Expected contextSubItemIndex 0 after 'g', got %d", m.contextSubItemIndex)
	}
}

func TestContextClipboardCopyAction(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 120
	m.height = 35
	m.activeView = ViewContext

	// Pressing 'y' copies and triggers toast
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updatedModel.(Model)

	if !strings.Contains(m.statusMessage, "Copied") {
		t.Errorf("Expected status message to contain 'Copied', got %q", m.statusMessage)
	}
}

func TestInboundPromptAntiOverflowMultiLine(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}} {
		m := NewModel("test-session", false)
		m.width = size.w
		m.height = size.h
		m.activeView = ViewContext
		m.contextSubItemIndex = SubcatPrompt

		// Add an extremely long, unwrapped multi-line user input
		m.history = []core.UnifiedAgentEvent{
			{
				StepIndex:  1,
				Type:       core.StepTypeUserInput,
				RawContent: "This is an extremely long user prompt that exceeds the terminal width by a huge margin and should be properly wrapped without causing any overflow or breaking of the surrounding borders in the terminal UI! " + strings.Repeat("VeryLongTokenSequenceWithoutSpaces", 5),
			},
		}

		v := m.View()
		lines := strings.Split(v, "\n")

		if len(lines) != size.h {
			t.Fatalf("[%dx%d] Inbound prompt view expected %d lines, got %d", size.w, size.h, size.h, len(lines))
		}

		for i, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth > size.w {
				t.Fatalf("[%dx%d] Line %d width %d exceeds terminal width %d:\n%s", size.w, size.h, i, lineWidth, size.w, line)
			}
		}
	}
}

type mockSessionSwitcher struct {
	called        bool
	lastSessionID string
}

func (m *mockSessionSwitcher) SwitchSession(sessionID string) error {
	m.called = true
	m.lastSessionID = sessionID
	return nil
}

func TestSessionSwitch_Neg_DropStaleSessionEvents(t *testing.T) {
	m := NewModel("session-new", false)
	staleEvent := core.UnifiedAgentEvent{
		SessionID: "session-old",
		StepIndex: 1,
		Summary:   "Stale event from previous session",
	}

	updated, _ := m.Update(AgentEventMsg(staleEvent))
	m = updated.(Model)

	if len(m.history) != 0 {
		t.Fatalf("Expected stale event to be dropped (history len 0), got %d events in history", len(m.history))
	}
}

func TestSessionSwitch_Pos_AcceptMatchingSessionEvent(t *testing.T) {
	m := NewModel("session-new", false)
	validEvent := core.UnifiedAgentEvent{
		SessionID: "session-new",
		StepIndex: 1,
		Summary:   "Valid event for active session",
	}

	updated, _ := m.Update(AgentEventMsg(validEvent))
	m = updated.(Model)

	if len(m.history) != 1 {
		t.Fatalf("Expected valid event to be accepted (history len 1), got %d", len(m.history))
	}
	if m.latestEvent.Summary != "Valid event for active session" {
		t.Fatalf("Expected latest event summary 'Valid event for active session', got '%s'", m.latestEvent.Summary)
	}
}

func TestSessionSwitch_Pos_SessionResetMsgClearsHistory(t *testing.T) {
	m := NewModel("session-old", false)
	m.history = []core.UnifiedAgentEvent{
		{SessionID: "session-old", StepIndex: 1, Summary: "Step 1"},
		{SessionID: "session-old", StepIndex: 2, Summary: "Step 2"},
	}
	m.dashboardIdx = 1

	updated, _ := m.Update(SessionResetMsg{SessionID: "session-new"})
	m = updated.(Model)

	if m.sessionID != "session-new" {
		t.Fatalf("Expected sessionID 'session-new', got '%s'", m.sessionID)
	}
	if len(m.history) != 0 {
		t.Fatalf("Expected history to be cleared (len 0), got %d", len(m.history))
	}
	if m.dashboardIdx != 0 {
		t.Fatalf("Expected dashboardIdx to be reset to 0, got %d", m.dashboardIdx)
	}
}

func TestSessionSwitch_Pos_SwitchSessionReqMsgDelegatesToSwitcher(t *testing.T) {
	mock := &mockSessionSwitcher{}
	m := NewModel("session-A", false, mock)
	m.history = []core.UnifiedAgentEvent{
		{SessionID: "session-A", StepIndex: 1, Summary: "Step 1"},
	}

	updated, _ := m.Update(SwitchSessionReqMsg{SessionID: "session-B"})
	m = updated.(Model)

	if !mock.called {
		t.Fatalf("Expected switcher.SwitchSession to be called, but it was not")
	}
	if mock.lastSessionID != "session-B" {
		t.Fatalf("Expected switcher target 'session-B', got '%s'", mock.lastSessionID)
	}
	if m.sessionID != "session-B" {
		t.Fatalf("Expected model sessionID 'session-B', got '%s'", m.sessionID)
	}
	if len(m.history) != 0 {
		t.Fatalf("Expected model history to be cleared on switch request, got %d", len(m.history))
	}
}
func TestVisualMode_Pos_ShiftV_Navigation_G_and_g(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 120
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusDetail
	m.history = []core.UnifiedAgentEvent{
		{
			StepIndex:  1,
			Summary:    "Step 1 Summary",
			RawContent: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\nLine 11\nLine 12\nLine 13\nLine 14\nLine 15",
		},
	}

	// 1. Enter Visual Mode with 'V'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m = updated.(Model)
	if !m.isVisualMode {
		t.Fatalf("Expected isVisualMode to be true on 'V'")
	}
	if m.visualCursor != 0 {
		t.Errorf("Expected visualCursor 0, got %d", m.visualCursor)
	}

	// 2. Press 'G' (Shift+G) to jump to last line
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updated.(Model)
	if m.visualCursor <= 0 {
		t.Errorf("Expected visualCursor to jump to last line, got %d", m.visualCursor)
	}
	lastLine := m.visualCursor

	// 3. Press 'g' to jump back to line 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = updated.(Model)
	if m.visualCursor != 0 {
		t.Errorf("Expected visualCursor 0 on 'g', got %d", m.visualCursor)
	}

	// 4. Press 'j' to move down 1 line
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)
	if m.visualCursor != 1 {
		t.Errorf("Expected visualCursor 1 on 'j', got %d", m.visualCursor)
	}

	// 5. Press 'k' to move back up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(Model)
	if m.visualCursor != 0 {
		t.Errorf("Expected visualCursor 0 on 'k', got %d", m.visualCursor)
	}

	// 6. Press 'G' again and yank with 'y'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updated.(Model)
	if m.visualCursor != lastLine {
		t.Errorf("Expected visualCursor %d on 'G', got %d", lastLine, m.visualCursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(Model)
	if m.isVisualMode {
		t.Errorf("Expected isVisualMode to be false after 'y'")
	}
	if !strings.Contains(m.clipboardStatus, "Copied") {
		t.Errorf("Expected clipboardStatus confirmation, got: %s", m.clipboardStatus)
	}
}

func TestNormalMode_Pos_CopyListAndDetail(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 120
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusList
	m.history = []core.UnifiedAgentEvent{
		{
			StepIndex:  1,
			Summary:    "Step 1 Old",
			RawContent: "Raw Content 1",
		},
		{
			StepIndex:  2,
			Summary:    "Step 2 Recent",
			RawContent: "Raw Content 2",
		},
	}

	// In FocusList (selectedIdx 0 corresponds to latest Step 2)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(Model)
	if !strings.Contains(m.clipboardStatus, "Step #2") {
		t.Errorf("Expected clipboardStatus for Step #2, got: %s", m.clipboardStatus)
	}

	// Switch to FocusDetail
	m.focusPane = FocusDetail
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(Model)
	if !strings.Contains(m.clipboardStatus, "Step #2") {
		t.Errorf("Expected clipboardStatus for Step #2 detail, got: %s", m.clipboardStatus)
	}
}

func TestVisualMode_Pos_YankWithCAndEnter(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 120
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusDetail
	m.history = []core.UnifiedAgentEvent{
		{
			StepIndex:  1,
			Summary:    "Step 1",
			RawContent: "Line A\nLine B\nLine C",
		},
	}

	// 1. Test yank with 'c'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(Model)
	if !m.isVisualMode {
		t.Fatalf("Expected isVisualMode on 'v'")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if m.isVisualMode {
		t.Errorf("Expected isVisualMode to exit on 'c'")
	}
	if !strings.Contains(m.clipboardStatus, "Copied") {
		t.Errorf("Expected clipboardStatus on 'c'")
	}

	// 2. Test yank with 'enter'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.isVisualMode {
		t.Errorf("Expected isVisualMode to exit on 'enter'")
	}
}

func TestVisualMode_Neg_EscCancelsWithoutYank(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 120
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusDetail
	m.history = []core.UnifiedAgentEvent{
		{
			StepIndex:  1,
			Summary:    "Step 1",
			RawContent: "Line A\nLine B",
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m = updated.(Model)
	if !m.isVisualMode {
		t.Fatalf("Expected isVisualMode")
	}

	// Press Esc to cancel
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.isVisualMode {
		t.Errorf("Expected isVisualMode to be false after Esc")
	}
}

func TestVisualMode_Pos_PageDownAndPageUp(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 120
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusDetail
	m.history = []core.UnifiedAgentEvent{
		{
			StepIndex:  1,
			Summary:    "Step 1",
			RawContent: "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15\nL16\nL17\nL18\nL19\nL20\nL21\nL22\nL23\nL24\nL25",
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m = updated.(Model)

	// Press Ctrl+D
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)
	if m.visualCursor <= 0 {
		t.Errorf("Expected visualCursor to advance on Ctrl+D, got %d", m.visualCursor)
	}

	// Press Ctrl+U
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(Model)
	if m.visualCursor != 0 {
		t.Errorf("Expected visualCursor to return to 0 on Ctrl+U, got %d", m.visualCursor)
	}
}
