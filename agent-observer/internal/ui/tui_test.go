package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"agent-observer/internal/adapters/antigravity"
	"agent-observer/internal/core"
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

		t.Logf("[%dx%d] Full View lines count: %d", size.w, size.h, len(lines))

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

	// 1. Press '3' or 'i' to switch to View 3 Docs (Default: English)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
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

	// 4. Press '/' to activate search mode and search for "cache"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	if !m.isDocsSearching {
		t.Error("Expected isDocsSearching=true after pressing '/'")
	}

	for _, r := range "cache" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	cacheFilteredView := m.View()
	if !strings.Contains(cacheFilteredView, "[CACHE HIT]") || !strings.Contains(cacheFilteredView, "[PARTIAL HIT]") {
		t.Errorf("Expected cache status definitions in cacheFilteredView, got: %s", cacheFilteredView)
	}

	// 5. Press Esc to clear search, then search for "cot"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	for _, r := range "cot" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if m.docsSearchQuery != "cot" {
		t.Errorf("Expected docsSearchQuery='cot', got '%s'", m.docsSearchQuery)
	}

	filteredView := m.View()
	if !strings.Contains(filteredView, "Active Turn / CoT") {
		t.Errorf("Expected filtered view to contain 'Active Turn / CoT', got: %s", filteredView)
	}

	// 6. Press Esc to clear search
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.docsSearchQuery != "" {
		t.Errorf("Expected docsSearchQuery to be cleared, got '%s'", m.docsSearchQuery)
	}

	// 7. Test overscroll prevention on 'j' and 'G'
	for i := 0; i < 200; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(Model)
	}
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

	// 2. Press Tab -> ViewDocs
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.activeView != ViewDocs {
		t.Fatalf("After 2nd Tab, expected ViewDocs, got %v", m.activeView)
	}

	// 3. Press Tab -> ViewDashboard (Clockwise wrap)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.activeView != ViewDashboard {
		t.Fatalf("After 3rd Tab, expected ViewDashboard, got %v", m.activeView)
	}

	// 4. Press Shift+Tab -> ViewDocs (Counter-clockwise wrap)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.activeView != ViewDocs {
		t.Fatalf("After Shift+Tab, expected ViewDocs, got %v", m.activeView)
	}

	// 5. Press Shift+Tab -> ViewHistory
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.activeView != ViewHistory {
		t.Fatalf("After 2nd Shift+Tab, expected ViewHistory, got %v", m.activeView)
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
		expectedBadge := fmt.Sprintf("[%03d|", expectedStepIdx)
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

	// Seed with distinct types and cache statuses
	m.history = []core.UnifiedAgentEvent{
		{StepIndex: 1, Type: core.StepTypeUserInput, CacheStatus: "MISS", Summary: "User question"},
		{StepIndex: 2, Type: core.StepTypeModelResponse, CacheStatus: "HIT", Tokens: core.TokenBreakdown{CacheHitRate: 85.0}, Summary: "Model plan"},
		{StepIndex: 3, Type: core.StepTypeToolCall, CacheStatus: "HIT", Tokens: core.TokenBreakdown{CacheHitRate: 90.0}, Summary: "Run command"},
		{StepIndex: 4, Type: core.StepTypeToolResult, CacheStatus: "PARTIAL", Tokens: core.TokenBreakdown{CacheHitRate: 40.0}, Summary: "Command result"},
		{StepIndex: 14, Type: core.StepTypeToolCall, CacheStatus: "HIT", Tokens: core.TokenBreakdown{CacheHitRate: 95.0}, Summary: "Write file"},
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

	// 3. Cycle Cache Filter: press 'c' -> Hit (should match Step 3, 14)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if m.historyCacheFilter != CacheFilterHit {
		t.Fatalf("Expected CacheFilterHit, got %v", m.historyCacheFilter)
	}
	filtered = m.getFilteredHistory()
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 hit events, got %d", len(filtered))
	}

	// Cycle Cache Filter: press 'c' -> Partial (should match Step 4)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if m.historyCacheFilter != CacheFilterPartial {
		t.Fatalf("Expected CacheFilterPartial, got %v", m.historyCacheFilter)
	}
	filtered = m.getFilteredHistory()
	if len(filtered) != 1 || filtered[0].StepIndex != 4 {
		t.Fatalf("Expected 1 partial event (step 4), got %v", filtered)
	}

	// 4. Press Esc to reset filters
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(Model)
	if m.historyTypeFilter != TypeFilterAll || m.historyCacheFilter != CacheFilterAll {
		t.Fatalf("Expected reset to All filters, got T:%v C:%v", m.historyTypeFilter, m.historyCacheFilter)
	}

	// 5. Step Search: press '/' then type '14'
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

	// Filtered history should match step 14 only
	filtered = m.getFilteredHistory()
	if len(filtered) != 1 || filtered[0].StepIndex != 14 {
		t.Fatalf("Expected 1 filtered event for step 14, got %v", filtered)
	}

	// Press Enter to confirm search
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.isHistorySearching {
		t.Fatal("Expected isHistorySearching=false after pressing Enter")
	}

	// Press Esc to clear search query
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(Model)
	if m.historyStepQuery != "" {
		t.Fatalf("Expected empty historyStepQuery after Esc, got '%s'", m.historyStepQuery)
	}
}

