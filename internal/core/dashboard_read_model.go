package core

import (
	"sort"
	"strconv"
	"strings"
)

// DashboardStepKind describes the presentation contract for one selected step.
// It is intentionally source-neutral: UI code never needs to inspect adapter
// records or infer whether an event caused a provider request.
type DashboardStepKind string

const (
	DashboardStepCloud      DashboardStepKind = "CLOUD_GENERATION"
	DashboardStepLocal      DashboardStepKind = "LOCAL_EXECUTION"
	DashboardStepUser       DashboardStepKind = "USER_INPUT"
	DashboardStepCheckpoint DashboardStepKind = "COMPACTION"
	DashboardStepSystem     DashboardStepKind = "SYSTEM_EVENT"
)

// ContextSectionEvidence tells the UI whether a token count came from decoded
// persisted evidence, a transcript reconstruction, or is unavailable.
type ContextSectionEvidence string

const (
	ContextSectionSnapshot      ContextSectionEvidence = "PERSISTED_SNAPSHOT"
	ContextSectionReconstructed ContextSectionEvidence = "TRANSCRIPT_RECONSTRUCTION"
	ContextSectionUnavailable   ContextSectionEvidence = "UNAVAILABLE"
)

// RequestContextSection is one non-overlapping, visible portion of the input
// evidence for one cloud generation. Tokens are local estimates only.
type RequestContextSection struct {
	Tokens   int
	Evidence ContextSectionEvidence
}

// RequestContextComposition is the four-part representation of context sent
// before a selected cloud generation. It is evidence-oriented rather than an
// assertion about an official HTTP request body.
type RequestContextComposition struct {
	SystemInstruction   RequestContextSection
	ToolSchemas         RequestContextSection
	ConversationContext RequestContextSection
	ActiveInput         RequestContextSection
	TotalVisibleTokens  int
}

// EmpiricalTurnTelemetry provides 100% authoritative telemetry metrics decoded
// directly from SQLite generation records (gen_metadata ModelUsage and ChatStartMetadata).
type EmpiricalTurnTelemetry struct {
	ObservedContextTokens int     `json:"observed_context_tokens"` // Input Context Tokens
	CachedContentTokens   int     `json:"cached_content_tokens"`   // F4.5
	UncachedPromptTokens  int     `json:"uncached_prompt_tokens"`  // F4.2
	ThinkingOutputTokens  int     `json:"thinking_output_tokens"`  // F4.3
	OutputContentTokens   int     `json:"output_content_tokens"`   // F4.9
	OutputTokens          int     `json:"output_tokens"`           // Thinking + Content
	TotalTokens           int     `json:"total_tokens"`            // ObservedContextTokens + OutputTokens
	ContextWindowLimit    int     `json:"context_window_limit"`    // F9.10.4 or model default
	UtilizationPercentage float64 `json:"utilization_percentage"`  // ObservedContextTokens / ContextWindowLimit * 100
	CacheHitPercentage    float64 `json:"cache_hit_percentage"`    // CachedContentTokens / ObservedContextTokens * 100
	TimeToFirstTokenMs    int64   `json:"time_to_first_token_ms"`  // F11
	StreamingDurationMs   int64   `json:"streaming_duration_ms"`   // F12
	UpstreamRequestID     string  `json:"upstream_request_id"`     // F4.11
}

// StepInspectionReadModel contains all data necessary to render the lower
// fixed-height dashboard panel for one transcript step.
type StepInspectionReadModel struct {
	Event              UnifiedAgentEvent
	Kind               DashboardStepKind
	RequestContext     *RequestContextComposition
	TurnTelemetry      *EmpiricalTurnTelemetry
	LocalAction        string
	ResultPreview      string
	TriggeredCloudStep int
}

// DashboardReadModel is the cached domain projection consumed by the dashboard
// view. It is built outside TUI rendering and is safe to query by step index.
type DashboardReadModel struct {
	SessionID   string
	Revision    uint64
	Metrics     SessionAggregateMetrics
	Inspections map[int]StepInspectionReadModel
}

const (
	noTriggeredCloudStep = 0
	noDashboardAction    = "unavailable"
)

