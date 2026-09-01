package antigravity

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"heimdall/internal/core"
)

const (
	watcherInitialReconcileInterval = 2 * time.Second
	watcherMaximumReconcileInterval = 30 * time.Second
	watcherReconcileBackoffFactor   = 2
	watcherEventDebounceInterval    = 100 * time.Millisecond
	watcherPendingCloudEventLimit   = 512
	watcherPreviewEventLimit        = 10
	watcherPreviewReadBlockBytes    = 64 * 1024
	persistedUsageObservationSource = "antigravity_gen_metadata_schema_inferred"
)

// RawTranscriptLine matches the JSONL schema from ~/.gemini/.../transcript_full.jsonl
type RawTranscriptLine struct {
	StepIndex int           `json:"step_index"`
	Source    string        `json:"source"`
	Type      string        `json:"type"`
	Status    string        `json:"status"`
	CreatedAt string        `json:"created_at"`
	Content   string        `json:"content"`
	Thinking  string        `json:"thinking,omitempty"`
	ToolCalls []RawToolCall `json:"tool_calls,omitempty"`
}

type RawToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// Watcher monitors the transcript_full.jsonl file and local SQLite database
type Watcher struct {
	filePath     string
	sqlitePath   string
	sessionID    string
	analyzer     *core.PayloadAnalyzer
	sqliteReader *SQLiteTelemetryReader
	tracker      *core.StepLinkageTracker
	lastStepIdx  int
	pendingLine  string
	pendingCloud map[int]core.UnifiedAgentEvent
	pendingOrder []int
}

// NewWatcher creates a new Antigravity log and telemetry file watcher
func NewWatcher(filePath string, sessionID string, analyzer *core.PayloadAnalyzer, sqlitePath string) *Watcher {
	var sqliteReader *SQLiteTelemetryReader
	if sqlitePath != "" {
		sqliteReader = NewSQLiteTelemetryReader(sqlitePath)
	}

	return &Watcher{
		filePath:     filePath,
		sqlitePath:   sqlitePath,
		sessionID:    sessionID,
		analyzer:     analyzer,
		sqliteReader: sqliteReader,
		tracker:      core.NewStepLinkageTracker(),
		lastStepIdx:  -1,
		pendingCloud: make(map[int]core.UnifiedAgentEvent),
	}
}

// Name returns the adapter identifier
func (w *Watcher) Name() string {
	return "antigravity"
}

