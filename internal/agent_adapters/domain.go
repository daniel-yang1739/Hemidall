package agents

import "heimdall/internal/core"

// Temporary aliases retain parser ergonomics while the domain is now core-owned.
type ToolCall = core.ToolCall
type Step = core.Step
type Generation = core.Generation
type UsageObservation = core.UsageObservation
type ContextSnapshot = core.ContextSnapshot
type ModelCatalog = core.ModelCatalog
type ModelRecord = core.ModelRecord
type Session = core.Session
