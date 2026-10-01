package antigravity

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"sync"

	"heimdall/internal/agent_adapters"
	contextparser "heimdall/internal/agent_adapters/antigravity/parsers/context"
	"heimdall/internal/agent_adapters/antigravity/parsers/conversation"
	"heimdall/internal/agent_adapters/antigravity/parsers/transcript"
	"heimdall/internal/core"
)

const (
	initialTranscriptOffset int64 = 0
	stepOutputFileName            = "output.txt"
	stepStatusRunning             = "RUNNING"
	stepOutputEvidenceNote        = "persisted step output"
)

// SessionStore keeps the parsed facts for one Antigravity session in memory.
// Original artifacts remain the source of truth; this store never writes them.
type SessionStore struct {
	ctx                     context.Context
	mu                      sync.RWMutex
	ref                     agents.SessionRef
	transcriptSource        agents.SourceRef
	transcriptOffset        int64
	pendingTranscript       string
	stepsByIndex            map[int]agents.Step
	generations             []agents.Generation
	lastParsedGenIndex      int
	contextSnapshots        []agents.ContextSnapshot
	conversationSource      *agents.SourceRef
	conversationState       sourceState
	stepOutputsSource       *agents.SourceRef
	stepOutputsState        sourceState
	stepOutputsScanned      bool
	stepOutputStates        map[int]sourceState
	pendingStepOutputs      map[int]struct{}
	transcriptContent       map[int]string
	revision                uint64
	diagnostics             []error
	conversationDiagnostics []error
	transcriptInfo          os.FileInfo
}

type sourceState struct {
	modifiedAt    int64
	byteCount     int64
	walModifiedAt int64
	walByteCount  int64
	available     bool
}

// NewSessionStore creates a store for sources already resolved by discovery.
func NewSessionStore(ref agents.SessionRef, sources []agents.SourceRef) (*SessionStore, error) {
	return NewSessionStoreWithContext(context.Background(), ref, sources)
}

// NewSessionStoreWithContext binds source reads to a session activation.
func NewSessionStoreWithContext(ctx context.Context, ref agents.SessionRef, sources []agents.SourceRef) (*SessionStore, error) {
	transcriptSource, ok := findTranscriptSource(sources)
	if !ok {
		return nil, fmt.Errorf("session %s has no transcript source", ref.SessionID)
	}
	conversationSource, conversationAvailable := findConversationSource(sources)
	stepOutputsSource, stepOutputsAvailable := findStepOutputsSource(sources)
	if !stepOutputsAvailable {
		stepOutputsSource, stepOutputsAvailable = discoverStepOutputsFromTranscript(transcriptSource.Path)
	}
	store := &SessionStore{
		ctx:                ctx,
		ref:                ref,
		transcriptSource:   transcriptSource,
		transcriptOffset:   initialTranscriptOffset,
		stepsByIndex:       make(map[int]agents.Step),
		stepOutputStates:   make(map[int]sourceState),
		pendingStepOutputs: make(map[int]struct{}),
		transcriptContent:  make(map[int]string),
		lastParsedGenIndex: -1,
	}
	if conversationAvailable {
		store.conversationSource = &conversationSource
	}
	if stepOutputsAvailable {
		store.stepOutputsSource = &stepOutputsSource
	}
	return store, nil
}

// Session returns an immutable copy of the current assembled read model.
func (store *SessionStore) Session() agents.Session {
	store.mu.RLock()
	defer store.mu.RUnlock()
	session := buildSession(store.ref, store.stepsByIndex, store.generations, store.contextSnapshots, store.revision)
	session.DiagnosticCount = len(store.diagnostics) + len(store.conversationDiagnostics)
	return core.CloneSession(session)
}