// Start emits a small latest-event preview before chronological history hydration,
// then uses filesystem notifications for live changes. The preview makes the
// first frame useful without corrupting the analyzer or step-linkage state: the
// complete replay remains chronological and updates the same stable step keys.
// An adaptive reconciliation timer is retained only as a recovery path for
// missed operating-system notifications.
func (w *Watcher) Start(ctx context.Context, out chan<- core.UnifiedAgentEvent) error {
	file, err := os.Open(w.filePath)
	if err != nil {
		return fmt.Errorf("failed to open transcript file %s: %w", w.filePath, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	if err := w.emitRecentPreview(ctx, out); err != nil {
		return err
	}
	w.refreshTelemetry(ctx, out)
	if err := w.drainAvailableLines(reader, ctx, out); err != nil {
		return err
	}

	storageWatcher, watcherErr := w.openStorageWatcher()
	if storageWatcher != nil {
		defer storageWatcher.Close()
	}
	var storageEvents <-chan fsnotify.Event
	var storageErrors <-chan error
	if watcherErr == nil && storageWatcher != nil {
		storageEvents = storageWatcher.Events
		storageErrors = storageWatcher.Errors
	}

	reconcileInterval := watcherInitialReconcileInterval
	reconcileTimer := time.NewTimer(reconcileInterval)
	defer reconcileTimer.Stop()
	var debounceTimer *time.Timer
	var debounceEvents <-chan time.Time

	armDebounce := func() {
		if debounceTimer == nil {
			debounceTimer = time.NewTimer(watcherEventDebounceInterval)
			debounceEvents = debounceTimer.C
			return
		}
		if !debounceTimer.Stop() {
			select {
			case <-debounceTimer.C:
			default:
			}
		}
		debounceTimer.Reset(watcherEventDebounceInterval)
		debounceEvents = debounceTimer.C
	}

	reconcile := func() error {
		w.refreshTelemetry(ctx, out)
		return w.drainAvailableLines(reader, ctx, out)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-storageEvents:
			if !ok {
				storageEvents = nil
				continue
			}
			if w.isRelevantStorageChange(event.Name) {
				armDebounce()
			}
		case <-debounceEvents:
			debounceEvents = nil
			if err := reconcile(); err != nil {
				return err
			}
			reconcileInterval = watcherInitialReconcileInterval
			resetWatcherTimer(reconcileTimer, reconcileInterval)
		case <-reconcileTimer.C:
			if err := reconcile(); err != nil {
				return err
			}
			reconcileInterval = nextWatcherReconcileInterval(reconcileInterval)
			reconcileTimer.Reset(reconcileInterval)
		case _, ok := <-storageErrors:
			if !ok {
				storageErrors = nil
			}
		}
	}
}

// emitRecentPreview reads only enough tail blocks to show the newest complete
// transcript rows. Preview rows intentionally bypass analyzer, linkage, and
// telemetry attachment; the chronological replay immediately following this
// method enriches and replaces them by the same session/step key.
func (w *Watcher) emitRecentPreview(ctx context.Context, out chan<- core.UnifiedAgentEvent) error {
	lines, err := readRecentTranscriptLines(w.filePath, watcherPreviewEventLimit)
	if err != nil {
		return err
	}
	for _, line := range lines {
		event, parseErr := w.decodeLine(line)
		if parseErr != nil {
			continue
		}
		select {
		case out <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// readRecentTranscriptLines bounds initial history work by reading backwards in
// fixed-size blocks until it has enough complete JSONL rows. It never loads an
// entire multi-megabyte transcript merely to paint the first frame.
func readRecentTranscriptLines(path string, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript preview %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat transcript preview %s: %w", path, err)
	}

	blocks := make([][]byte, 0, 1)
	newlineCount := 0
	offset := info.Size()
	for offset > 0 && newlineCount <= limit {
		blockSize := int64(watcherPreviewReadBlockBytes)
		if offset < blockSize {
			blockSize = offset
		}
		offset -= blockSize
		block := make([]byte, blockSize)
		readCount, readErr := file.ReadAt(block, offset)
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("read transcript preview %s: %w", path, readErr)
		}
		block = block[:readCount]
		blocks = append(blocks, block)
		newlineCount += bytes.Count(block, []byte{'\n'})
	}

	tail := joinReverseBlocks(blocks)
	lines := strings.Split(string(tail), "\n")
	if offset > 0 && len(lines) > 0 {
		lines = lines[1:]
	}
	return lastNonEmptyTranscriptLines(lines, limit), nil
}

func joinReverseBlocks(blocks [][]byte) []byte {
	byteCount := 0
	for _, block := range blocks {
		byteCount += len(block)
	}
	joined := make([]byte, 0, byteCount)
	for index := len(blocks) - 1; index >= 0; index-- {
		joined = append(joined, blocks[index]...)
	}
	return joined
}

func lastNonEmptyTranscriptLines(lines []string, limit int) []string {
	selected := make([]string, 0, limit)
	for index := len(lines) - 1; index >= 0 && len(selected) < limit; index-- {
		if strings.TrimSpace(lines[index]) == "" {
			continue
		}
		selected = append(selected, lines[index])
	}
	for left, right := 0, len(selected)-1; left < right; left, right = left+1, right-1 {
		selected[left], selected[right] = selected[right], selected[left]
	}
	return selected
}

func (w *Watcher) openStorageWatcher() (*fsnotify.Watcher, error) {
	storageWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	directories := map[string]struct{}{
		filepath.Dir(w.filePath): {},
	}
	if w.sqlitePath != "" {
		directories[filepath.Dir(w.sqlitePath)] = struct{}{}
	}
	for directory := range directories {
		if err := storageWatcher.Add(directory); err != nil {
			_ = storageWatcher.Close()
			return nil, err
		}
	}
	return storageWatcher, nil
}

func (w *Watcher) isRelevantStorageChange(changedPath string) bool {
	cleanPath := filepath.Clean(changedPath)
	if cleanPath == filepath.Clean(w.filePath) || cleanPath == filepath.Clean(w.sqlitePath) {
		return true
	}
	return cleanPath == filepath.Clean(w.sqlitePath+"-wal") || cleanPath == filepath.Clean(w.sqlitePath+"-shm")
}

func (w *Watcher) drainAvailableLines(reader *bufio.Reader, ctx context.Context, out chan<- core.UnifiedAgentEvent) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				w.pendingLine += line
				return nil
			}
			return fmt.Errorf("read transcript lines: %w", err)
		}
		completeLine := w.pendingLine + line
		w.pendingLine = ""
		w.processLine(completeLine, ctx, out)
	}
}

