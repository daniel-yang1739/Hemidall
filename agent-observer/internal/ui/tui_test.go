package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		if !strings.Contains(footerLine, "Help") && !strings.Contains(footerLine, "Quit") && !strings.Contains(footerLine, "Focus") {
			t.Errorf("[%dx%d] Expected footer on line %d, got: %s", size.w, size.h, size.h-1, footerLine)
		}
	}
}

func TestView3HelpAndDocsRenderingAndSearch(t *testing.T) {
	m := NewModel("test-session", false)
	m.width = 100
	m.height = 30

	// 1. Press '3' or '?' to switch to View 3
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = updated.(Model)
	if m.activeView != ViewHelp {
		t.Fatalf("Expected activeView=ViewHelp, got %v", m.activeView)
	}

	view := m.View()
	if !strings.Contains(view, "HELP & ARCHITECTURE GLOSSARY") {
		t.Errorf("Expected 'HELP & ARCHITECTURE GLOSSARY' in view, got: %s", view)
	}
	if !strings.Contains(view, "Global Navigation") {
		t.Errorf("Expected 'Global Navigation' in view, got: %s", view)
	}

	// 2. Press '/' to activate search mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	if !m.isHelpSearching {
		t.Error("Expected isHelpSearching=true after pressing '/'")
	}

	// 3. Type query "cot" to search
	for _, r := range "cot" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if m.helpSearchQuery != "cot" {
		t.Errorf("Expected helpSearchQuery='cot', got '%s'", m.helpSearchQuery)
	}

	filteredView := m.View()
	if !strings.Contains(filteredView, "Active Turn / CoT") {
		t.Errorf("Expected filtered view to contain 'Active Turn / CoT', got: %s", filteredView)
	}

	// 4. Type query "accumulated" to search
	m.helpSearchQuery = ""
	for _, r := range "accumulated" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	filteredView2 := m.View()
	if !strings.Contains(filteredView2, "Raw Log Accumulated") {
		t.Errorf("Expected filtered view to contain 'Raw Log Accumulated', got: %s", filteredView2)
	}

	// 5. Press Esc to clear search
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.helpSearchQuery != "" {
		t.Errorf("Expected helpSearchQuery to be cleared, got '%s'", m.helpSearchQuery)
	}
}

func TestView3ZeroHeightVariationAcrossSizes(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}, {140, 40}} {
		m := NewModel("test-session", false)
		m.width = size.w
		m.height = size.h
		m.activeView = ViewHelp

		view := m.View()
		lines := strings.Split(view, "\n")

		if len(lines) != size.h {
			t.Fatalf("[%dx%d] ViewHelp expected exactly %d lines, got %d", size.w, size.h, size.h, len(lines))
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
	if !strings.Contains(view, "SWITCH SESSION") {
		t.Errorf("Expected modal title 'SWITCH SESSION' in view, got: %s", view)
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
	m.width = 80
	m.height = 24

	// Mock available sessions
	m.availableSessions = []antigravity.SessionInfo{
		{SessionID: "session-alpha", StepCount: 100, LastModified: time.Now()},
		{SessionID: "session-beta", StepCount: 50, LastModified: time.Now().Add(-1 * time.Hour)},
		{SessionID: "session-gamma", StepCount: 10, LastModified: time.Now().Add(-2 * time.Hour)},
	}
	m.filteredSessions = m.availableSessions
	m.switcherSelectedIdx = 0

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

	// 3. Test Typing Filter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("gamma")})
	m = updated.(Model)
	if len(m.filteredSessions) != 1 || m.filteredSessions[0].SessionID != "session-gamma" {
		t.Fatalf("Expected 1 filtered session 'session-gamma', got %d", len(m.filteredSessions))
	}

	// 4. Test Backspace
	for i := 0; i < 5; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = updated.(Model)
	}
	if len(m.filteredSessions) != 3 {
		t.Errorf("Expected 3 sessions restored after backspace, got %d", len(m.filteredSessions))
	}

	// 5. Test Escape Key (Cancel without switching)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.isSessionSwitcherOpen {
		t.Error("Expected Esc key to close switcher modal")
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