// BuildDashboardReadModel assembles stable dashboard data from one parsed
// session. It performs all history token classification before UI rendering.
func BuildDashboardReadModel(session Session) DashboardReadModel {
	events := ProjectSessionEvents(session)
	inspections := make(map[int]StepInspectionReadModel, len(events))
	requestContexts := buildRequestContextTimeline(events, session.ContextSnapshots)
	actionsByParentStep := dashboardActionsByCloudStep(events)

	for _, event := range events {
		inspection := StepInspectionReadModel{
			Event: event,
			Kind:  dashboardStepKind(event),
		}
		if inspection.Kind == DashboardStepCloud {
			composition := requestContexts[event.StepIndex]
			inspection.RequestContext = &composition
			telemetry := BuildEmpiricalTurnTelemetry(event)
			inspection.TurnTelemetry = &telemetry
		} else {
			inspection.LocalAction = dashboardLocalAction(event, actionsByParentStep)
			inspection.ResultPreview = dashboardResultPreview(event)
			if event.ParentStepIdx > noTriggeredCloudStep {
				inspection.TriggeredCloudStep = event.ParentStepIdx
			}
		}
		inspections[event.StepIndex] = inspection
	}

	return DashboardReadModel{
		SessionID:   session.Ref.SessionID,
		Revision:    session.Revision,
		Metrics:     generationMetrics(session.Generations),
		Inspections: inspections,
	}
}

// BuildDashboardReadModelFromEvents provides a source-neutral fallback before
// an adapter-backed session has been hydrated.
func BuildDashboardReadModelFromEvents(sessionID string, events []UnifiedAgentEvent) DashboardReadModel {
	session := Session{
		Ref:   SessionRef{SessionID: sessionID},
		Steps: dashboardStepsFromEvents(events),
	}
	for _, event := range events {
		if !event.IsCloudStep() {
			continue
		}
		session.Generations = append(session.Generations, Generation{
			ID:           generationIDFromEvent(event),
			StepIndex:    event.StepIndex,
			HasStepIndex: true,
			Provider:     event.Usage.Provider,
			ModelID:      event.Usage.ModelName,
			Usage: UsageObservation{
				HasThinkingOutputTokens:  event.Usage.HasThinkingOutputTokens,
				HasOutputContentTokens:   event.Usage.HasOutputContentTokens,
				HasObservedContextTokens: event.Usage.HasObservedContextTokens,
				ObservedContextTokens:    event.Usage.ObservedContextTokens,
				HasUncachedInputTokens:   event.Usage.HasUncachedInputTokens,
				UncachedInputTokens:      event.Usage.UncachedInputTokens,
				HasCachedInputTokens:     event.Usage.HasCachedInputTokens,
				CachedInputTokens:        event.Usage.CachedInputTokens,
				HasContextLimit:          event.Usage.HasContextLimit,
				ContextLimit:             event.Usage.ContextLimit,
				ThinkingOutputTokens:     event.Usage.ThinkingOutputTokens,
				OutputContentTokens:      event.Usage.OutputContentTokens,
				TotalTokens:              event.Usage.TotalTokens,
				TimeToFirstTokenMs:       event.Usage.TimeToFirstTokenMs,
				StreamingDurationMs:      event.Usage.StreamingDurationMs,
				UpstreamRequestID:        event.Usage.UpstreamRequestID,
			},
		})
	}
	return BuildDashboardReadModel(session)
}

// BuildEmpiricalTurnTelemetry projects stored counters without local fallbacks.
// Availability remains on Event.Usage; missing fields must not render as zero.
func BuildEmpiricalTurnTelemetry(event UnifiedAgentEvent) EmpiricalTurnTelemetry {
	u := event.Usage
	result := EmpiricalTurnTelemetry{
		ObservedContextTokens: u.ObservedContextTokens,
		CachedContentTokens:   u.CachedInputTokens, UncachedPromptTokens: u.UncachedInputTokens,
		ThinkingOutputTokens: u.ThinkingOutputTokens, OutputContentTokens: u.OutputContentTokens,
		OutputTokens:       u.ThinkingOutputTokens + u.OutputContentTokens,
		TotalTokens:        u.UncachedInputTokens + u.CachedInputTokens + u.ThinkingOutputTokens + u.OutputContentTokens,
		ContextWindowLimit: u.ContextLimit, TimeToFirstTokenMs: u.TimeToFirstTokenMs,
		StreamingDurationMs: u.StreamingDurationMs, UpstreamRequestID: u.UpstreamRequestID,
	}
	if u.HasObservedContextTokens && u.HasContextLimit && u.ContextLimit > 0 {
		result.UtilizationPercentage = float64(u.ObservedContextTokens) / float64(u.ContextLimit) * cacheSharePercentageScale
	}
	if u.HasUncachedInputTokens && u.UncachedInputTokens+u.CachedInputTokens > 0 {
		result.CacheHitPercentage = float64(u.CachedInputTokens) / float64(u.UncachedInputTokens+u.CachedInputTokens) * cacheSharePercentageScale
	}
	return result
}