func (w *Watcher) refreshTelemetry(ctx context.Context, out chan<- core.UnifiedAgentEvent) {
	if w.sqliteReader == nil || w.sqliteReader.PollLatest() != nil {
		return
	}
	for stepIndex, event := range w.pendingCloud {
		if !w.attachPersistedUsage(&event) {
			continue
		}
		delete(w.pendingCloud, stepIndex)
		select {
		case out <- event:
		case <-ctx.Done():
			return
		}
	}
	w.compactPendingCloudOrder()
}

func (w *Watcher) rememberPendingCloud(event core.UnifiedAgentEvent) {
	if _, exists := w.pendingCloud[event.StepIndex]; !exists {
		w.pendingOrder = append(w.pendingOrder, event.StepIndex)
	}
	w.pendingCloud[event.StepIndex] = event
	for len(w.pendingCloud) > watcherPendingCloudEventLimit && len(w.pendingOrder) > 0 {
		oldestStepIndex := w.pendingOrder[0]
		w.pendingOrder = w.pendingOrder[1:]
		delete(w.pendingCloud, oldestStepIndex)
	}
}

func (w *Watcher) compactPendingCloudOrder() {
	kept := w.pendingOrder[:0]
	for _, stepIndex := range w.pendingOrder {
		if _, exists := w.pendingCloud[stepIndex]; exists {
			kept = append(kept, stepIndex)
		}
	}
	w.pendingOrder = kept
}

func (w *Watcher) attachPersistedUsage(event *core.UnifiedAgentEvent) bool {
	if w.sqliteReader == nil {
		return false
	}
	metadata := w.sqliteReader.TakeTelemetryForGeneratedStep(event.StepIndex)
	if metadata == nil {
		return false
	}
	event.Usage = core.PersistedUsageObservation{
		Available:                true,
		Source:                   persistedUsageObservationSource,
		GenerationIndex:          metadata.GenIndex,
		StepIndex:                event.StepIndex,
		ModelName:                metadata.ModelName,
		HasObservedContextTokens: metadata.HasObservedContextTokens,
		ObservedContextTokens:    metadata.ObservedContextTokens,
		HasMeteredInputTokens:    metadata.HasMeteredInputTokens,
		MeteredInputTokens:       metadata.MeteredInputTokens,
		HasCachedContentTokens:   metadata.HasCachedContentTokens,
		CachedContentTokens:      metadata.CachedContentTokens,
		HasContextLimit:          metadata.HasContextLimit,
		ContextLimit:             metadata.ContextLimit,
	}
	return true
}

func resetWatcherTimer(timer *time.Timer, interval time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(interval)
}

func nextWatcherReconcileInterval(current time.Duration) time.Duration {
	next := current * watcherReconcileBackoffFactor
	if next > watcherMaximumReconcileInterval {
		return watcherMaximumReconcileInterval
	}
	return next
}