// Refresh incrementally reads only transcript bytes appended since the last
// successful refresh. If the source was truncated, the transcript facts are
// rebuilt from the beginning rather than mixing two file generations.
func (store *SessionStore) Refresh() (agents.SessionDelta, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.ctx.Err(); err != nil {
		return agents.SessionDelta{}, err
	}
	info, err := os.Stat(store.transcriptSource.Path)
	if err != nil {
		return agents.SessionDelta{}, fmt.Errorf("stat transcript: %w", err)
	}
	unchanged := store.transcriptInfo != nil && os.SameFile(info, store.transcriptInfo) && info.Size() == store.transcriptOffset && info.ModTime() == store.transcriptInfo.ModTime()
	if unchanged && store.conversationSource != nil {
		state, stateErr := readSourceState(store.conversationSource.Path)
		if stateErr != nil {
			return agents.SessionDelta{}, stateErr
		}
		unchanged = state == store.conversationState
	}
	if unchanged {
		artifactsUnchanged, artifactsErr := store.stepOutputsUnchanged()
		if artifactsErr != nil {
			return agents.SessionDelta{}, artifactsErr
		}
		unchanged = artifactsUnchanged
	}
	if unchanged {
		return store.currentDelta(nil), nil
	}

	// Parse against private working state; cursors and facts commit together.
	working := &SessionStore{
		ctx: store.ctx,
		ref: store.ref, transcriptSource: store.transcriptSource,
		transcriptOffset: store.transcriptOffset, pendingTranscript: store.pendingTranscript,
		stepsByIndex: make(map[int]agents.Step, len(store.stepsByIndex)),
		generations:  store.generations, contextSnapshots: store.contextSnapshots,
		conversationSource: store.conversationSource, conversationState: store.conversationState,
		stepOutputsSource: store.stepOutputsSource, stepOutputsState: store.stepOutputsState,
		stepOutputsScanned: store.stepOutputsScanned,
		stepOutputStates:   cloneSourceStates(store.stepOutputStates),
		pendingStepOutputs: cloneIndexSet(store.pendingStepOutputs),
		transcriptContent:  cloneTranscriptContent(store.transcriptContent),
		revision:           store.revision, transcriptInfo: store.transcriptInfo,
		diagnostics:             append([]error(nil), store.diagnostics...),
		conversationDiagnostics: append([]error(nil), store.conversationDiagnostics...),
	}
	for index, step := range store.stepsByIndex {
		working.stepsByIndex[index] = step
	}
	delta, err := working.refresh()
	if err != nil {
		return agents.SessionDelta{}, err
	}
	if err := store.ctx.Err(); err != nil {
		return agents.SessionDelta{}, err
	}
	working.transcriptInfo = info
	store.transcriptOffset, store.pendingTranscript = working.transcriptOffset, working.pendingTranscript
	store.stepsByIndex, store.generations = working.stepsByIndex, working.generations
	store.contextSnapshots, store.conversationState = working.contextSnapshots, working.conversationState
	store.stepOutputsState, store.stepOutputsScanned = working.stepOutputsState, working.stepOutputsScanned
	store.stepOutputStates, store.pendingStepOutputs = working.stepOutputStates, working.pendingStepOutputs
	store.transcriptContent = working.transcriptContent
	store.revision, store.transcriptInfo, store.diagnostics = working.revision, working.transcriptInfo, working.diagnostics
	store.conversationDiagnostics = working.conversationDiagnostics
	return delta, nil
}

