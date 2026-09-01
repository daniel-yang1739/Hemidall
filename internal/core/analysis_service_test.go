package core

import "testing"

func TestAnalysisService_SessionMetricsAggregatesModelsAndDefaultCache(t *testing.T) {
	query := testSessionQuery{session: Session{Generations: []Generation{
		{ModelID: "model-a", Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 100, HasCachedInputTokens: true, CachedInputTokens: 90}},
		{ModelID: "model-a", Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 20}},
		{ModelID: "model-b", Usage: UsageObservation{HasUncachedInputTokens: true, UncachedInputTokens: 30, HasCachedInputTokens: true, CachedInputTokens: 10}},
	}}}
	metrics := NewAnalysisService(query).SessionMetrics()

	requireAnalysisEqual(t, metrics.TotalStats.TurnCount, 3)
	requireAnalysisEqual(t, metrics.TotalStats.UncachedInputTokenSum, 150)
	requireAnalysisEqual(t, metrics.TotalStats.CachedInputTokenSum, 100)
	requireAnalysisEqual(t, metrics.TotalStats.InferredZeroCacheTurnCount, 1)
	requireAnalysisEqual(t, metrics.ModelStats[0].ModelName, "model-a")
	requireAnalysisEqual(t, metrics.ModelStats[1].ModelName, "model-b")
}

func TestAnalysisService_SessionMetricsSkipsIncompleteUsage(t *testing.T) {
	query := testSessionQuery{session: Session{Generations: []Generation{
		{ModelID: "model-a", Usage: UsageObservation{HasObservedContextTokens: true, ObservedContextTokens: 160_000}},
	}}}
	metrics := NewAnalysisService(query).SessionMetrics()

	requireAnalysisEqual(t, metrics.TotalStats.TurnCount, 0)
	requireAnalysisEqual(t, len(metrics.ModelStats), 0)
}

type testSessionQuery struct {
	session Session
}

func (query testSessionQuery) Session() Session { return query.session }

func (query testSessionQuery) StepByIndex(int) (Step, bool) { return Step{}, false }

func (query testSessionQuery) GenerationByID(string) (Generation, bool) { return Generation{}, false }

func (query testSessionQuery) ModelByID(string) (ModelRecord, bool) { return ModelRecord{}, false }

func (query testSessionQuery) Models() ModelCatalog { return query.session.Models }

func requireAnalysisEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
