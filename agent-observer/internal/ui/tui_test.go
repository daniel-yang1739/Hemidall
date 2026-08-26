package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"agent-observer/internal/core"
)

func TestCJKAndLongPayloadZeroHeightVariation(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 30}, {120, 35}, {140, 40}} {
		m := NewModel("test-session", false)
		m.width = size.w
		m.height = size.h

		// Create steps with heavy Traditional Chinese text, emojis, and long payloads
		for i := 0; i < 25; i++ {
			chineseSummary := "【實測連線】這是一個非常長的繁體中文步驟說明文字，包含特殊符號 🌟 🚀 與長度測試"
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
		if !strings.Contains(footerLine, "Dashboard") && !strings.Contains(footerLine, "Quit") && !strings.Contains(footerLine, "Focus") {
			t.Errorf("[%dx%d] Expected footer on line %d, got: %s", size.w, size.h, size.h-1, footerLine)
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
	if !strings.Contains(view, "Switch Session") {
		t.Errorf("Expected modal title 'Switch Session' in view, got: %s", view)
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
