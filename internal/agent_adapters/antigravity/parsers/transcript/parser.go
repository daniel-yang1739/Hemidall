package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/core"
)

const (
	transcriptScannerBufferBytes = 1_024 * 1_024
	transcriptSourceName         = "transcript"
	antigravityPlannerResponse   = "PLANNER_RESPONSE"
)

// RawTranscriptLine is the observed Antigravity JSONL shape needed by the
// transcript parser. Unknown JSON fields intentionally remain ignored.
type RawTranscriptLine struct {
	StepIndex int           `json:"step_index"`
	Source    string        `json:"source"`
	Type      string        `json:"type"`
	Status    string        `json:"status"`
	CreatedAt string        `json:"created_at"`
	Content   string        `json:"content"`
	Thinking  string        `json:"thinking"`
	ToolCalls []RawToolCall `json:"tool_calls"`
}

// RawToolCall is the transcript representation of one tool invocation.
type RawToolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// TranscriptParser converts transcript JSONL into source-neutral step facts.
type TranscriptParser struct{}

// Parse reads all complete JSONL lines from reader. Malformed lines are returned
// as diagnostics while valid lines remain available to the caller.
func (TranscriptParser) Parse(reader io.Reader, source agents.SourceRef) ([]agents.Step, []error) {
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, transcriptScannerBufferBytes)
	scanner.Buffer(buffer, transcriptScannerBufferBytes)
	steps := make([]agents.Step, 0)
	diagnostics := make([]error, 0)
	for scanner.Scan() {
		step, err := ParseTranscriptLine(scanner.Text(), source)
		if err != nil {
			diagnostics = append(diagnostics, err)
			continue
		}
		steps = append(steps, step)
	}
	if err := scanner.Err(); err != nil {
		diagnostics = append(diagnostics, fmt.Errorf("scan transcript: %w", err))
	}
	return steps, diagnostics
}

// ParseTranscriptLine converts one JSONL record without reading external state.
func ParseTranscriptLine(line string, source agents.SourceRef) (agents.Step, error) {
	var raw RawTranscriptLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return agents.Step{}, fmt.Errorf("decode transcript line: %w", err)
	}
	parsedTime, err := time.Parse(time.RFC3339, raw.CreatedAt)
	if err != nil {
		return agents.Step{}, fmt.Errorf("parse transcript timestamp: %w", err)
	}
	toolCalls := make([]agents.ToolCall, len(raw.ToolCalls))
	for index, toolCall := range raw.ToolCalls {
		toolCalls[index] = agents.ToolCall{Name: toolCall.Name, Args: toolCall.Args}
	}
	kind := canonicalStepKind(raw.Type)
	if raw.Type == antigravityPlannerResponse && len(toolCalls) > 0 {
		kind = string(core.StepTypeToolCall)
	}

	return agents.Step{
		Index:         raw.StepIndex,
		Timestamp:     parsedTime,
		Source:        raw.Source,
		Kind:          kind,
		Status:        raw.Status,
		Content:       raw.Content,
		ContentSource: agents.SourceKindTranscript,
		Thinking:      raw.Thinking,
		ToolCalls:     toolCalls,
		Evidence: []agents.Evidence{{
			Level:   agents.EvidenceObservedOnly,
			Source:  source,
			Locator: fmt.Sprintf("step_index=%d", raw.StepIndex),
			Note:    transcriptSourceName,
		}},
	}, nil
}

// canonicalStepKind converts observed Antigravity wire labels into the
// provider-neutral StepType vocabulary before lifecycle assembly runs.
func canonicalStepKind(observedKind string) string {
	if observedKind == antigravityPlannerResponse {
		return string(core.StepTypeModelResponse)
	}
	return observedKind
}

// ParseTranscriptFragment parses complete newline-delimited records and returns
// an incomplete trailing fragment for the next incremental refresh.
func ParseTranscriptFragment(fragment string, source agents.SourceRef) ([]agents.Step, []error, string) {
	lines := strings.Split(fragment, "\n")
	trailingFragment := lines[len(lines)-1]
	completeLines := lines[:len(lines)-1]
	steps := make([]agents.Step, 0, len(completeLines))
	diagnostics := make([]error, 0)
	for _, line := range completeLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		step, err := ParseTranscriptLine(line, source)
		if err != nil {
			diagnostics = append(diagnostics, err)
			continue
		}
		steps = append(steps, step)
	}
	return steps, diagnostics, trailingFragment
}
