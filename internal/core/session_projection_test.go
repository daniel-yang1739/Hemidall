package core

import (
	"testing"
	"time"
)

const projectionStepIndex = 42

func TestProjectSessionEvents_PreservesStepAndGenerationFacts(t *testing.T) {
	session := Session{
		Ref: SessionRef{SessionID: "session-a"},
		Steps: []Step{{
			Index: projectionStepIndex, Timestamp: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			Source: "MODEL", Kind: string(StepTypeModelResponse), Status: "DONE", Content: "response",
			ToolCalls: []ToolCall{{Name: "read_file", Args: map[string]any{"path": "README.md"}}},
		}},
		Generations: []Generation{{ID: "11", StepIndex: projectionStepIndex, ModelID: "gemini-3.7-flash", Usage: UsageObservation{
			HasUncachedInputTokens: true, UncachedInputTokens: 100,
			HasCachedInputTokens: true, CachedInputTokens: 90,
		}}},
	}

	events := ProjectSessionEvents(session)

	requireProjectionEqual(t, len(events), 1)
	requireProjectionEqual(t, events[0].SessionID, "session-a")
	requireProjectionEqual(t, events[0].StepIndex, projectionStepIndex)
	requireProjectionEqual(t, events[0].Type, StepTypeModelResponse)
	requireProjectionEqual(t, events[0].ToolCalls[0].ToolName, "read_file")
	requireProjectionEqual(t, events[0].Usage.UncachedInputTokens, 100)
	requireProjectionEqual(t, events[0].Usage.CachedInputTokens, 90)
	requireProjectionEqual(t, events[0].Usage.ModelName, "gemini-3.7-flash")
	requireProjectionEqual(t, events[0].Usage.GenerationIndex, 11)
}

func TestProjectSessionEvents_UsesUnknownForMissingStepKind(t *testing.T) {
	events := ProjectSessionEvents(Session{Ref: SessionRef{SessionID: "session-a"}, Steps: []Step{{Index: projectionStepIndex}}})

	requireProjectionEqual(t, events[0].Type, StepTypeUnknown)
}

func requireProjectionEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
