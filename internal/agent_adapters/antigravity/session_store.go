package antigravity

import (
	"fmt"
	"io"
	"os"
	"sync"

	"heimdall/internal/agent_adapters"
	contextparser "heimdall/internal/agent_adapters/antigravity/parsers/context"
	"heimdall/internal/agent_adapters/antigravity/parsers/conversation"
	"heimdall/internal/agent_adapters/antigravity/parsers/transcript"
)

const initialTranscriptOffset int64 = 0

// SessionStore keeps the parsed facts for one Antigravity session in memory.
// Original artifacts remain the source of truth; this store never writes them.
type SessionStore struct {
	mu                 sync.RWMutex
	ref                agents.SessionRef
	transcriptSource   agents.SourceRef
	transcriptOffset   int64
	pendingTranscript  string
	stepsByIndex       map[int]agents.Step
	generations        []agents.Generation
	contextSnapshots   []agents.ContextSnapshot
	conversationSource *agents.SourceRef
	conversationState  sourceState
	revision           uint64
	diagnostics        []error
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
	transcriptSource, ok := findTranscriptSource(sources)
	if !ok {
		return nil, fmt.Errorf("session %s has no transcript source", ref.SessionID)
	}
	conversationSource, conversationAvailable := findConversationSource(sources)
	store := &SessionStore{
		ref:              ref,
		transcriptSource: transcriptSource,
		transcriptOffset: initialTranscriptOffset,
		stepsByIndex:     make(map[int]agents.Step),
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
	return cloneSession(buildSession(store.ref, store.stepsByIndex, store.generations, store.contextSnapshots, store.revision))
}

// Refresh incrementally reads only transcript bytes appended since the last
// successful refresh. If the source was truncated, the transcript facts are
// rebuilt from the beginning rather than mixing two file generations.
func (store *SessionStore) Refresh() (agents.SessionDelta, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	fileInfo, err := os.Stat(store.transcriptSource.Path)
	if err != nil {
		return agents.SessionDelta{}, fmt.Errorf("stat transcript: %w", err)
	}
	transcriptReset := fileInfo.Size() < store.transcriptOffset
	resetIndexes := make([]int, 0)
	if transcriptReset {
		resetIndexes = store.resetTranscriptFacts()
	}
	if fileInfo.Size() == store.transcriptOffset {
		conversationChanged, conversationErr := store.refreshConversation()
		if conversationErr != nil {
			return agents.SessionDelta{}, conversationErr
		}
		if conversationChanged || transcriptReset {
			store.revision++
		}
		return store.currentDelta(resetIndexes), nil
	}
	file, err := os.Open(store.transcriptSource.Path)
	if err != nil {
		return agents.SessionDelta{}, fmt.Errorf("open transcript: %w", err)
	}
	defer file.Close()
	if _, err := file.Seek(store.transcriptOffset, io.SeekStart); err != nil {
		return agents.SessionDelta{}, fmt.Errorf("seek transcript: %w", err)
	}
	appendedBytes, err := io.ReadAll(file)
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
	_, conversationErr := store.refreshConversation()
	if conversationErr != nil {
		return agents.SessionDelta{}, conversationErr
	}
	store.revision++
	return store.currentDelta(changedIndexes), nil
}

// Diagnostics returns copies of non-fatal parser diagnostics accumulated by refresh.
func (store *SessionStore) Diagnostics() []error {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return append([]error(nil), store.diagnostics...)
}

func (store *SessionStore) resetTranscriptFacts() []int {
	changedIndexes := make([]int, 0, len(store.stepsByIndex))
	for stepIndex := range store.stepsByIndex {
		changedIndexes = append(changedIndexes, stepIndex)
	}
	store.transcriptOffset = initialTranscriptOffset
	store.pendingTranscript = ""
	store.stepsByIndex = make(map[int]agents.Step)
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

func (store *SessionStore) refreshConversation() (bool, error) {
	if store.conversationSource == nil {
		return false, nil
	}
	state, err := readSourceState(store.conversationSource.Path)
	if err != nil {
		return false, fmt.Errorf("stat conversation database: %w", err)
	}
	if state == store.conversationState {
		return false, nil
	}
	generations, diagnostics, parseErr := (conversation.Parser{}).ParseDatabase(store.conversationSource.Path, *store.conversationSource)
	if parseErr != nil {
		return false, parseErr
	}
	store.generations = generations
	store.diagnostics = append(store.diagnostics, diagnostics...)
	snapshot, snapshotErr := (contextparser.Parser{}).ParseLatest(store.conversationSource.Path, *store.conversationSource)
	if snapshotErr == nil {
		store.contextSnapshots = []agents.ContextSnapshot{snapshot}
	} else {
		store.contextSnapshots = nil
	}
	store.conversationState = state
	return true, nil
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