func TestHistoryParentChildNavigationPN(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30
	m.activeView = ViewHistory
	m.focusPane = FocusDetail

	// Structured turn with Parent/Child and Consumed linkages:
	// Step 1 [USER] -> Step 2 [TOOL_CALL] -> Step 3 [RUN_CMD] -> Step 4 [MODEL_RESP]
	m.history = []core.UnifiedAgentEvent{
		{StepIndex: 1, Type: core.StepTypeUserInput, Scope: core.ScopeUserInteraction, Summary: "User question"},
		{StepIndex: 2, Type: core.StepTypeToolCall, Scope: core.ScopeCloudInference, ParentStepIdx: 1, ConsumedStepIndices: nil, Summary: "Run command"},
		{StepIndex: 3, Type: core.StepTypeRunCommand, Scope: core.ScopeLocalExecution, ParentStepIdx: 2, PackagedInStepIdx: 4, Summary: "Command output"},
		{StepIndex: 4, Type: core.StepTypeModelResponse, Scope: core.ScopeCloudInference, ParentStepIdx: 1, ConsumedStepIndices: []int{3}, Summary: "Final answer"},
	}

	// In getFilteredHistory(), realIdx = len(history) - 1 - selectedIdx
	// So selectedIdx = 0 corresponds to Step 4 (last event)
	// selectedIdx = 1 corresponds to Step 3 (RunCommand)
	m.selectedIdx = 1
	ev, ok := m.getSelectedEvent()
	if !ok || ev.StepIndex != 3 {
		t.Fatalf("Expected selected event to be Step 3, got %+v", ev)
	}

	// 1. On Local Step 3, press 'p' -> Should jump to Parent Step 2 (TOOL_CALL)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	ev, ok = m.getSelectedEvent()
	if !ok || ev.StepIndex != 2 {
		t.Fatalf("Expected jump to Step 2 after 'p', got Step %d", ev.StepIndex)
	}

	// 2. On Tool Call Step 2, press 'p' -> Should jump to User Input Step 1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	ev, ok = m.getSelectedEvent()
	if !ok || ev.StepIndex != 1 {
		t.Fatalf("Expected jump to Step 1 after 'p', got Step %d", ev.StepIndex)
	}

	// 3. On Step 3 (Local RunCommand), press 'n' -> Should jump to PackagedIn Step 4
	m.selectedIdx = 1 // Step 3
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = updated.(Model)
	ev, ok = m.getSelectedEvent()
	if !ok || ev.StepIndex != 4 {
		t.Fatalf("Expected jump to Step 4 after 'n', got Step %d", ev.StepIndex)
	}

	// 5. Test that 'p' and 'n' work in FocusList as well
	m.focusPane = FocusList
	m.selectedIdx = 1 // Step 3
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	ev, ok = m.getSelectedEvent()
	if !ok || ev.StepIndex != 2 {
		t.Fatalf("Expected jump to Step 2 after 'p' in FocusList, got Step %d", ev.StepIndex)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = updated.(Model)
	ev, ok = m.getSelectedEvent()
	if !ok || ev.StepIndex != 3 {
		t.Fatalf("Expected jump to Step 3 after 'n' in FocusList, got Step %d", ev.StepIndex)
	}

	// 6. Test that GENERIC steps render with '└── ' and '(Local)' in History View
	m.history = append(m.history, core.UnifiedAgentEvent{
		StepIndex: 5,
		Type:      core.StepTypeGeneric,
		Scope:     core.ScopeLocalExecution,
		Summary:   "Generic output",
	})
	rendered := m.renderHistoryView()
	if !strings.Contains(rendered, "└──") {
		t.Fatalf("Expected '└──' in rendered history view, got:\n%s", rendered)
	}
}


func TestAllViewsZeroHeightVariationAcrossSizes(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}, {140, 40}} {
		for _, view := range []ActiveView{ViewDashboard, ViewHistory, ViewDocs} {
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
	m := NewModel("aa726359-08e2-4687-a15c-073a2f4a705b", true)
	m.width = 100
	m.height = 30

	if !m.isSessionSwitcherOpen {
		t.Fatal("Expected Session Switcher to be open by default")
	}

	view := m.View()
	if !strings.Contains(view, "Antigravity") {
		t.Errorf("Expected tab 'Antigravity' in view, got: %s", view)
	}
	if !strings.Contains(view, "[1] Antigravity") {
		t.Errorf("Expected active tab '[1] Antigravity' in view, got: %s", view)
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
			AgentType:    core.AgentTypeAntigravity,
			SessionID:    fmt.Sprintf("session-narrow-%d", i),
			WorkspaceDir: fmt.Sprintf("/Users/test/Documents/self/proj-%d", i),
			ShortPath:    fmt.Sprintf("self/proj-%d", i),
			InitialGoal:  fmt.Sprintf("Initial goal for session %d", i),
			LastPrompt:   fmt.Sprintf("Last prompt for session %d", i),
			StepCount:    10 * (i + 1),
			LastModified: time.Now().Add(time.Duration(-i) * time.Hour),
		})
	}
	m.availableSessions = mockSessions
	m.selectedAgentTab = core.AgentTypeAntigravity
	m.filteredSessions = filterSessions(m.availableSessions, "", m.selectedAgentTab)

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
	if !strings.Contains(viewBot, "self/proj-5") {
		t.Errorf("Expected last session 'self/proj-5' to be visible, got: %s", viewBot)
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
			AgentType:    core.AgentTypeAntigravity,
			SessionID:    "session-alpha-12345678",
			WorkspaceDir: "/Users/test/Documents/self/project-a",
			ShortPath:    "self/project-a",
			InitialGoal:  "Create a new microservice",
			LastPrompt:   "Add unit tests",
			StepCount:    100,
			LastModified: time.Now(),
		},
		{
			AgentType:    core.AgentTypeAntigravity,
			SessionID:    "session-beta-87654321",
			WorkspaceDir: "/Users/test/Documents/self/project-b",
			ShortPath:    "self/project-b",
			InitialGoal:  "Fix database deadlock bug",
			LastPrompt:   "Verify WAL mode",
			StepCount:    50,
			LastModified: time.Now().Add(-1 * time.Hour),
		},
		{
			AgentType:    core.AgentTypeClaudeCode,
			SessionID:    "session-claude-999999",
			WorkspaceDir: "/Users/test/Documents/self/claude-app",
			ShortPath:    "self/claude-app",
			InitialGoal:  "Refactor React frontend",
			LastPrompt:   "Update Tailwind styles",
			StepCount:    10,
			LastModified: time.Now().Add(-2 * time.Hour),
		},
	}
	m.selectedAgentTab = core.AgentTypeAntigravity
	m.filteredSessions = filterSessions(m.availableSessions, "", m.selectedAgentTab)
	m.switcherSelectedIdx = 0

	// Initial AGY tab should only match 2 sessions
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

	// 3. Test Agent Tab Cycle with Tab key
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.selectedAgentTab != core.AgentTypeClaudeCode {
		t.Errorf("Expected selectedAgentTab=core.AgentTypeClaudeCode after Tab, got %s", m.selectedAgentTab)
	}
	if len(m.filteredSessions) != 1 || m.filteredSessions[0].SessionID != "session-claude-999999" {
		t.Fatalf("Expected 1 Claude session, got %d", len(m.filteredSessions))
	}

	// 4. Test Agent Tab Cycle back with Shift+Tab
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.selectedAgentTab != core.AgentTypeAntigravity {
		t.Errorf("Expected selectedAgentTab=core.AgentTypeAntigravity after Shift+Tab, got %s", m.selectedAgentTab)
	}

	// 5. Test Typing Filter for path or prompt keywords
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("deadlock")})
	m = updated.(Model)
	if len(m.filteredSessions) != 1 || m.filteredSessions[0].SessionID != "session-beta-87654321" {
		t.Fatalf("Expected 1 filtered session matching 'deadlock', got %d", len(m.filteredSessions))
	}

	// 6. Test Backspace
	for i := 0; i < 8; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = updated.(Model)
	}
	if len(m.filteredSessions) != 2 {
		t.Errorf("Expected 2 sessions restored after backspace, got %d", len(m.filteredSessions))
	}

	// 7. Test Dual-Pane Rendering Output (Zero Emojis & Inspector content)
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
	if !strings.Contains(view, "self/project-a") {
		t.Errorf("Expected 'self/project-a' short path in view, got: %s", view)
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
		{"/Users/daniel_y_yang/Documents/self/ithome2026", "self/ithome2026"},
		{"/Users/daniel_y_yang/Documents/self/bookkeeper", "self/bookkeeper"},
		{"/Users/daniel_y_yang/.gemini/antigravity-cli", ".gemini/antigravity-cli"},
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
			Tokens: core.TokenBreakdown{
				TotalTokens:  1500,
				CachedTokens: 1200,
				CacheHitRate: 80.0,
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
	m := NewModel("aa726359-08e2-4687-a15c-073a2f4a705b", false)
	m.width = 120
	m.height = 30
	m.eventCount = 2200

	header := m.renderHeader()
	t.Logf("Header length: %d, string: %s", lipgloss.Width(header), header)

	m.activeView = ViewDashboard
	dash := m.renderDashboardView()
	dashLines := strings.Split(dash, "\n")
	for i, l := range dashLines {
		t.Logf("Dash line %d width: %d | %s", i, lipgloss.Width(l), l)
		if i > 3 {
			break
		}
	}

	m.activeView = ViewHistory
	hist := m.renderHistoryView()
	histLines := strings.Split(hist, "\n")
	for i, l := range histLines {
		t.Logf("Hist line %d width: %d | %s", i, lipgloss.Width(l), l)
		if i > 3 {
			break
		}
	}
}

