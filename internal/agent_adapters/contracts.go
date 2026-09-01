package agents

import "heimdall/internal/core"

// Temporary aliases retain parser ergonomics while the domain is now core-owned.
type AgentID = core.AgentID
type SourceKind = core.SourceKind
type EvidenceLevel = core.EvidenceLevel
type SessionRef = core.SessionRef
type SourceRef = core.SourceRef
type Evidence = core.Evidence
type SessionDelta = core.SessionDelta

const (
	SourceKindHistory        = core.SourceKindHistory
	SourceKindTranscript     = core.SourceKindTranscript
	SourceKindArtifacts      = core.SourceKindArtifacts
	SourceKindConversation   = core.SourceKindConversation
	EvidenceObservedOnly     = core.EvidenceObservedOnly
	EvidenceWireStructure    = core.EvidenceWireStructure
	EvidenceColumnEquality   = core.EvidenceColumnEquality
	EvidenceArtifactEquality = core.EvidenceArtifactEquality
	EvidenceStrongInference  = core.EvidenceStrongInference
	EvidenceUnavailable      = core.EvidenceUnavailable
)

// SessionStore is the read-only query boundary for one observed session.
// Implementations may refresh their in-memory data from external sources but
// must never require consumers to know the source file or database schema.
type SessionStore interface {
	Session() core.Session
	Refresh() (SessionDelta, error)
}