func dashboardStepsFromEvents(events []UnifiedAgentEvent) []Step {
	steps := make([]Step, 0, len(events))
	for _, event := range events {
		steps = append(steps, Step{
			Index:               event.StepIndex,
			Timestamp:           event.Timestamp,
			Source:              event.Source,
			Kind:                string(event.Type),
			Status:              event.Status,
			Content:             event.RawContent,
			Thinking:            event.Thinking,
			ToolCalls:           dashboardToolCallsFromEvent(event.ToolCalls),
			Scope:               event.Scope,
			AgentRole:           event.AgentRole,
			ParentStepIndex:     event.ParentStepIdx,
			PackagedInStepIndex: event.PackagedInStepIdx,
			ConsumedStepIndexes: append([]int(nil), event.ConsumedStepIndices...),
		})
	}
	return steps
}

func dashboardToolCallsFromEvent(calls []ToolCallInfo) []ToolCall {
	projected := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		projected = append(projected, ToolCall{Name: call.ToolName, Args: call.Arguments})
	}
	return projected
}

func generationIDFromEvent(event UnifiedAgentEvent) string {
	if event.Usage.GenerationIndex > 0 {
		return strconv.Itoa(event.Usage.GenerationIndex)
	}
	return ""
}

func dashboardStepKind(event UnifiedAgentEvent) DashboardStepKind {
	switch {
	case event.IsCloudStep():
		return DashboardStepCloud
	case event.IsCompactionStep():
		return DashboardStepCheckpoint
	case event.Type == StepTypeUserInput || event.Scope == ScopeUserInteraction:
		return DashboardStepUser
	case event.IsLocalStep() || event.Scope == ScopeLocalExecution:
		return DashboardStepLocal
	default:
		return DashboardStepSystem
	}
}

func dashboardActionsByCloudStep(events []UnifiedAgentEvent) map[int]string {
	actions := make(map[int]string)
	for _, event := range events {
		if !event.IsCloudStep() || len(event.ToolCalls) == 0 {
			continue
		}
		if event.ToolCalls[0].ToolName != "" {
			actions[event.StepIndex] = event.ToolCalls[0].ToolName
		}
	}
	return actions
}

func dashboardLocalAction(event UnifiedAgentEvent, actionsByParentStep map[int]string) string {
	if len(event.ToolCalls) > 0 && event.ToolCalls[0].ToolName != "" {
		return event.ToolCalls[0].ToolName
	}
	if action := actionsByParentStep[event.ParentStepIdx]; action != "" {
		return action
	}
	if event.Type != "" {
		return string(event.Type)
	}
	return noDashboardAction
}

func dashboardResultPreview(event UnifiedAgentEvent) string {
	if event.RawContent != "" {
		return event.RawContent
	}
	if event.Summary != "" {
		return event.Summary
	}
	return "No visible result content was persisted."
}

func buildRequestContextTimeline(events []UnifiedAgentEvent, snapshots []ContextSnapshot) map[int]RequestContextComposition {
	compositions := make(map[int]RequestContextComposition, len(events))
	snapshotByStep := snapshotsByGeneratedStep(snapshots)
	state := newRequestContextTimelineState()

	// Historical reconstruction never borrows a later generation's snapshot.

	for _, event := range events {
		if event.IsCloudStep() {
			// A tool call emitted by this generation proves that its schema had
			// to be available before the generation. Record it before projecting
			// this request's reconstructed input.
			state.observeToolDefinitions(event.ToolCalls)
			composition := state.composition(event.StepIndex)
			if snapshot, found := snapshotByStep[event.StepIndex]; found {
				composition = applySnapshotEvidence(composition, snapshot)
			}
			compositions[event.StepIndex] = composition
			state.consumeCloudGeneration(event)
			continue
		}
		state.consumeNonCloudEvent(event)
	}
	return compositions
}

func snapshotsByGeneratedStep(snapshots []ContextSnapshot) map[int]ContextSnapshot {
	byStep := make(map[int]ContextSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		if !snapshot.HasInputBoundary {
			continue
		}
		byStep[snapshot.InputBoundaryStepIndex+snapshotGeneratedStepOffset] = snapshot
	}
	return byStep
}

