package core

import "testing"

const (
	dashboardModelSystemStep      = 0
	dashboardModelUserStep        = 1
	dashboardModelFirstCloudStep  = 2
	dashboardModelLocalStep       = 3
	dashboardModelSecondCloudStep = 4
	dashboardModelSnapshotIndex   = 42
)

func TestBuildDashboardReadModel_Pos_ClassifiesSnapshotAndLocalExecution(t *testing.T) {
	readModel := BuildDashboardReadModel(dashboardReadModelFixture(true))

	snapshotCloud := readModel.Inspections[dashboardModelSecondCloudStep]
	localStep := readModel.Inspections[dashboardModelLocalStep]

	requireDashboardStepKind(t, snapshotCloud.Kind, DashboardStepCloud)
	requireDashboardSnapshotSections(t, *snapshotCloud.RequestContext)
	requireDashboardStepKind(t, localStep.Kind, DashboardStepLocal)
	requireDashboardTriggeredCloudStep(t, localStep.TriggeredCloudStep, dashboardModelFirstCloudStep)
	requireDashboardString(t, localStep.LocalAction, "run_cmd")
	requireDashboardString(t, localStep.ResultPreview, "command output")
}

func TestBuildDashboardReadModel_Pos_UsesLocalResultAsNextCloudActiveInput(t *testing.T) {
	readModel := BuildDashboardReadModel(dashboardReadModelFixture(false))
	composition := *readModel.Inspections[dashboardModelSecondCloudStep].RequestContext

	requireDashboardStepKind(t, readModel.Inspections[dashboardModelSecondCloudStep].Kind, DashboardStepCloud)
	requireDashboardEvidence(t, composition.SystemInstruction.Evidence, ContextSectionReconstructed)
	requireDashboardEvidence(t, composition.ToolSchemas.Evidence, ContextSectionReconstructed)
	requireDashboardEvidence(t, composition.ConversationContext.Evidence, ContextSectionReconstructed)
	requireDashboardEvidence(t, composition.ActiveInput.Evidence, ContextSectionReconstructed)
	requireDashboardCompositionTotal(t, composition)
}

func TestBuildDashboardReadModel_Pos_UsesLatestUserMessageAsActiveInput(t *testing.T) {
	readModel := BuildDashboardReadModel(dashboardReadModelFixture(false))
	composition := *readModel.Inspections[dashboardModelFirstCloudStep].RequestContext

	requireDashboardEvidence(t, composition.ActiveInput.Evidence, ContextSectionReconstructed)
	requireDashboardCompositionTotal(t, composition)
}

func TestBuildDashboardReadModel_Pos_PreservesReconstructedActiveInputAlongsideSnapshot(t *testing.T) {
	readModel := BuildDashboardReadModel(dashboardReadModelFixture(true))
	composition := *readModel.Inspections[dashboardModelSecondCloudStep].RequestContext

	requireDashboardEvidence(t, composition.SystemInstruction.Evidence, ContextSectionSnapshot)
	requireDashboardEvidence(t, composition.ActiveInput.Evidence, ContextSectionReconstructed)
	requireDashboardCompositionTotal(t, composition)
}

func TestBuildDashboardReadModel_Boundary_LeavesMissingTriggeredCloudStepUnavailable(t *testing.T) {
	session := dashboardReadModelFixture(false)
	session.Steps[dashboardModelLocalStep].ParentStepIndex = 0
	readModel := BuildDashboardReadModel(session)

	requireDashboardTriggeredCloudStep(t, readModel.Inspections[dashboardModelLocalStep].TriggeredCloudStep, noTriggeredCloudStep)
}

func dashboardReadModelFixture(includeSnapshot bool) Session {
	session := Session{
		Ref:      SessionRef{SessionID: "dashboard-session"},
		Revision: 7,
		Steps: []Step{
			{Index: dashboardModelSystemStep, Kind: string(StepTypeSystemInit), Scope: ScopeSystemBootstrap, Content: "<identity>assistant</identity><user_rules>rules</user_rules>"},
			{Index: dashboardModelUserStep, Kind: string(StepTypeUserInput), Scope: ScopeUserInteraction, Content: "Explain the observation."},
			{Index: dashboardModelFirstCloudStep, Kind: string(StepTypeModelResponse), Scope: ScopeCloudInference, Content: "I will inspect the session.", ToolCalls: []ToolCall{{Name: "run_cmd", Args: map[string]any{"cmd": "pwd"}}}},
			{Index: dashboardModelLocalStep, Kind: string(StepTypeRunCommand), Scope: ScopeLocalExecution, Content: "command output", ParentStepIndex: dashboardModelFirstCloudStep},
			{Index: dashboardModelSecondCloudStep, Kind: string(StepTypeModelResponse), Scope: ScopeCloudInference, Content: "The command completed."},
		},
		Generations: []Generation{
			{ID: "1", StepIndex: dashboardModelFirstCloudStep, ModelID: "gemini-3.7-flash", Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 10, HasCachedInputTokens: true, CachedInputTokens: 90}},
			{ID: "2", StepIndex: dashboardModelSecondCloudStep, ModelID: "gemini-3.7-flash", Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 20, HasCachedInputTokens: true, CachedInputTokens: 80}},
		},
	}
	if includeSnapshot {
		session.ContextSnapshots = []ContextSnapshot{{
			GenerationIndex:         dashboardModelSnapshotIndex,
			InputBoundaryStepIndex:  dashboardModelLocalStep,
			HasInputBoundary:        true,
			SystemPrompt:            "Persisted system instruction.",
			NativeTools:             []ToolSignature{{RawSchema: `{"name":"run_cmd","parameters":{"cmd":"string"}}`}},
			PersistedContextRecords: []PersistedContextRecord{{PrimaryText: "Persisted compacted history."}},
		}}
	}
	return session
}

func requireDashboardStepKind(t *testing.T, got, want DashboardStepKind) {
	t.Helper()
	if got != want {
		t.Fatalf("dashboard step kind: got %q, want %q", got, want)
	}
}

func requireDashboardEvidence(t *testing.T, got, want ContextSectionEvidence) {
	t.Helper()
	if got != want {
		t.Fatalf("section evidence: got %q, want %q", got, want)
	}
}

func requireDashboardString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string value: got %q, want %q", got, want)
	}
}

func requireDashboardTriggeredCloudStep(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("triggered cloud step: got %d, want %d", got, want)
	}
}

func requireDashboardSnapshotSections(t *testing.T, composition RequestContextComposition) {
	requireDashboardEvidence(t, composition.SystemInstruction.Evidence, ContextSectionSnapshot)
	requireDashboardEvidence(t, composition.ToolSchemas.Evidence, ContextSectionSnapshot)
	requireDashboardEvidence(t, composition.ConversationContext.Evidence, ContextSectionSnapshot)
	requireDashboardEvidence(t, composition.ActiveInput.Evidence, ContextSectionReconstructed)
}

func requireDashboardCompositionTotal(t *testing.T, composition RequestContextComposition) {
	want := composition.SystemInstruction.Tokens + composition.ToolSchemas.Tokens + composition.ConversationContext.Tokens + composition.ActiveInput.Tokens
	if composition.TotalVisibleTokens != want {
		t.Fatalf("composition total: got %d, want %d", composition.TotalVisibleTokens, want)
	}
}
