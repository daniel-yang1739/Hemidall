package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const (
	reportTestFirstStep  = 1
	reportTestSecondStep = 2
	reportTestThirdStep  = 3
	reportTestInput      = 100
	reportTestContext    = 200
	reportTestGrowth     = 100
	reportTestRevision   = 7
	reportTestModel      = "gemini-3.7-flash"
)

func TestReportPreservesMissingZeroAndDerivedMeasurements(t *testing.T) {
	cases := []struct {
		name            string
		usage           UsageObservation
		cachedKind      string
		outputAvailable bool
		costAvailable   bool
		partial         bool
	}{
		{"missing", UsageObservation{}, MeasurementUnavailable, false, false, true},
		{"observed-zero", UsageObservation{HasUncachedInputTokens: true, HasCachedInputTokens: true, HasThinkingOutputTokens: true, HasOutputContentTokens: true}, MeasurementObserved, true, true, false},
		{"inferred-cache", UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: reportTestInput}, MeasurementDerived, false, true, true},
		{"cache-only", UsageObservation{HasCachedInputTokens: true, CachedInputTokens: reportTestInput}, MeasurementObserved, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := analyzeGeneration(Generation{ID: "1", ModelID: reportTestModel, Usage: tc.usage})
			if item.CachedInput.Kind != tc.cachedKind || item.ContentOutput.Available != tc.outputAvailable || item.EstimatedCost.Available != tc.costAvailable || item.CostPartial != tc.partial {
				t.Fatalf("incorrect field provenance: %+v", item)
			}
		})
	}
}

func TestReportCountsGenerationsNotStepsAndReplacesDuplicateIDs(t *testing.T) {
	session := Session{Ref: SessionRef{SessionID: "report"}, Generations: []Generation{
		{ID: "1", StepIndex: reportTestFirstStep, ModelID: reportTestModel, Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 10}},
		{ID: "2", StepIndex: reportTestFirstStep, ModelID: reportTestModel, Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 20}},
		{ID: "1", StepIndex: reportTestFirstStep, ModelID: reportTestModel, Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 30}},
	}, Steps: []Step{{Index: reportTestFirstStep, Kind: string(StepTypeModelResponse)}}}
	report := BuildSessionAnalysisReport(session)
	dashboard := BuildDashboardReadModel(session)
	if report.Metrics.TotalStats.TurnCount != 2 || report.Metrics.TotalStats.UncachedInputTokenSum != 50 {
		t.Fatalf("generation upsert/dedup failed: %+v", report.Metrics)
	}
	if !reflect.DeepEqual(report.Metrics, dashboard.Metrics) {
		t.Fatal("report and dashboard disagree")
	}
	if report.TokenConsumers[0].GenerationID != "1" {
		t.Fatal("incorrect consumption ranking")
	}
}

func TestContextGrowthDoesNotCrossEvidenceBoundaries(t *testing.T) {
	cases := []struct {
		name        string
		nextID      string
		nextModel   string
		hasContext  bool
		checkpoints []Step
		expected    int
	}{
		{"comparable", "2", reportTestModel, true, nil, 1},
		{"missing-field", "2", reportTestModel, false, nil, 0},
		{"model-switch", "2", "other-model", true, nil, 0},
		{"generation-gap", "3", reportTestModel, true, nil, 0},
		{"checkpoint", "2", reportTestModel, true, []Step{{Index: reportTestSecondStep, Kind: string(StepTypeCheckpoint)}}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := BuildSessionAnalysisReport(Session{Steps: tc.checkpoints, Generations: []Generation{
				{ID: "1", StepIndex: reportTestFirstStep, ModelID: reportTestModel, Usage: UsageObservation{HasObservedContextTokens: true, ObservedContextTokens: reportTestInput}},
				{ID: tc.nextID, StepIndex: reportTestThirdStep, ModelID: tc.nextModel, Usage: UsageObservation{HasObservedContextTokens: tc.hasContext, ObservedContextTokens: reportTestContext}},
			}})
			if len(report.ContextGrowth) != tc.expected {
				t.Fatalf("growth crossed boundary: %+v", report.ContextGrowth)
			}
		})
	}
}

