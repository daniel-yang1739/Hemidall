package antigravity

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"heimdall/internal/core"
)

type contextSnapshotEntry struct {
	mu       sync.Mutex
	loaded   bool
	state    contextSnapshotFileState
	snapshot *ContextSnapshot
	err      error
}

// contextSnapshotFileState captures both SQLite's main database and its WAL.
// Antigravity may commit a new generation to the WAL before checkpointing the
// main database, so checking the database file alone would preserve stale UI.
type contextSnapshotFileState struct {
	databaseModified time.Time
	databaseBytes    int64
	walModified      time.Time
	walBytes         int64
}

const (
	provenanceTranscriptBootstrapIdentity = "parsed_from_transcript_bootstrap_identity_tag"
	provenanceTranscriptBootstrapRules    = "parsed_from_transcript_bootstrap_user_rules_tag"
	provenanceUnavailableFromTranscript   = "unavailable_from_transcript"
	provenancePersistedSystemPrompt       = "persisted_gen_metadata_field_1.1"
	provenancePersistedIdentity           = "parsed_from_persisted_system_prompt_identity_section"
	provenancePersistedUserRules          = "parsed_from_persisted_system_prompt_user_rules_section"
	provenancePersistedSkills             = "persisted_system_prompt_skills_section"
	provenancePersistedTools              = "persisted_gen_metadata_field_1.8_repeated"
	provenancePersistedRecords            = "persisted_gen_metadata_field_1.2_repeated"
	provenanceMCPUnknown                  = "not_attributable_without_official_protobuf_schema"
)

// ContextPayloadBuilder caches session snapshots while their SQLite database
// and WAL stay unchanged. UI redraws avoid repeat decoding, while a persisted
// generation update invalidates exactly that session's cached snapshot.
type ContextPayloadBuilder struct {
	mu        sync.Mutex
	snapshots map[string]*contextSnapshotEntry
	static    sync.Once
	cwd       string
	home      string
	skills    []core.SkillInfo
}

// NewContextPayloadBuilder creates a lazy, session-scoped evidence builder.
func NewContextPayloadBuilder() *ContextPayloadBuilder {
	return &ContextPayloadBuilder{snapshots: make(map[string]*contextSnapshotEntry)}
}

// Build assembles transcript observations and overlays a cached persisted
// snapshot when one is available. It does not fabricate an HTTP request.
func (b *ContextPayloadBuilder) Build(history []core.UnifiedAgentEvent, sessionID, targetModel string) core.AgentContextPayload {
	if targetModel == "" {
		targetModel = latestObservedModel(history)
	}
	b.static.Do(func() {
		b.cwd, _ = os.Getwd()
		b.home, _ = os.UserHomeDir()
		b.skills = core.DiscoverSkillDefinitions(
			core.FindNearestDirectory(b.cwd, filepath.Join(".agents", "skills")),
			BuiltinSkillsPath(b.home),
		)
	})
	payload := core.BuildContextPayloadFromHistory(core.ContextBuildInput{
		History:      history,
		SessionID:    sessionID,
		TargetModel:  targetModel,
		NativeTools:  core.GetNativeToolsDefinitions(history),
		ActiveSkills: b.skills,
		RuntimeMetadata: map[string]string{
			"OS":        runtime.GOOS,
			"Arch":      runtime.GOARCH,
			"Shell":     os.Getenv("SHELL"),
			"Cwd":       b.cwd,
			"SessionID": sessionID,
			"Model":     targetModel,
		},
		Provenance: transcriptProvenance(),
	})
	payload.SourceKind = core.ContextEvidenceTranscriptFallback
	snapshot, err := b.snapshotFor(sessionID)
	if err != nil {
		return payload
	}
	return overlayPersistedSnapshot(payload, snapshot)
}

func (b *ContextPayloadBuilder) snapshotFor(sessionID string) (*ContextSnapshot, error) {
	if sessionID == "" {
		return nil, os.ErrNotExist
	}
	dbPath := ConversationDatabasePath(b.home, sessionID)
	state, err := readContextSnapshotFileState(dbPath)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	entry := b.snapshots[sessionID]
	if entry == nil {
		entry = &contextSnapshotEntry{}
		b.snapshots[sessionID] = entry
	}
	b.mu.Unlock()
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.matches(state) {
		return entry.snapshot, entry.err
	}
	entry.snapshot, entry.err = LoadContextSnapshotAtPath(dbPath)
	entry.loaded = true
	entry.state = state
	return entry.snapshot, entry.err
}

