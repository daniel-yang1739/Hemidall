package antigravity

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"

	"heimdall/internal/agent_adapters"
	contextparser "heimdall/internal/agent_adapters/antigravity/parsers/context"
	"heimdall/internal/agent_adapters/antigravity/parsers/conversation"
	"heimdall/internal/agent_adapters/antigravity/parsers/transcript"
	"heimdall/internal/core"
)

const initialTranscriptOffset int64 = 0

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
	store := &SessionStore{
		ctx:                ctx,
		ref:                ref,
		transcriptSource:   transcriptSource,
		transcriptOffset:   initialTranscriptOffset,
		stepsByIndex:       make(map[int]agents.Step),
		lastParsedGenIndex: -1,
	}
	if conversationAvailable {
		store.conversationSource = &conversationSource
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
		revision: store.revision, transcriptInfo: store.transcriptInfo,
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
		if len(genSteps) > 0 || transcriptReset {
			store.revision++
		}
		changedIndexes := append(resetIndexes, genSteps...)
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
	store.transcriptOffset += int64(len(appendedBytes))
	steps, diagnostics, trailingFragment := transcript.ParseTranscriptFragment(store.pendingTranscript+string(appendedBytes), store.transcriptSource)
	store.pendingTranscript = trailingFragment
	store.diagnostics = append(store.diagnostics, diagnostics...)
	changedIndexes := make([]int, 0, len(resetIndexes)+len(steps))
	changedIndexes = append(changedIndexes, resetIndexes...)
	for _, step := range steps {
		store.stepsByIndex[step.Index] = step
		changedIndexes = append(changedIndexes, step.Index)
	}
	genSteps, conversationErr := store.refreshConversation()
	if conversationErr != nil {
		return agents.SessionDelta{}, conversationErr
	}
	changedIndexes = append(changedIndexes, genSteps...)
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
