package antigravity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestWatcher_Boundary_PendingCloudEventsStayBounded(t *testing.T) {
	watcher := NewWatcher("test.jsonl", "test-session", nil, "")

	rememberPendingCloudRange(watcher, watcherPendingCloudEventLimit+1)

	requirePendingCloudCount(t, watcher, watcherPendingCloudEventLimit)
	requirePendingCloudAbsent(t, watcher, 0)
	requirePendingCloudPresent(t, watcher, watcherPendingCloudEventLimit)
}

func TestReadRecentTranscriptLines_Pos_ReturnsChronologicalTail(t *testing.T) {
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	requireAntigravityNoError(t, os.WriteFile(transcriptPath, []byte(buildTranscriptJSONL(watcherPreviewEventLimit+1)), fixtureFilePermissions))

	lines, readErr := readRecentTranscriptLines(transcriptPath, watcherPreviewEventLimit)

	requireAntigravityNoError(t, readErr)
	requireAntigravityEqual(t, watcherPreviewEventLimit, len(lines))
	requireAntigravityEqual(t, transcriptJSONLine(1), lines[0])
	requireAntigravityEqual(t, transcriptJSONLine(watcherPreviewEventLimit), lines[watcherPreviewEventLimit-1])
}

func TestReadRecentTranscriptLines_Boundary_DiscardsPartialLeadingLine(t *testing.T) {
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	largePrefix := strings.Repeat("x", watcherPreviewReadBlockBytes)
	content := largePrefix + "\n" + buildTranscriptJSONL(watcherPreviewEventLimit+1)
	requireAntigravityNoError(t, os.WriteFile(transcriptPath, []byte(content), fixtureFilePermissions))

	lines, readErr := readRecentTranscriptLines(transcriptPath, watcherPreviewEventLimit)

	requireAntigravityNoError(t, readErr)
	requireAntigravityEqual(t, watcherPreviewEventLimit, len(lines))
	requireAntigravityEqual(t, transcriptJSONLine(1), lines[0])
	requireAntigravityEqual(t, transcriptJSONLine(watcherPreviewEventLimit), lines[watcherPreviewEventLimit-1])
}

func TestWatcherPreview_Pos_DoesNotAdvanceCanonicalReplayState(t *testing.T) {
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	requireAntigravityNoError(t, os.WriteFile(transcriptPath, []byte(buildTranscriptJSONL(watcherPreviewEventLimit+1)), fixtureFilePermissions))
	watcher := NewWatcher(transcriptPath, "test-session", core.NewPayloadAnalyzer(), "")
	output := make(chan core.UnifiedAgentEvent, watcherPreviewEventLimit)

	requireAntigravityNoError(t, watcher.emitRecentPreview(context.Background(), output))

	requireAntigravityEqual(t, -1, watcher.lastStepIdx)
	requireAntigravityEqual(t, watcherPreviewEventLimit, len(output))
	preview := <-output
	requireAntigravityEqual(t, 1, preview.StepIndex)
	requireAntigravityEqual(t, 0, preview.Tokens.StepDelta)
}

func TestWatcher_Pos_ReEmitsStatusUpdateForSameTranscriptStep(t *testing.T) {
	watcher := NewWatcher("test.jsonl", "test-session", core.NewPayloadAnalyzer(), "")
	output := make(chan core.UnifiedAgentEvent, 2)
	runningLine := `{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"RUNNING","created_at":"2026-08-19T01:00:01Z","content":"working"}`
	doneLine := `{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-08-19T01:00:02Z","content":"finished"}`

	watcher.processLine(runningLine, context.Background(), output)
	watcher.processLine(doneLine, context.Background(), output)
	first := <-output
	second := <-output

	requireAntigravityEqual(t, "RUNNING", first.Status)
	requireAntigravityEqual(t, "DONE", second.Status)
	requireAntigravityEqual(t, 1, watcher.lastStepIdx)
	requireAntigravityGreater(t, first.Tokens.StepDelta, 0)
	requireAntigravityEqual(t, 0, second.Tokens.StepDelta)
}

func rememberPendingCloudRange(watcher *Watcher, count int) {
	for stepIndex := 0; stepIndex < count; stepIndex++ {
		watcher.rememberPendingCloud(core.UnifiedAgentEvent{StepIndex: stepIndex})
	}
}

func buildTranscriptJSONL(count int) string {
	lines := make([]string, 0, count)
	for stepIndex := 0; stepIndex < count; stepIndex++ {
		lines = append(lines, transcriptJSONLine(stepIndex))
	}
	return strings.Join(lines, "\n") + "\n"
}

func transcriptJSONLine(stepIndex int) string {
	return fmt.Sprintf(`{"step_index":%d,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-08-19T01:00:00Z","content":"step %d"}`, stepIndex, stepIndex)
}

func requirePendingCloudCount(t *testing.T, watcher *Watcher, expected int) {
	t.Helper()
	if len(watcher.pendingCloud) != expected {
		t.Fatalf("pending cloud count: got %d, want %d", len(watcher.pendingCloud), expected)
	}
}

func requirePendingCloudAbsent(t *testing.T, watcher *Watcher, stepIndex int) {
	t.Helper()
	if _, exists := watcher.pendingCloud[stepIndex]; exists {
		t.Fatalf("pending cloud event %d should be evicted", stepIndex)
	}
}

func requirePendingCloudPresent(t *testing.T, watcher *Watcher, stepIndex int) {
	t.Helper()
	if _, exists := watcher.pendingCloud[stepIndex]; !exists {
		t.Fatalf("pending cloud event %d should remain", stepIndex)
	}
}
