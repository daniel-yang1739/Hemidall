package antigravity

import (
	"testing"

	"heimdall/internal/core"
)

func TestParseLineUserInput(t *testing.T) {
	watcher := NewWatcher("test.jsonl", "test-session", nil, "")
	line := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","created_at":"2026-08-19T01:00:00Z","content":"<USER_REQUEST>Hello! Please write code for me</USER_REQUEST>"}`

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
	if event.Summary != "👤 User: Hello! Please write code for me" {
		t.Errorf("unexpected summary: %s", event.Summary)
	}
}

func TestParseLinePlannerWithToolCalls(t *testing.T) {
	watcher := NewWatcher("test.jsonl", "test-session", nil, "")
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

func TestWatcher_Neg_DuplicateDoneNotificationsNoGrow(t *testing.T) {
	watcher := NewWatcher("test.jsonl", "test-session", nil, "")

	lineRunning := `{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"RUNNING","created_at":"2026-08-19T01:00:01Z","content":"Running task"}`
	lineDone := `{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-08-19T01:00:03Z","content":"Finished task"}`

	ev1, _ := watcher.parseLine(lineRunning)
	ev2, _ := watcher.parseLine(lineDone)

	if ev1.StepIndex != ev2.StepIndex {
		t.Errorf("Step indices should be identical (1), got %d and %d", ev1.StepIndex, ev2.StepIndex)
	}
	if ev2.Status != "DONE" {
		t.Errorf("Final status should be DONE, got %s", ev2.Status)
	}
}

