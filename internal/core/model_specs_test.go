package core

import "testing"

const (
	modelSpecComparableTurns      = 2
	modelSpecComparableTokens     = 1_000
	modelSpecCachedTokens         = 900
	modelSpecUncachedTokens       = 100
	modelSpecCacheMultiplier      = 0.10
	modelSpecEffectiveInputTokens = 190.0
)

func TestModelPricing_Pos_ProjectsExactVerifiedModel(t *testing.T) {
	stats := ModelTokenStats{
		ModelName:        gemini37FlashModelID,
		CachedTurnCount:  modelSpecComparableTurns,
		ComparableTokens: modelSpecComparableTokens,
		TotalCached:      modelSpecCachedTokens,
		TotalNew:         modelSpecUncachedTokens,
	}

	projection, available := ProjectCacheAdjustedInput(stats)

	requireModelSpecEqual(t, true, available)
	requireModelSpecEqual(t, gemini37FlashModelID, projection.ModelID)
	requireModelSpecFloatEqual(t, modelSpecCacheMultiplier, projection.CacheInputMultiplier)
	requireModelSpecFloatEqual(t, modelSpecEffectiveInputTokens, projection.EffectiveInputTokens)
}

func TestModelPricing_Neg_DoesNotProjectUnknownModel(t *testing.T) {
	stats := ModelTokenStats{
		ModelName:        "unknown",
		CachedTurnCount:  modelSpecComparableTurns,
		ComparableTokens: modelSpecComparableTokens,
		TotalCached:      modelSpecCachedTokens,
		TotalNew:         modelSpecUncachedTokens,
	}

	_, available := ProjectCacheAdjustedInput(stats)

	requireModelSpecEqual(t, false, available)
}

func requireModelSpecEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func requireModelSpecFloatEqual(t *testing.T, want, got float64) {
	t.Helper()
	if difference := want - got; difference > modelSpecFloatTolerance || difference < -modelSpecFloatTolerance {
		t.Fatalf("want %v, got %v", want, got)
	}
}

const modelSpecFloatTolerance = 0.000001
