package core

import "testing"

func TestVisibleContextEvidenceEstimate_Pos_CountsVisibleSnapshotDimensions(t *testing.T) {
	const inputBoundaryStep = 42
	const generatedStep = inputBoundaryStep + snapshotGeneratedStepOffset
	payload := AgentContextPayload{
		SnapshotAvailable:         true,
		SnapshotGenIndex:          7,
		SnapshotInputBoundaryStep: inputBoundaryStep,
		SnapshotHasInputBoundary:  true,
		SystemPrompt:              "You are a careful engineering assistant.",
		NativeTools:               []ToolSignature{{RawSchema: `{"name":"read_file","required":["path"]}`}},
		PersistedRecords:          []PersistedContextRecord{{PrimaryText: "A persisted history record."}},
		StagedBuffers:             "tool output buffer",
		LatestPrompt:              "Please explain the result.",
	}

	estimate := EstimateVisibleContextEvidence(payload)

	requireContextEstimateAvailable(t, estimate, true)
	requireContextEstimateSnapshotAnchor(t, estimate, generatedStep)
	requireContextEstimateDimension(t, estimate.SystemTokens)
	requireContextEstimateDimension(t, estimate.ToolsTokens)
	requireContextEstimateDimension(t, estimate.ToolBufferTokens)
	requireContextEstimateDimension(t, estimate.HistoryTokens)
	requireContextEstimateDimension(t, estimate.InboundTokens)
	requireContextEstimateTotal(t, estimate)
}

func requireContextEstimateSnapshotAnchor(t *testing.T, estimate VisibleContextEvidenceEstimate, wantStep int) {
	t.Helper()
	if !estimate.HasSnapshotGeneratedStep || estimate.SnapshotGeneratedStepIndex != wantStep {
		t.Fatalf("snapshot generated step: got (%d, %t), want (%d, true)", estimate.SnapshotGeneratedStepIndex, estimate.HasSnapshotGeneratedStep, wantStep)
	}
}

func TestVisibleContextEvidenceEstimate_Neg_MarksTranscriptFallbackAsNotSnapshotEvidence(t *testing.T) {
	payload := AgentContextPayload{LatestPrompt: "Transcript-only prompt"}

	estimate := EstimateVisibleContextEvidence(payload)

	requireContextEstimateAvailable(t, estimate, true)
	requireContextEstimateSource(t, estimate, ContextEvidenceTranscriptFallback)
	requireContextEstimateDimension(t, estimate.InboundTokens)
}

func TestBuildPlaybackContextEvidence_Pos_TracksEachSelectedStep(t *testing.T) {
	const systemStep = 0
	const firstPromptStep = 1
	const toolStep = 2
	const cloudStep = 3
	const latestPromptStep = 4
	history := []UnifiedAgentEvent{
		{StepIndex: systemStep, Type: StepTypeSystemInit, RawContent: "<identity>assistant</identity><user_rules>english</user_rules>"},
		{StepIndex: firstPromptStep, Type: StepTypeUserInput, RawContent: "first question"},
		{StepIndex: toolStep, Type: StepTypeRunCommand, RawContent: "tool output", ToolCalls: []ToolCallInfo{{ToolName: "run_cmd", Arguments: map[string]interface{}{"cmd": "pwd"}}}},
		{StepIndex: cloudStep, Type: StepTypeModelResponse, Scope: ScopeCloudInference, RawContent: "model response"},
		{StepIndex: latestPromptStep, Type: StepTypeUserInput, RawContent: "latest question"},
	}

	estimates := BuildPlaybackContextEvidence(history)

	requirePlaybackEstimateCount(t, estimates, len(history))
	requirePlaybackEstimateStep(t, estimates[systemStep], systemStep)
	requirePlaybackEstimateSource(t, estimates[firstPromptStep], ContextEvidenceTranscriptFallback)
	requireContextEstimateDimension(t, estimates[firstPromptStep].InboundTokens)
	requireContextEstimateDimension(t, estimates[toolStep].ToolBufferTokens)
	requireContextEstimateDimension(t, estimates[cloudStep].ToolBufferTokens)
	requirePlaybackEstimateGenerationInput(t, estimates[cloudStep])
	requirePlaybackEstimateStep(t, estimates[latestPromptStep], latestPromptStep)
	requireContextEstimateDimension(t, estimates[latestPromptStep].InboundTokens)
}

func requireContextEstimateAvailable(t *testing.T, estimate VisibleContextEvidenceEstimate, want bool) {
	t.Helper()
	if estimate.Available != want {
		t.Fatalf("estimate availability: got %t, want %t", estimate.Available, want)
	}
}

func requireContextEstimateSource(t *testing.T, estimate VisibleContextEvidenceEstimate, want ContextEvidenceKind) {
	t.Helper()
	if estimate.SourceKind != want {
		t.Fatalf("estimate source: got %q, want %q", estimate.SourceKind, want)
	}
}

func requirePlaybackEstimateCount(t *testing.T, estimates []VisibleContextEvidenceEstimate, want int) {
	t.Helper()
	if len(estimates) != want {
		t.Fatalf("playback estimate count: got %d, want %d", len(estimates), want)
	}
}

func requirePlaybackEstimateStep(t *testing.T, estimate VisibleContextEvidenceEstimate, want int) {
	t.Helper()
	if !estimate.Available || estimate.SelectedStepIndex != want {
		t.Fatalf("playback estimate step: got (available=%t, step=%d), want (true, %d)", estimate.Available, estimate.SelectedStepIndex, want)
	}
}

func requirePlaybackEstimateSource(t *testing.T, estimate VisibleContextEvidenceEstimate, want ContextEvidenceKind) {
	t.Helper()
	if estimate.SourceKind != want {
		t.Fatalf("playback estimate source: got %q, want %q", estimate.SourceKind, want)
	}
}

func requirePlaybackEstimateGenerationInput(t *testing.T, estimate VisibleContextEvidenceEstimate) {
	t.Helper()
	if !estimate.IsGenerationInputEstimate {
		t.Fatalf("expected selected cloud step to use preceding generation input evidence")
	}
}

func requireContextEstimateDimension(t *testing.T, tokens int) {
	t.Helper()
	if tokens <= 0 {
		t.Fatalf("expected positive token estimate, got %d", tokens)
	}
}

func requireContextEstimateTotal(t *testing.T, estimate VisibleContextEvidenceEstimate) {
	t.Helper()
	want := estimate.SystemTokens + estimate.ToolsTokens + estimate.ToolBufferTokens + estimate.HistoryTokens + estimate.InboundTokens
	if estimate.TotalTokens != want {
		t.Fatalf("estimate total: got %d, want %d", estimate.TotalTokens, want)
	}
}