type requestContextTimelineState struct {
	systemText         string
	toolDefinitions    map[string]map[string]struct{}
	toolSchemaTokens   int
	conversationTokens int
	activeInputTokens  int
}

func newRequestContextTimelineState() requestContextTimelineState {
	return requestContextTimelineState{toolDefinitions: make(map[string]map[string]struct{})}
}

func (state *requestContextTimelineState) consumeNonCloudEvent(event UnifiedAgentEvent) {
	if event.Type == StepTypeSystemInit || event.Scope == ScopeSystemBootstrap || strings.Contains(event.RawContent, "<identity>") || strings.Contains(event.RawContent, "<user_rules>") {
		state.observeSystemInstruction(event)
		if event.Type == StepTypeSystemInit || event.Scope == ScopeSystemBootstrap {
			return
		}
	}
	if event.IsCompactionStep() || strings.Contains(event.RawContent, "<CONTEXT_SUMMARY>") {
		state.conversationTokens = playbackEventTokens(event)
		state.activeInputTokens = 0
		return
	}
	// Any newly observed event after the previous cloud generation is input to
	// the next cloud request. This includes explicit user input and local tool
	// results. It becomes conversation context only after that generation runs.
	state.activeInputTokens += playbackEventTokens(event)
}

func (state *requestContextTimelineState) consumeCloudGeneration(event UnifiedAgentEvent) {
	state.conversationTokens += state.activeInputTokens + playbackEventTokens(event)
	state.activeInputTokens = 0
}

func (state *requestContextTimelineState) observeToolDefinitions(calls []ToolCallInfo) {
	if !observeToolDefinitions(state.toolDefinitions, calls) {
		return
	}
	state.toolSchemaTokens = countToolDefinitionTokens(toolSignaturesFromObservedArguments(state.toolDefinitions))
}

func (state *requestContextTimelineState) observeSystemInstruction(event UnifiedAgentEvent) {
	identity := extractTranscriptTaggedSection(event.RawContent, "identity")
	rules := extractTranscriptTaggedSection(event.RawContent, "user_rules")
	state.systemText = strings.TrimSpace(identity + "\n" + rules)
}

func (state requestContextTimelineState) composition(stepIndex int) RequestContextComposition {
	systemTokens := CountTokens(state.systemText)
	return newRequestContextComposition(
		requestContextSection(systemTokens, ContextSectionReconstructed),
		requestContextSection(state.toolSchemaTokens, ContextSectionReconstructed),
		requestContextSection(state.conversationTokens, ContextSectionReconstructed),
		requestContextSection(state.activeInputTokens, ContextSectionReconstructed),
	)
}

func applySnapshotEvidence(composition RequestContextComposition, snapshot ContextSnapshot) RequestContextComposition {
	systemText := snapshot.SystemPrompt
	if systemText == "" {
		systemText = strings.TrimSpace(snapshot.IdentityPrompt + "\n" + snapshot.ConstitutionDoc)
	}
	historyTokens := 0
	for _, record := range snapshot.PersistedContextRecords {
		historyTokens += CountTokens(record.PrimaryText)
	}
	return newRequestContextComposition(
		requestContextSection(CountTokens(systemText), ContextSectionSnapshot),
		requestContextSection(countToolDefinitionTokens(snapshot.NativeTools), ContextSectionSnapshot),
		requestContextSection(historyTokens, ContextSectionSnapshot),
		composition.ActiveInput,
	)
}

func requestContextSection(tokens int, evidence ContextSectionEvidence) RequestContextSection {
	if tokens <= 0 {
		return RequestContextSection{Evidence: ContextSectionUnavailable}
	}
	return RequestContextSection{Tokens: tokens, Evidence: evidence}
}

func newRequestContextComposition(system, tools, conversation, input RequestContextSection) RequestContextComposition {
	total := system.Tokens + tools.Tokens + conversation.Tokens + input.Tokens
	return RequestContextComposition{
		SystemInstruction:   system,
		ToolSchemas:         tools,
		ConversationContext: conversation,
		ActiveInput:         input,
		TotalVisibleTokens:  total,
	}
}

// SortDashboardInspections returns stable step indexes for consumers that need
// deterministic presentation without coupling to the backing map.
func SortDashboardInspections(inspections map[int]StepInspectionReadModel) []int {
	indexes := make([]int, 0, len(inspections))
	for index := range inspections {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes
}
