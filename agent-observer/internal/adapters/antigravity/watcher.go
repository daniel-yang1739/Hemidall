package antigravity

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"agent-observer/internal/core"
)

// RawTranscriptLine represents a single JSON record in transcript_full.jsonl
type RawTranscriptLine struct {
	StepIndex int           `json:"step_index"`
	Source    string        `json:"source"`
	Type      string        `json:"type"`
	Status    string        `json:"status"`
	CreatedAt string        `json:"created_at"`
	Content   string        `json:"content,omitempty"`
	Thinking  string        `json:"thinking,omitempty"`
	ToolCalls []RawToolCall `json:"tool_calls,omitempty"`
}

// RawToolCall represents individual tool call payloads in transcript
type RawToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// Watcher implements the adapters.AgentAdapter interface for Antigravity CLI logs
type Watcher struct {
	filePath    string
	sessionID   string
	lastStepIdx int
	analyzer    *core.PayloadAnalyzer
}

// NewWatcher creates an Antigravity Transcript Watcher
func NewWatcher(filePath string, sessionID string, analyzer *core.PayloadAnalyzer) *Watcher {
	return &Watcher{
		filePath:    filePath,
		sessionID:   sessionID,
		lastStepIdx: -1,
		analyzer:    analyzer,
	}
}

func (w *Watcher) Name() string {
	return "antigravity"
}

// Start begins tailing transcript_full.jsonl and streams new events
func (w *Watcher) Start(ctx context.Context, out chan<- core.UnifiedAgentEvent) error {
	file, err := os.Open(w.filePath)
	if err != nil {
		return fmt.Errorf("failed to open transcript file %s: %w", w.filePath, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// 1. Silent Warmup: fast-forward existing history to initialize context token state without flooding output
	warmupCount := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					w.warmupLine(line)
					warmupCount++
				}
				break
			}
			return fmt.Errorf("error reading initial lines: %w", err)
		}
		w.warmupLine(line)
		warmupCount++
	}

	fmt.Printf("✅ History warm-up complete! Pre-loaded %d steps. Watching for LIVE events...\n\n", warmupCount)

	// 2. Poll file tail periodically for new live steps
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF && len(line) > 0 {
						w.handleLiveLine(line, ctx, out)
					}
					break
				}
				w.handleLiveLine(line, ctx, out)
			}
		}
	}
}

func (w *Watcher) warmupLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	event, err := w.parseLine(trimmed)
	if err != nil {
		return
	}
	if event.StepIndex <= w.lastStepIdx && event.StepIndex != 0 {
		return
	}
	w.lastStepIdx = event.StepIndex

	// Quietly initialize context state baseline
	if w.analyzer != nil {
		w.analyzer.AnalyzeStep(&event)
	}
}

func (w *Watcher) handleLiveLine(line string, ctx context.Context, out chan<- core.UnifiedAgentEvent) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	event, err := w.parseLine(trimmed)
	if err != nil {
		return
	}

	// Deduplication protection
	if event.StepIndex <= w.lastStepIdx && event.StepIndex != 0 {
		return
	}
	w.lastStepIdx = event.StepIndex

	select {
	case out <- event:
	case <-ctx.Done():
	}
}

// parseLine converts raw Antigravity JSON into domain UnifiedAgentEvent
func (w *Watcher) parseLine(line string) (core.UnifiedAgentEvent, error) {
	var raw RawTranscriptLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return core.UnifiedAgentEvent{}, err
	}

	t, err := time.Parse(time.RFC3339, raw.CreatedAt)
	if err != nil {
		t = time.Now()
	}

	stepType := core.StepTypeUnknown
	var summary string

	switch raw.Type {
	case "USER_INPUT":
		stepType = core.StepTypeUserInput
		cleanContent := cleanUserRequest(raw.Content)
		summary = fmt.Sprintf("👤 User: %s", truncate(cleanContent, 60))
	case "PLANNER_RESPONSE":
		if len(raw.ToolCalls) > 0 {
			stepType = core.StepTypeToolCall
			var toolNames []string
			for _, tc := range raw.ToolCalls {
				toolNames = append(toolNames, tc.Name)
			}
			summary = fmt.Sprintf("🛠️  Model Tool Call: [%s]", strings.Join(toolNames, ", "))
		} else {
			stepType = core.StepTypeModelResponse
			summary = fmt.Sprintf("🤖 Model: %s", truncate(raw.Content, 60))
		}
	case "CONVERSATION_HISTORY":
		stepType = core.StepTypeSystemInit
		summary = "📜 System: Loaded Conversation History"
	case "RUN_COMMAND":
		stepType = core.StepTypeRunCommand
		summary = fmt.Sprintf("💻 Terminal Output: %s", truncate(raw.Content, 50))
	case "VIEW_FILE":
		stepType = core.StepTypeViewFile
		summary = fmt.Sprintf("📄 File Content Read: %s", truncate(raw.Content, 50))
	case "CODE_ACTION":
		stepType = core.StepTypeCodeAction
		summary = fmt.Sprintf("✏️  Code Action Diff: %s", truncate(raw.Content, 50))
	case "LIST_DIRECTORY":
		stepType = core.StepTypeListDirectory
		summary = fmt.Sprintf("📁 Directory Listing: %s", truncate(raw.Content, 50))
	case "ASK_QUESTION":
		stepType = core.StepTypeAskQuestion
		summary = fmt.Sprintf("❓ Interactive Question: %s", truncate(raw.Content, 50))
	default:
		stepType = core.StepType(raw.Type)
		summary = fmt.Sprintf("⚙️  %s (%s)", raw.Type, raw.Source)
	}

	var toolCalls []core.ToolCallInfo
	for _, tc := range raw.ToolCalls {
		toolCalls = append(toolCalls, core.ToolCallInfo{
			ToolName:  tc.Name,
			Arguments: tc.Args,
		})
	}

	event := core.UnifiedAgentEvent{
		SessionID:   w.sessionID,
		StepIndex:   raw.StepIndex,
		Timestamp:   t,
		Source:      raw.Source,
		Type:        stepType,
		Status:      raw.Status,
		Summary:     summary,
		RawContent:  raw.Content,
		Thinking:    raw.Thinking,
		ToolCalls:   toolCalls,
		CacheStatus: "UNKNOWN",
	}

	return event, nil
}

func cleanUserRequest(s string) string {
	s = strings.ReplaceAll(s, "<USER_REQUEST>", "")
	s = strings.ReplaceAll(s, "</USER_REQUEST>", "")
	return strings.TrimSpace(s)
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