func (store *SessionStore) refresh() (agents.SessionDelta, error) {

	fileInfo, err := os.Stat(store.transcriptSource.Path)
	if err != nil {
		return agents.SessionDelta{}, fmt.Errorf("stat transcript: %w", err)
	}
	transcriptReset := fileInfo.Size() < store.transcriptOffset || (store.transcriptInfo != nil && (!os.SameFile(fileInfo, store.transcriptInfo) || (fileInfo.Size() == store.transcriptOffset && fileInfo.ModTime() != store.transcriptInfo.ModTime())))
	resetIndexes := make([]int, 0)
	if transcriptReset {
		resetIndexes = store.resetTranscriptFacts()
	}
	if fileInfo.Size() == store.transcriptOffset {
		genSteps, conversationErr := store.refreshConversation()
		if conversationErr != nil {
			return agents.SessionDelta{}, conversationErr
		}
		artifactSteps, artifactsErr := store.refreshStepOutputs()
		if artifactsErr != nil {
			return agents.SessionDelta{}, artifactsErr
		}
		if len(genSteps) > 0 || len(artifactSteps) > 0 || transcriptReset {
			store.revision++
		}
		changedIndexes := append(resetIndexes, genSteps...)
		changedIndexes = append(changedIndexes, artifactSteps...)
		delta := store.currentDelta(changedIndexes)
		delta.Reset = transcriptReset
		return delta, nil
	}
	file, err := os.Open(store.transcriptSource.Path)
	if err != nil {
		return agents.SessionDelta{}, fmt.Errorf("open transcript: %w", err)
	}
	defer file.Close()
	if _, err := file.Seek(store.transcriptOffset, io.SeekStart); err != nil {
		return agents.SessionDelta{}, fmt.Errorf("seek transcript: %w", err)
	}
	appendedBytes, err := io.ReadAll(sessionReader{ctx: store.ctx, reader: file})
	if err != nil {
		return agents.SessionDelta{}, fmt.Errorf("read transcript: %w", err)
	}
	initialTranscriptRead := store.transcriptOffset == initialTranscriptOffset
	store.transcriptOffset += int64(len(appendedBytes))
	steps, diagnostics, trailingFragment := transcript.ParseTranscriptFragment(store.pendingTranscript+string(appendedBytes), store.transcriptSource)
	store.pendingTranscript = trailingFragment
	store.diagnostics = append(store.diagnostics, diagnostics...)
	changedIndexes := make([]int, 0, len(resetIndexes)+len(steps))
	changedIndexes = append(changedIndexes, resetIndexes...)
	for _, step := range steps {
		store.stepsByIndex[step.Index] = step
		store.transcriptContent[step.Index] = step.Content
		if isStepOutputCandidate(step) && (!initialTranscriptRead || step.Status == stepStatusRunning) {
			store.pendingStepOutputs[step.Index] = struct{}{}
		}
		changedIndexes = append(changedIndexes, step.Index)
	}
	genSteps, conversationErr := store.refreshConversation()
	if conversationErr != nil {
		return agents.SessionDelta{}, conversationErr
	}
	changedIndexes = append(changedIndexes, genSteps...)
	artifactSteps, artifactsErr := store.refreshStepOutputs()
	if artifactsErr != nil {
		return agents.SessionDelta{}, artifactsErr
	}
	changedIndexes = append(changedIndexes, artifactSteps...)
	store.revision++
	delta := store.currentDelta(changedIndexes)
	delta.Reset = transcriptReset
	return delta, nil
}

// Diagnostics returns copies of non-fatal parser diagnostics accumulated by refresh.
func (store *SessionStore) Diagnostics() []error {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return append(append([]error(nil), store.diagnostics...), store.conversationDiagnostics...)
}

func (store *SessionStore) resetTranscriptFacts() []int {
	changedIndexes := make([]int, 0, len(store.stepsByIndex))
	for stepIndex := range store.stepsByIndex {
		changedIndexes = append(changedIndexes, stepIndex)
	}
	store.transcriptOffset = initialTranscriptOffset
	store.pendingTranscript = ""
	store.stepsByIndex = make(map[int]agents.Step)
	store.generations = nil
	store.lastParsedGenIndex = -1
	store.contextSnapshots = nil
	store.conversationState = sourceState{}
	store.stepOutputsState = sourceState{}
	store.stepOutputsScanned = false
	store.stepOutputStates = make(map[int]sourceState)
	store.pendingStepOutputs = make(map[int]struct{})
	store.transcriptContent = make(map[int]string)
	store.diagnostics, store.conversationDiagnostics = nil, nil
	return changedIndexes
}

func (store *SessionStore) currentDelta(changedIndexes []int) agents.SessionDelta {
	return agents.SessionDelta{
		SessionID:          store.ref.SessionID,
		Revision:           store.revision,
		ChangedStepIndexes: append([]int(nil), changedIndexes...),
	}
}

func findTranscriptSource(sources []agents.SourceRef) (agents.SourceRef, bool) {
	for _, source := range sources {
		if source.Kind == agents.SourceKindTranscript {
			return source, true
		}
	}
	return agents.SourceRef{}, false
}

func findConversationSource(sources []agents.SourceRef) (agents.SourceRef, bool) {
	for _, source := range sources {
		if source.Kind == agents.SourceKindConversation {
			return source, true
		}
	}
	return agents.SourceRef{}, false
}

func findStepOutputsSource(sources []agents.SourceRef) (agents.SourceRef, bool) {
	for _, source := range sources {
		if source.Kind == agents.SourceKindArtifacts {
			return source, true
		}
	}
	return agents.SourceRef{}, false
}

func cloneSourceStates(states map[int]sourceState) map[int]sourceState {
	cloned := make(map[int]sourceState, len(states))
	for index, state := range states {
		cloned[index] = state
	}
	return cloned
}

func cloneIndexSet(indexes map[int]struct{}) map[int]struct{} {
	cloned := make(map[int]struct{}, len(indexes))
	for index := range indexes {
		cloned[index] = struct{}{}
	}
	return cloned
}