func (w *Watcher) processLine(line string, ctx context.Context, out chan<- core.UnifiedAgentEvent) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	event, err := w.decodeLine(trimmed)
	if err != nil {
		return
	}

	if event.StepIndex < w.lastStepIdx {
		return
	}
	if event.StepIndex > w.lastStepIdx {
		w.lastStepIdx = event.StepIndex
		w.enrichEvent(&event)
		if w.analyzer != nil {
			w.analyzer.AnalyzeStep(&event)
		}
	}

	select {
	case out <- event:
	case <-ctx.Done():
	}
}

func (w *Watcher) parseLine(line string) (core.UnifiedAgentEvent, error) {
	event, err := w.decodeLine(line)
	if err != nil {
		return core.UnifiedAgentEvent{}, err
	}
	w.enrichEvent(&event)
	return event, nil
}

// decodeLine converts one persisted JSONL record without mutating watcher
// state. It is safe for the short initial preview; stateful enrichment is kept
// in enrichEvent so the canonical replay can remain chronological.
func (w *Watcher) decodeLine(line string) (core.UnifiedAgentEvent, error) {
	var raw RawTranscriptLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return core.UnifiedAgentEvent{}, err
	}

	t, err := time.Parse(time.RFC3339, raw.CreatedAt)
	if err != nil {
		t = time.Now()
	} else {
		t = t.Local()
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
		summary = cleanToolOutputSnippet(raw.Content, 60)
	case "VIEW_FILE":
		stepType = core.StepTypeViewFile
		summary = cleanToolOutputSnippet(raw.Content, 60)
	case "CODE_ACTION":
		stepType = core.StepTypeCodeAction
		summary = cleanToolOutputSnippet(raw.Content, 60)
	case "LIST_DIRECTORY":
		stepType = core.StepTypeListDirectory
		summary = cleanToolOutputSnippet(raw.Content, 60)
	case "ASK_QUESTION":
		stepType = core.StepTypeAskQuestion
		summary = cleanToolOutputSnippet(raw.Content, 60)
	case "GENERIC":
		stepType = core.StepTypeGeneric
		summary = cleanToolOutputSnippet(raw.Content, 60)
	case "ERROR_MESSAGE":
		stepType = core.StepTypeError
		summary = cleanToolOutputSnippet(raw.Content, 60)
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
		SessionID:  w.sessionID,
		StepIndex:  raw.StepIndex,
		Timestamp:  t,
		Source:     raw.Source,
		Type:       stepType,
		Status:     raw.Status,
		Summary:    summary,
		RawContent: raw.Content,
		Thinking:   raw.Thinking,
		ToolCalls:  toolCalls,
	}

	return event, nil
}

func (w *Watcher) enrichEvent(event *core.UnifiedAgentEvent) {
	if w.tracker != nil {
		w.tracker.ProcessEvent(event)
	}
	// Persisted usage may arrive after its transcript record. Retain only
	// unresolved cloud events, then emit one in-place update when metadata is
	// observed; this avoids both fabricated telemetry and unbounded retention.
	if event.IsCloudStep() && !w.attachPersistedUsage(event) && w.sqliteReader != nil {
		w.rememberPendingCloud(*event)
	}
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

func cleanToolOutputSnippet(content string, maxLen int) string {
	lines := strings.Split(content, "\n")
	var meaningful []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "Created At:") || strings.HasPrefix(l, "Completed At:") ||
			strings.HasPrefix(l, "File Path:") || strings.HasPrefix(l, "Total Lines:") ||
			strings.HasPrefix(l, "Total Bytes:") || strings.HasPrefix(l, "Showing lines") ||
			strings.HasPrefix(l, "The command exited with code") || strings.HasPrefix(l, "Output:") {
			continue
		}
		meaningful = append(meaningful, l)
		if len(meaningful) >= 3 {
			break
		}
	}
	if len(meaningful) == 0 {
		return truncate(content, maxLen)
	}
	joined := strings.Join(meaningful, " ")
	return truncate(joined, maxLen)
}