func TestReportExcludesPayloadAndMarksUnknownPrices(t *testing.T) {
	report := BuildSessionAnalysisReport(Session{Steps: []Step{{Index: reportTestFirstStep, Kind: string(StepTypeRunCommand), Content: "PRIVATE_TOOL_OUTPUT"}}, Generations: []Generation{{ID: "1", ModelID: "unknown-model", Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: reportTestInput}}}})
	var output bytes.Buffer
	if err := WriteAnalysisReport(&output, report, "json"); err != nil {
		t.Fatal(err)
	}
	var decoded SessionAnalysisReport
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "PRIVATE_TOOL_OUTPUT") || decoded.SchemaVersion != ReportSchemaVersion {
		t.Fatal("export leaked content or lost schema version")
	}
	if len(decoded.CostConsumers) != 0 || !decoded.Coverage.CostPartial || decoded.Metrics.TotalStats.HasEstimatedCost {
		t.Fatal("unknown price must remain unavailable")
	}
	if len(decoded.ToolOutputs) != 1 || decoded.ToolOutputs[0].Amount.Kind != MeasurementEstimated {
		t.Fatal("tool output estimate missing")
	}
}

func TestReportWriterPropagatesErrors(t *testing.T) {
	cases := []string{"json", "text", "unsupported"}
	for _, format := range cases {
		t.Run(format, func(t *testing.T) {
			if err := WriteAnalysisReport(failingReportWriter{}, SessionAnalysisReport{}, format); err == nil {
				t.Fatal("expected write/format failure")
			}
		})
	}
}

type failingReportWriter struct{}

func (failingReportWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestAnalysisCacheIsRevisionScopedAndIsolated(t *testing.T) {
	source := &testSessionSource{session: Session{Ref: SessionRef{SessionID: "cache"}, Revision: reportTestRevision, Steps: []Step{{Index: reportTestFirstStep, Kind: string(StepTypeRunCommand), Content: "output"}}}}
	query := NewQueryService(source)
	service := NewAnalysisService(query)
	first := service.Report()
	first.ToolOutputs[0].Amount.Value = -1
	second := service.Report()
	if second.ToolOutputs[0].Amount.Value < 0 || second.Revision != reportTestRevision {
		t.Fatal("report mutation escaped cache boundary")
	}
	source.session = Session{Ref: SessionRef{SessionID: "cache"}, Revision: reportTestRevision + 1}
	source.delta = SessionDelta{Revision: reportTestRevision + 1}
	if _, err := query.Refresh(); err != nil {
		t.Fatal(err)
	}
	third := service.Report()
	if len(third.ToolOutputs) != 0 || third.Revision != reportTestRevision+1 {
		t.Fatal("cache failed to invalidate")
	}
}

func TestSessionSnapshotDeepIsolation(t *testing.T) {
	source := &testSessionSource{session: Session{Steps: []Step{{ToolCalls: []ToolCall{{Args: map[string]any{"items": []any{map[string]any{"value": "original"}}}}}, ConsumedStepIndexes: []int{reportTestFirstStep}}}, ContextSnapshots: []ContextSnapshot{{NativeTools: []ToolSignature{{Required: []string{"original"}}}}}}}
	query := NewQueryService(source)
	first := query.Session()
	first.Steps[0].ToolCalls[0].Args["items"].([]any)[0].(map[string]any)["value"] = "changed"
	first.Steps[0].ConsumedStepIndexes[0] = reportTestSecondStep
	first.ContextSnapshots[0].NativeTools[0].Required[0] = "changed"
	second := query.Session()
	if second.Steps[0].ToolCalls[0].Args["items"].([]any)[0].(map[string]any)["value"] != "original" || second.Steps[0].ConsumedStepIndexes[0] != reportTestFirstStep || second.ContextSnapshots[0].NativeTools[0].Required[0] != "original" {
		t.Fatal("nested mutation escaped snapshot boundary")
	}
}

func TestHistoricalContextDoesNotBorrowFutureSnapshot(t *testing.T) {
	session := dashboardReadModelFixture(false)
	before := BuildDashboardReadModel(session).Inspections[dashboardModelFirstCloudStep].RequestContext
	session.ContextSnapshots = []ContextSnapshot{{GenerationID: "2", HasInputBoundary: true, InputBoundaryStepIndex: dashboardModelLocalStep, SystemPrompt: strings.Repeat("future-only ", reportTestInput)}}
	after := BuildDashboardReadModel(session).Inspections[dashboardModelFirstCloudStep].RequestContext
	if !reflect.DeepEqual(before, after) {
		t.Fatal("future snapshot changed historical context")
	}
}

func BenchmarkSessionAnalysis(b *testing.B) {
	GetTokenizer()
	cases := []struct {
		name string
		size int
	}{{"1000", 1000}, {"10000", 10000}}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			session := benchmarkSession(tc.size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				BuildSessionAnalysisReport(session)
			}
		})
	}
}

func benchmarkSession(size int) Session {
	session := Session{Steps: make([]Step, size)}
	for i := range session.Steps {
		session.Steps[i] = Step{Index: i, Kind: string(StepTypeRunCommand), Content: "example command output with useful diagnostic details"}
	}
	return session
}
