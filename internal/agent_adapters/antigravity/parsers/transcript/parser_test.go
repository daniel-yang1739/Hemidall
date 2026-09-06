package transcript

import (
	"strings"
	"testing"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/core"
)

func TestTranscriptParser_Parse_PreservesValidStepsAndReportsMalformedLines(t *testing.T) {
	testCases := []struct {
		name                string
		input               string
		expectedStepCount   int
		expectedDiagnostics int
	}{
		{
			name:                "valid transcript",
			input:               `{"step_index":7,"source":"USER","type":"USER_INPUT","status":"DONE","created_at":"2026-09-01T01:02:03Z","content":"hello"}` + "\n",
			expectedStepCount:   1,
			expectedDiagnostics: 0,
		},
		{
			name:                "mixed valid and malformed transcript",
			input:               `{"step_index":7,"source":"USER","type":"USER_INPUT","status":"DONE","created_at":"2026-09-01T01:02:03Z","content":"hello"}` + "\n" + `{not json}` + "\n",
			expectedStepCount:   1,
			expectedDiagnostics: 1,
		},
		{
			name:                "invalid timestamp",
			input:               `{"step_index":7,"source":"USER","type":"USER_INPUT","status":"DONE","created_at":"not-a-time","content":"hello"}` + "\n",
			expectedStepCount:   0,
			expectedDiagnostics: 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			steps, diagnostics := (TranscriptParser{}).Parse(strings.NewReader(testCase.input), testSource())
			requireEqual(t, len(steps), testCase.expectedStepCount)
			requireEqual(t, len(diagnostics), testCase.expectedDiagnostics)
		})
	}
}

func TestParseTranscriptLine_Pos_NormalizesPlannerResponseToCloudModelResponse(t *testing.T) {
	step, parseErr := ParseTranscriptLine(`{"step_index":8,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-09-01T01:02:03Z","content":"world"}`, testSource())

	requireEqual(t, parseErr == nil, true)
	requireEqual(t, step.Kind, string(core.StepTypeModelResponse))
}

func TestParseTranscriptLine_Pos_NormalizesPlannerResponseWithToolCallsToCloudToolCall(t *testing.T) {
	step, parseErr := ParseTranscriptLine(`{"step_index":9,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-09-01T01:02:03Z","content":"executing","tool_calls":[{"name":"run_command","args":{"command":"ls"}}]}`, testSource())

	requireEqual(t, parseErr == nil, true)
	requireEqual(t, step.Kind, string(core.StepTypeToolCall))
}

func TestParseTranscriptFragment_PreservesTrailingIncompleteLine(t *testing.T) {
	firstFragment := `{"step_index":7,"source":"USER","type":"USER_INPUT","status":"DONE","created_at":"2026-09-01T01:02:03Z","content":"hello"}` + "\n" + `{"step_index":8`
	firstSteps, firstDiagnostics, trailingFragment := ParseTranscriptFragment(firstFragment, testSource())
	secondSteps, secondDiagnostics, finalTrailingFragment := ParseTranscriptFragment(trailingFragment+`,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-09-01T01:02:03Z","content":"world"}`+"\n", testSource())

	requireEqual(t, len(firstSteps), 1)
	requireEqual(t, len(firstDiagnostics), 0)
	requireEqual(t, len(secondSteps), 1)
	requireEqual(t, len(secondDiagnostics), 0)
	requireEqual(t, finalTrailingFragment, "")
	requireEqual(t, secondSteps[0].Index, 8)
}

func testSource() agents.SourceRef {
	return agents.SourceRef{Kind: agents.SourceKindTranscript, Path: "/fixture/transcript_full.jsonl"}
}

func requireEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