func (entry *contextSnapshotEntry) matches(state contextSnapshotFileState) bool {
	return entry.loaded && entry.state == state
}

func readContextSnapshotFileState(dbPath string) (contextSnapshotFileState, error) {
	databaseInfo, err := os.Stat(dbPath)
	if err != nil {
		return contextSnapshotFileState{}, err
	}
	state := contextSnapshotFileState{
		databaseModified: databaseInfo.ModTime(),
		databaseBytes:    databaseInfo.Size(),
	}
	walInfo, walErr := os.Stat(dbPath + "-wal")
	if walErr == nil {
		state.walModified = walInfo.ModTime()
		state.walBytes = walInfo.Size()
	}
	return state, nil
}

func transcriptProvenance() map[string]string {
	return map[string]string{
		"system_prompt":     provenanceUnavailableFromTranscript,
		"identity":          provenanceTranscriptBootstrapIdentity,
		"user_rules":        provenanceTranscriptBootstrapRules,
		"checkpoint":        "observed_from_antigravity_session_history",
		"active_turns":      "observed_from_antigravity_session_history",
		"latest_prompt":     "observed_from_antigravity_session_history",
		"staged_buffers":    "inferred_from_local_steps_after_last_cloud_step",
		"tokens":            "persisted_usage_observation_when_available; otherwise unavailable",
		"runtime":           "observed_from_heimdall_process_environment",
		"tools":             "inferred_from_observed_tool_arguments",
		"skills":            "discovered_from_local_skill_directories",
		"mcp":               provenanceUnavailableFromTranscript,
		"persisted_records": provenanceUnavailableFromTranscript,
	}
}

func overlayPersistedSnapshot(payload core.AgentContextPayload, snapshot *ContextSnapshot) core.AgentContextPayload {
	payload.SourceKind = core.ContextEvidencePersistedSnapshot
	payload.SourcePath = snapshot.SourcePath
	payload.SnapshotAvailable = true
	payload.SnapshotGenIndex = snapshot.GenIndex
	payload.SnapshotBytes = snapshot.BlobBytes
	payload.SnapshotInputBoundaryStep = snapshot.InputBoundaryStepIndex
	payload.SnapshotHasInputBoundary = snapshot.HasInputBoundary
	payload.SystemPrompt = snapshot.SystemPrompt
	payload.IdentityPrompt = snapshot.Identity
	payload.ConstitutionDoc = snapshot.UserRules
	payload.SkillsSection = snapshot.SkillsSection
	payload.MCPSection = snapshot.MCPSection
	payload.PersistedContextEntryCount = snapshot.PersistedContextEntryCount
	payload.PersistedRecords = snapshot.PersistedRecords
	payload.NativeTools = snapshot.Tools
	payload.ActiveSkills = nil
	payload.MCPServers = nil
	payload.Provenance = overlaySnapshotProvenance(payload.Provenance)
	return payload
}

// overlaySnapshotProvenance replaces only the fields the persisted snapshot
// actually supplies. Transcript-derived turns and Heimdall runtime metadata
// intentionally retain their original provenance.
func overlaySnapshotProvenance(base map[string]string) map[string]string {
	provenance := make(map[string]string, len(base)+7)
	for key, value := range base {
		provenance[key] = value
	}
	provenance["system_prompt"] = provenancePersistedSystemPrompt
	provenance["identity"] = provenancePersistedIdentity
	provenance["user_rules"] = provenancePersistedUserRules
	provenance["skills"] = provenancePersistedSkills
	provenance["tools"] = provenancePersistedTools
	provenance["persisted_records"] = provenancePersistedRecords
	provenance["mcp"] = provenanceMCPUnknown
	return provenance
}

// ExtractAgentContextPayload preserves the original convenience API. Long-lived
// callers should reuse ContextPayloadBuilder instead.
func ExtractAgentContextPayload(history []core.UnifiedAgentEvent, sessionID, targetModel string) core.AgentContextPayload {
	return NewContextPayloadBuilder().Build(history, sessionID, targetModel)
}

func latestObservedModel(history []core.UnifiedAgentEvent) string {
	for index := len(history) - 1; index >= 0; index-- {
		if history[index].Usage.ModelName != "" {
			return history[index].Usage.ModelName
		}
	}
	return ""
}