func cloneTranscriptContent(content map[int]string) map[int]string {
	cloned := make(map[int]string, len(content))
	for index, value := range content {
		cloned[index] = value
	}
	return cloned
}

func (store *SessionStore) refreshStepOutputs() ([]int, error) {
	if store.stepOutputsSource == nil {
		return nil, nil
	}
	rootState, err := readSourceState(store.stepOutputsSource.Path)
	if os.IsNotExist(err) {
		return store.removeUnavailableStepOutputs(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat step outputs: %w", err)
	}

	targets := make(map[int]struct{}, len(store.stepOutputStates)+len(store.pendingStepOutputs))
	for index := range store.stepOutputStates {
		targets[index] = struct{}{}
	}
	for index := range store.pendingStepOutputs {
		targets[index] = struct{}{}
	}
	if !store.stepOutputsScanned || rootState != store.stepOutputsState {
		entries, readErr := os.ReadDir(store.stepOutputsSource.Path)
		if readErr != nil {
			return nil, fmt.Errorf("read step outputs: %w", readErr)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			index, parseErr := strconv.Atoi(entry.Name())
			if parseErr == nil {
				targets[index] = struct{}{}
			}
		}
		store.stepOutputsState = rootState
		store.stepOutputsScanned = true
	}

	indexes := make([]int, 0, len(targets))
	for index := range targets {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	changed := make([]int, 0)
	for _, index := range indexes {
		stepChanged, refreshErr := store.refreshStepOutput(index)
		if refreshErr != nil {
			return nil, refreshErr
		}
		if stepChanged {
			changed = append(changed, index)
		}
	}
	return changed, nil
}

func (store *SessionStore) stepOutputsUnchanged() (bool, error) {
	if store.stepOutputsSource == nil {
		return true, nil
	}
	rootState, err := readSourceState(store.stepOutputsSource.Path)
	if os.IsNotExist(err) {
		return !store.stepOutputsState.available && len(store.stepOutputStates) == 0, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat step outputs: %w", err)
	}
	if !store.stepOutputsScanned || rootState != store.stepOutputsState {
		return false, nil
	}
	for index, knownState := range store.stepOutputStates {
		path := filepath.Join(store.stepOutputsSource.Path, strconv.Itoa(index), stepOutputFileName)
		currentState, stateErr := readSourceState(path)
		if os.IsNotExist(stateErr) {
			return false, nil
		}
		if stateErr != nil {
			return false, fmt.Errorf("stat step output %d: %w", index, stateErr)
		}
		if currentState != knownState {
			return false, nil
		}
	}
	for index := range store.pendingStepOutputs {
		path := filepath.Join(store.stepOutputsSource.Path, strconv.Itoa(index), stepOutputFileName)
		_, stateErr := os.Stat(path)
		if stateErr == nil {
			return false, nil
		}
		if !os.IsNotExist(stateErr) {
			return false, fmt.Errorf("stat pending step output %d: %w", index, stateErr)
		}
	}
	return true, nil
}

func (store *SessionStore) refreshStepOutput(index int) (bool, error) {
	path := filepath.Join(store.stepOutputsSource.Path, strconv.Itoa(index), stepOutputFileName)
	state, err := readSourceState(path)
	if os.IsNotExist(err) {
		delete(store.stepOutputStates, index)
		return store.restoreTranscriptContent(index), nil
	}
	if err != nil {
		return false, fmt.Errorf("stat step output %d: %w", index, err)
	}
	step, exists := store.stepsByIndex[index]
	if !exists {
		store.stepOutputStates[index] = state
		return false, nil
	}
	knownState, stateKnown := store.stepOutputStates[index]
	if stateKnown && knownState == state && step.ContentSource == agents.SourceKindArtifacts {
		delete(store.pendingStepOutputs, index)
		return false, nil
	}
	content, readErr := readFileContext(store.ctx, path)
	if readErr != nil {
		return false, fmt.Errorf("read step output %d: %w", index, readErr)
	}
	changed := step.Content != content || step.ContentSource != agents.SourceKindArtifacts
	step.Content = content
	step.ContentSource = agents.SourceKindArtifacts
	step.Evidence = withoutStepOutputEvidence(step.Evidence)
	step.Evidence = append(step.Evidence, agents.Evidence{
		Level:   agents.EvidenceObservedOnly,
		Source:  *store.stepOutputsSource,
		Locator: filepath.Join(strconv.Itoa(index), stepOutputFileName),
		Note:    stepOutputEvidenceNote,
	})
	store.stepsByIndex[index] = step
	store.stepOutputStates[index] = state
	delete(store.pendingStepOutputs, index)
	return changed, nil
}

func (store *SessionStore) removeUnavailableStepOutputs() []int {
	changed := make([]int, 0, len(store.stepOutputStates))
	for index := range store.stepOutputStates {
		if store.restoreTranscriptContent(index) {
			changed = append(changed, index)
		}
	}
	store.stepOutputsState = sourceState{}
	store.stepOutputsScanned = false
	store.stepOutputStates = make(map[int]sourceState)
	sort.Ints(changed)
	return changed
}

func (store *SessionStore) restoreTranscriptContent(index int) bool {
	step, stepExists := store.stepsByIndex[index]
	content, contentExists := store.transcriptContent[index]
	if !stepExists || !contentExists || step.ContentSource != agents.SourceKindArtifacts {
		return false
	}
	step.Content = content
	step.ContentSource = agents.SourceKindTranscript
	step.Evidence = withoutStepOutputEvidence(step.Evidence)
	store.stepsByIndex[index] = step
	if isStepOutputCandidate(step) {
		store.pendingStepOutputs[index] = struct{}{}
	}
	return true
}

func withoutStepOutputEvidence(evidence []agents.Evidence) []agents.Evidence {
	filtered := make([]agents.Evidence, 0, len(evidence))
	for _, item := range evidence {
		if item.Note != stepOutputEvidenceNote {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func isStepOutputCandidate(step agents.Step) bool {
	switch core.StepType(step.Kind) {
	case core.StepTypeToolResult,
		core.StepTypeRunCommand,
		core.StepTypeViewFile,
		core.StepTypeCodeAction,
		core.StepTypeListDirectory,
		core.StepTypeAskQuestion,
		core.StepTypeGeneric,
		core.StepTypeError,
		core.StepTypeUnknown:
		return true
	default:
		return false
	}
}

func readFileContext(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(sessionReader{ctx: ctx, reader: file})
	return string(content), err
}

func (store *SessionStore) refreshConversation() ([]int, error) {
	if store.conversationSource == nil {
		return nil, nil
	}
	state, err := readSourceState(store.conversationSource.Path)
	if err != nil {
		return nil, fmt.Errorf("stat conversation database: %w", err)
	}
	if state == store.conversationState {
		return nil, nil
	}

	// Existing rows may receive usage after their initial insertion. Reconcile
	// all rows on database/WAL changes rather than assuming append-only records.
	generations, diagnostics, parseErr := (conversation.Parser{}).ParseDatabaseContext(store.ctx, store.conversationSource.Path, *store.conversationSource)
	if parseErr != nil {
		return nil, parseErr
	}
	if !reflect.DeepEqual(store.conversationDiagnostics, diagnostics) {
		store.conversationDiagnostics = diagnostics
		store.revision++
	}

	changedStepIndexes := make([]int, 0, len(generations))
	if !reflect.DeepEqual(store.generations, generations) {
		for _, gen := range store.generations {
			changedStepIndexes = append(changedStepIndexes, gen.StepIndex)
		}
		store.generations = generations
		for _, gen := range generations {
			changedStepIndexes = append(changedStepIndexes, gen.StepIndex)
		}
	}

	snapshot, snapshotErr := (contextparser.Parser{}).ParseLatestContext(store.ctx, store.conversationSource.Path, *store.conversationSource)
	var snapshots []agents.ContextSnapshot
	if snapshotErr == nil {
		snapshots = []agents.ContextSnapshot{snapshot}
	}
	if !reflect.DeepEqual(store.contextSnapshots, snapshots) {
		store.contextSnapshots = snapshots
		// A context-only change must invalidate every derived read model.
		store.revision++
	}
	store.conversationState = state
	return changedStepIndexes, nil
}

type sessionReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader sessionReader) Read(data []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(data)
}

func readSourceState(path string) (sourceState, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sourceState{}, err
	}
	state := sourceState{modifiedAt: info.ModTime().UnixNano(), byteCount: info.Size(), available: true}
	walInfo, walErr := os.Stat(path + "-wal")
	if walErr == nil {
		state.walModifiedAt = walInfo.ModTime().UnixNano()
		state.walByteCount = walInfo.Size()
	}
	return state, nil
}
