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

// RawTranscriptLine matches the JSONL schema from ~/.gemini/.../transcript_full.jsonl
type RawTranscriptLine struct {
	StepIndex int                      `json:"step_index"`
	Source    string                   `json:"source"`
	Type      string                   `json:"type"`
	Status    string                   `json:"status"`
	CreatedAt string                   `json:"created_at"`
	Content   string                   `json:"content"`
	Thinking  string                   `json:"thinking,omitempty"`
	ToolCalls []RawToolCall            `json:"tool_calls,omitempty"`
}

type RawToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// Watcher monitors the transcript_full.jsonl file and local SQLite database
type Watcher struct {
	filePath     string
	sessionID    string
	analyzer     *core.PayloadAnalyzer
	sqliteReader *SQLiteTelemetryReader
	lastStepIdx  int
}

// NewWatcher creates a new Antigravity log and telemetry file watcher
func NewWatcher(filePath string, sessionID string, analyzer *core.PayloadAnalyzer, sqlitePath string) *Watcher {
	var sqliteReader *SQLiteTelemetryReader
	if sqlitePath != "" {
		sqliteReader = NewSQLiteTelemetryReader(sqlitePath)
	}

	return &Watcher{
		filePath:     filePath,
		sessionID:    sessionID,
		analyzer:     analyzer,
		sqliteReader: sqliteReader,
		lastStepIdx:  -1,
	}
}

// Name returns the adapter identifier
func (w *Watcher) Name() string {
	return "antigravity"
}

// Start opens the transcript file, preloads history into channel, and monitors tail
func (w *Watcher) Start(ctx context.Context, out chan<- core.UnifiedAgentEvent) error {
	file, err := os.Open(w.filePath)
	if err != nil {
		return fmt.Errorf("failed to open transcript file %s: %w", w.filePath, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// 1. Initial SQLite polling to connect official telemetry
	if w.sqliteReader != nil {
		_ = w.sqliteReader.PollLatest()
	}

	// 2. Warmup: fast-forward existing history and push to out channel so TUI receives all historical events
	warmupCount := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					w.warmupLine(line, ctx, out)
					warmupCount++
				}
				break
			}
			return fmt.Errorf("error reading initial lines: %w", err)
		}
		w.warmupLine(line, ctx, out)
		warmupCount++
	}

	// 3. Poll file tail and SQLite periodically for new live steps
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if w.sqliteReader != nil {
				_ = w.sqliteReader.PollLatest()
			}

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

func (w *Watcher) warmupLine(line string, ctx context.Context, out chan<- core.UnifiedAgentEvent) {
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

	if w.analyzer != nil {
		w.analyzer.AnalyzeStep(&event)
	}

	select {
	case out <- event:
	case <-ctx.Done():
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

	if event.StepIndex <= w.lastStepIdx && event.StepIndex != 0 {
		return
	}
	w.lastStepIdx = event.StepIndex

	select {
	case out <- event:
	case <-ctx.Done():
	}
}

func (w *Watcher) parseLine(line string) (core.UnifiedAgentEvent, error) {
	var raw RawTranscriptLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return core.UnifiedAgentEvent{}, err
	}

	t, err := time.Parse(time.RFC3339, raw.CreatedAt)
	if err != nil {
		t = time.Now()
	}

	var stepType core.StepType
	var summary string

	switch raw.Type {
	case "USER_INPUT":
		stepType = core.StepTypeUserInput
		clean := strings.TrimPrefix(raw.Content, "<USER_REQUEST>")
		clean = strings.TrimSuffix(clean, "</USER_REQUEST>")
		summary = fmt.Sprintf("👤 User: %s", truncate(clean, 60))
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

	// Check if official telemetry from SQLite is available for this step
	if w.sqliteReader != nil {
		if meta := w.sqliteReader.GetTelemetryForStep(raw.StepIndex); meta != nil && meta.TotalTokens > 0 {
			event.Tokens.IsOfficialData = true
			event.Tokens.TotalTokens = meta.TotalTokens
			event.Tokens.CachedTokens = meta.CachedTokens
			event.Tokens.NewTokens = meta.TotalTokens - meta.CachedTokens
			event.Tokens.CacheHitRate = meta.CacheHitRate
			event.Tokens.OfficialModel = meta.ModelName
			event.Tokens.OfficialContextLimit = meta.ContextLimit
		} else if raw.Type == "PLANNER_RESPONSE" {
			if latest := w.sqliteReader.GetLatestTelemetry(); latest != nil && latest.TotalTokens > 0 {
				event.Tokens.IsOfficialData = true
				event.Tokens.TotalTokens = latest.TotalTokens
				event.Tokens.CachedTokens = latest.CachedTokens
				event.Tokens.NewTokens = latest.TotalTokens - latest.CachedTokens
				event.Tokens.CacheHitRate = latest.CacheHitRate
				event.Tokens.OfficialModel = latest.ModelName
				event.Tokens.OfficialContextLimit = latest.ContextLimit
			}
		}
	}

	return event, nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
