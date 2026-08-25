package antigravity

import (
	"testing"

	"agent-observer/internal/core"
)

func TestParseLineUserInput(t *testing.T) {
	watcher := NewWatcher("test.jsonl", "test-session", nil)
	line := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","created_at":"2026-08-19T01:00:00Z","content":"<USER_REQUEST>你好！請幫我寫 Code</USER_REQUEST>"}`

	event, err := watcher.parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.StepIndex != 0 {
		t.Errorf("expected step_index 0, got %d", event.StepIndex)
	}
	if event.Type != core.StepTypeUserInput {
		t.Errorf("expected type USER_INPUT, got %s", event.Type)
	}
	if event.Status != "DONE" {
		t.Errorf("expected status DONE, got %s", event.Status)
	}
	if event.Summary != "👤 User: 你好！請幫我寫 Code" {
		t.Errorf("unexpected summary: %s", event.Summary)
	}
}

func TestParseLinePlannerWithToolCalls(t *testing.T) {
	watcher := NewWatcher("test.jsonl", "test-session", nil)
	line := `{"step_index":2,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-08-19T01:00:05Z","content":"Let me check files","tool_calls":[{"name":"list_dir","args":{"path":"/workspace"}}]}`

	event, err := watcher.parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.StepIndex != 2 {
		t.Errorf("expected step_index 2, got %d", event.StepIndex)
	}
	if event.Type != core.StepTypeToolCall {
		t.Errorf("expected type TOOL_CALL, got %s", event.Type)
	}
	if len(event.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(event.ToolCalls))
	}
	if event.ToolCalls[0].ToolName != "list_dir" {
		t.Errorf("expected tool name list_dir, got %s", event.ToolCalls[0].ToolName)
	}
}
