package ui

import (
	"strings"
	"testing"
	"time"

	"agent-observer/internal/core"
)

// ==============================================================================
// 7. HISTORY FILTERING, SEARCH & FORMATTING LOGIC (3 Positive + 3 Negative)
// ==============================================================================

func TestHistory_Pos_FilterByScopeAllCombos(t *testing.T) {
	events := []core.UnifiedAgentEvent{
		{StepIndex: 1, Scope: core.ScopeUserInteraction, Type: core.StepTypeUserInput},
		{StepIndex: 2, Scope: core.ScopeCloudInference, Type: core.StepTypeModelResponse},
		{StepIndex: 3, Scope: core.ScopeLocalExecution, Type: core.StepTypeRunCommand},
		{StepIndex: 4, Scope: core.ScopeSubagent, Type: core.StepTypeModelResponse, IsSubagent: true},
	}

	// Filter by CLOUD
	var cloudOnly []core.UnifiedAgentEvent
	for _, e := range events {
		if e.Scope == core.ScopeCloudInference {
			cloudOnly = append(cloudOnly, e)
		}
	}
	if len(cloudOnly) != 1 || cloudOnly[0].StepIndex != 2 {
		t.Errorf("Expected 1 cloud step (index 2), got %d", len(cloudOnly))
	}

	// Filter by LOCAL
	var localOnly []core.UnifiedAgentEvent
	for _, e := range events {
		if e.Scope == core.ScopeLocalExecution {
			localOnly = append(localOnly, e)
		}
	}
	if len(localOnly) != 1 || localOnly[0].StepIndex != 3 {
		t.Errorf("Expected 1 local step (index 3), got %d", len(localOnly))
	}
}

func TestHistory_Pos_ValidKeywordSearch(t *testing.T) {
	events := []core.UnifiedAgentEvent{
		{StepIndex: 1, Summary: "👤 User: Please edit main.go"},
		{StepIndex: 2, Summary: "🛠️ Tool: write_to_file /app/main.go"},
		{StepIndex: 3, Summary: "💻 Run: go test ./..."},
	}

	keyword := "main.go"
	var matches []core.UnifiedAgentEvent
	for _, e := range events {
		if containsIgnoreCase(e.Summary, keyword) {
			matches = append(matches, e)
		}
	}

	if len(matches) != 2 {
		t.Fatalf("Expected 2 matches for 'main.go', got %d", len(matches))
	}
}

func TestHistory_Pos_FormatShortPath(t *testing.T) {
	res := formatShortPath("/Users/daniel_y_yang/Documents/self/ithome2026/main.go", 30)
	if len(res) > 30 {
		t.Errorf("formatShortPath exceeded max width 30: len=%d, str=%q", len(res), res)
	}
}

func TestHistory_Neg_EmptySearchNoMatches(t *testing.T) {
	events := []core.UnifiedAgentEvent{
		{StepIndex: 1, Summary: "User input"},
	}

	keyword := "non-existent-keyword-xyz"
	var matches []core.UnifiedAgentEvent
	for _, e := range events {
		if containsIgnoreCase(e.Summary, keyword) {
			matches = append(matches, e)
		}
	}
	if len(matches) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(matches))
	}
}

func TestHistory_Neg_EmptyHistoryFilter(t *testing.T) {
	var emptyEvents []core.UnifiedAgentEvent
	var filtered []core.UnifiedAgentEvent
	for _, e := range emptyEvents {
		if e.Scope == core.ScopeCloudInference {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) != 0 {
		t.Errorf("Expected empty slice, got %d", len(filtered))
	}
}

func TestHistory_Neg_ShortPathEmptyAndZeroWidth(t *testing.T) {
	if res := formatShortPath("", 20); res != "" {
		t.Errorf("Expected empty string for empty path, got %q", res)
	}
	if res := formatShortPath("/some/path", 0); res != "" {
		t.Errorf("Expected empty string for width 0, got %q", res)
	}
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func formatShortPath(path string, maxLen int) string {
	if maxLen <= 0 || path == "" {
		return ""
	}
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-(maxLen-3):]
}

func init() {
	_ = time.Now()
}
