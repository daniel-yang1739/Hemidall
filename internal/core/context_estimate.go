package core

import "strings"

// VisibleContextEvidenceEstimate is a local cl100k_base estimate of readable
// evidence displayed in Context. It combines snapshot text with selected
// transcript observations and is deliberately separate from persisted
// generation totals: private fields, protobuf framing, tokenizer differences,
// and omitted fields prevent an equality claim.
type VisibleContextEvidenceEstimate struct {
	Available                  bool
	SourceKind                 ContextEvidenceKind
	SelectedStepIndex          int
	IsGenerationInputEstimate  bool
	SnapshotGenIndex           int
	SnapshotGeneratedStepIndex int
	HasSnapshotGeneratedStep   bool
	SystemTokens               int
	ToolsTokens                int
	ToolBufferTokens           int
	HistoryTokens              int
	InboundTokens              int
	TotalTokens                int
}

const snapshotGeneratedStepOffset = 1

// EstimateVisibleContextEvidence counts only readable Context evidence. It does
// not reconstruct an outbound request.
func EstimateVisibleContextEvidence(payload AgentContextPayload) VisibleContextEvidenceEstimate {
	estimate := VisibleContextEvidenceEstimate{
		SourceKind:       payload.SourceKind,
		SnapshotGenIndex: payload.SnapshotGenIndex,
	}
	if estimate.SourceKind == "" {
		if payload.SnapshotAvailable {
			estimate.SourceKind = ContextEvidencePersistedSnapshot
		} else {
			estimate.SourceKind = ContextEvidenceTranscriptFallback
		}
	}
	if payload.SnapshotHasInputBoundary {
		estimate.SnapshotGeneratedStepIndex = payload.SnapshotInputBoundaryStep + snapshotGeneratedStepOffset
		estimate.HasSnapshotGeneratedStep = true
	}
	systemText := payload.SystemPrompt
	if systemText == "" {
		systemText = payload.IdentityPrompt + "\n" + payload.ConstitutionDoc
	}
	estimate.SystemTokens = CountTokens(systemText)

	for _, tool := range payload.NativeTools {
		toolText := tool.RawSchema
		if toolText == "" {
			toolText = tool.Signature + "\n" + tool.Description
		}
		estimate.ToolsTokens += CountTokens(toolText)
	}

	if len(payload.PersistedRecords) > 0 {
		for _, record := range payload.PersistedRecords {
			estimate.HistoryTokens += CountTokens(record.PrimaryText)
		}
	} else {
		estimate.HistoryTokens = CountTokens(payload.CheckpointSummary)
		for _, event := range payload.ActiveHistoryTurns {
			estimate.HistoryTokens += CountTokens(event.RawContent)
		}
	}

	estimate.ToolBufferTokens = CountTokens(payload.StagedBuffers)
	estimate.InboundTokens = CountTokens(payload.LatestPrompt)
	estimate.TotalTokens = estimate.SystemTokens + estimate.ToolsTokens + estimate.ToolBufferTokens + estimate.HistoryTokens + estimate.InboundTokens
	estimate.Available = payload.SnapshotAvailable || estimate.TotalTokens > 0
	return estimate
}

// BuildPlaybackContextEvidence creates one local five-dimension estimate per
// transcript event. Each item represents evidence observed through that event,
// so Dashboard playback can move without opening SQLite or recounting the
// entire transcript. It is not an outbound-request reconstruction.
func BuildPlaybackContextEvidence(history []UnifiedAgentEvent) []VisibleContextEvidenceEstimate {
	estimates := make([]VisibleContextEvidenceEstimate, 0, len(history))
	state := newPlaybackContextEvidenceState()
	for _, event := range history {
		if event.IsCloudStep() {
			estimates = append(estimates, state.estimate(event.StepIndex, true))
			state.consume(event)
			continue
		}
		state.consume(event)
		estimates = append(estimates, state.estimate(event.StepIndex, false))
	}
	return estimates
}

type playbackContextEvidenceState struct {
	identityPrompt     string
	constitutionDoc    string
	systemTokens       int
	observedTools      map[string]map[string]struct{}
	toolsTokens        int
	historyTokens      int
	toolBufferTokens   int
	latestInboundToken int
}

func newPlaybackContextEvidenceState() playbackContextEvidenceState {
	return playbackContextEvidenceState{observedTools: make(map[string]map[string]struct{})}
}

func (state *playbackContextEvidenceState) consume(event UnifiedAgentEvent) {
	state.observeSystemSections(event)
	if observeToolDefinitions(state.observedTools, event.ToolCalls) {
		state.toolsTokens = countToolDefinitionTokens(toolSignaturesFromObservedArguments(state.observedTools))
	}

	eventTokens := playbackEventTokens(event)
	switch {
	case event.Type == StepTypeSystemInit || event.StepIndex == 0:
		return
	case event.Type == StepTypeCheckpoint || strings.Contains(event.RawContent, "<CONTEXT_SUMMARY>"):
		state.historyTokens = eventTokens
		state.toolBufferTokens = 0
	case event.Type == StepTypeUserInput:
		state.historyTokens += state.latestInboundToken
		state.latestInboundToken = eventTokens
	case event.IsCloudStep():
		state.historyTokens += eventTokens
		state.toolBufferTokens = 0
	case event.IsLocalStep():
		state.toolBufferTokens += eventTokens
	default:
		state.historyTokens += eventTokens
	}
}

func (state *playbackContextEvidenceState) observeSystemSections(event UnifiedAgentEvent) {
	if event.Type != StepTypeSystemInit && event.StepIndex != 0 {
		return
	}
	if identity := extractTranscriptTaggedSection(event.RawContent, "identity"); identity != "" {
		state.identityPrompt = identity
	}
	if constitution := extractTranscriptTaggedSection(event.RawContent, "user_rules"); constitution != "" {
		state.constitutionDoc = constitution
	}
	state.systemTokens = CountTokens(state.identityPrompt + "\n" + state.constitutionDoc)
}

func (state playbackContextEvidenceState) estimate(stepIndex int, isGenerationInput bool) VisibleContextEvidenceEstimate {
	totalTokens := state.systemTokens + state.toolsTokens + state.toolBufferTokens + state.historyTokens + state.latestInboundToken
	return VisibleContextEvidenceEstimate{
		Available:                 true,
		SourceKind:                ContextEvidenceTranscriptFallback,
		SelectedStepIndex:         stepIndex,
		IsGenerationInputEstimate: isGenerationInput,
		SystemTokens:              state.systemTokens,
		ToolsTokens:               state.toolsTokens,
		ToolBufferTokens:          state.toolBufferTokens,
		HistoryTokens:             state.historyTokens,
		InboundTokens:             state.latestInboundToken,
		TotalTokens:               totalTokens,
	}
}

func playbackEventTokens(event UnifiedAgentEvent) int {
	if event.Tokens.StepDelta > 0 {
		return event.Tokens.StepDelta
	}
	return CountTokens(event.RawContent) + CountTokens(event.Thinking) + estimateToolArguments(event.ToolCalls)
}

func countToolDefinitionTokens(tools []ToolSignature) int {
	total := 0
	for _, tool := range tools {
		toolText := tool.RawSchema
		if toolText == "" {
			toolText = tool.Signature + "\n" + tool.Description
		}
		total += CountTokens(toolText)
	}
	return total
}
