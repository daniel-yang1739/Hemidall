package core

import "time"

// AgentID identifies an observed agent provider.
type AgentID string

// SourceKind identifies an externally persisted source family.
type SourceKind string

const (
	SourceKindHistory      SourceKind = "history"
	SourceKindTranscript   SourceKind = "transcript"
	SourceKindArtifacts    SourceKind = "artifacts"
	SourceKindConversation SourceKind = "conversation_database"
)

// EvidenceLevel describes how strongly Heimdall can support a displayed value.
type EvidenceLevel string

const (
	EvidenceObservedOnly     EvidenceLevel = "observed-only"
	EvidenceWireStructure    EvidenceLevel = "confirmed-wire-structure"
	EvidenceColumnEquality   EvidenceLevel = "confirmed-by-column-equality"
	EvidenceArtifactEquality EvidenceLevel = "confirmed-by-artifact-equality"
	EvidenceStrongInference  EvidenceLevel = "strong-inference"
	EvidenceUnavailable      EvidenceLevel = "unavailable"
)

// SessionRef uniquely identifies a provider session without exposing host paths.
type SessionRef struct {
	AgentID      AgentID   `json:"agent_id"`
	SessionID    string    `json:"session_id"`
	Workspace    string    `json:"workspace,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// SourceRef locates one provider artifact used to build session data.
type SourceRef struct {
	Kind SourceKind `json:"kind"`
	Path string     `json:"path"`
}

// Evidence preserves provenance for a parsed or assembled value.
type Evidence struct {
	Level   EvidenceLevel `json:"level"`
	Source  SourceRef     `json:"source"`
	Locator string        `json:"locator"`
	Note    string        `json:"note"`
}

// SessionDelta identifies source changes applied to a session snapshot.
type SessionDelta struct {
	SessionID          string
	Revision           uint64
	ChangedStepIndexes []int
	Reset              bool
}

// ToolCall is the source-neutral representation of one observed tool request.
type ToolCall struct {
	Name string
	Args map[string]any
}

// Step is the user-facing observation of one agent step.
type Step struct {
	Index               int
	Timestamp           time.Time
	Source              string
	Kind                string
	Status              string
	Content             string
	ContentSource       SourceKind
	Thinking            string
	ToolCalls           []ToolCall
	Scope               StepScope
	AgentRole           string
	ParentStepIndex     int
	PackagedInStepIndex int
	ConsumedStepIndexes []int
	Evidence            []Evidence
}

// UsageObservation is schema-inferred persisted usage evidence for one generation.
type UsageObservation struct {
	HasThinkingOutputTokens  bool
	HasOutputContentTokens   bool
	HasObservedContextTokens bool
	ObservedContextTokens    int
	HasUncachedInputTokens   bool
	UncachedInputTokens      int
	HasCachedInputTokens     bool
	CachedInputTokens        int
	HasContextLimit          bool
	ContextLimit             int
	ThinkingOutputTokens     int
	OutputContentTokens      int
	TotalTokens              int
	TimeToFirstTokenMs       int64
	StreamingDurationMs      int64
	UpstreamRequestID        string
}

// Generation contains model and usage observations associated with one model turn.
type Generation struct {
	ID           string
	StepIndex    int
	HasStepIndex bool
	Provider     ProviderName
	ModelID      string
	Usage        UsageObservation
	Evidence     []Evidence
}

// ContextSnapshot identifies persisted or explicitly labelled fallback context evidence.
type ContextSnapshot struct {
	GenerationID            string
	GenerationIndex         int
	Source                  SourceRef
	ByteSize                int
	InputBoundaryStepIndex  int
	HasInputBoundary        bool
	SystemPrompt            string
	IdentityPrompt          string
	ConstitutionDoc         string
	SkillsSection           string
	MCPSection              string
	PersistedContextRecords []PersistedContextRecord
	NativeTools             []ToolSignature
	Evidence                []Evidence
}

// ModelRecord is one dynamically discovered model identity used in a session.
// It is built from parsed generation facts, not from a hard-coded provider list.
type ModelRecord struct {
	ModelID       string
	GenerationIDs []string
	Evidence      []Evidence
}

// ModelCatalog holds the dynamically assembled session model table.
type ModelCatalog struct {
	Records []ModelRecord
}

// Session is the provider-neutral read model consumed by core services and views.
type Session struct {
	DiagnosticCount  int
	Ref              SessionRef
	Steps            []Step
	Generations      []Generation
	ContextSnapshots []ContextSnapshot
	Models           ModelCatalog
	Revision         uint64
}
