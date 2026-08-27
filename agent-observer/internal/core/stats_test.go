package core_test

import (
	"testing"

	"agent-observer/internal/core"
)

func TestGetModelDiscount(t *testing.T) {
	tests := []struct {
		model    string
		wantDisc float64
		wantFact float64
	}{
		{"gemini-3.7-flash", 0.75, 0.25},
		{"gemini-2.5-pro", 0.75, 0.25},
		{"claude-3-7-sonnet-20250219", 0.90, 0.10},
		{"gpt-4o", 0.50, 0.50},
		{"deepseek-r1", 0.90, 0.10},
		{"unknown-model", 0.75, 0.25},
	}

	for _, tt := range tests {
		disc, fact, label := core.GetModelDiscount(tt.model)
		if disc != tt.wantDisc || fact != tt.wantFact {
			t.Errorf("GetModelDiscount(%q) = (%v, %v), want (%v, %v)", tt.model, disc, fact, tt.wantDisc, tt.wantFact)
		}
		if label == "" {
			t.Errorf("GetModelDiscount(%q) returned empty label", tt.model)
		}
	}
}

func TestComputeSessionAggregateMetrics(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{
			StepIndex: 1,
			Scope:     core.ScopeUserInteraction,
			Type:      core.StepTypeUserInput,
			Tokens: core.TokenBreakdown{
				TotalTokens: 0,
			},
		},
		{
			StepIndex: 2,
			Scope:     core.ScopeCloudInference,
			Type:      core.StepTypeModelResponse,
			Tokens: core.TokenBreakdown{
				OfficialModel: "gemini-3.7-flash",
				TotalTokens:   100000,
				CachedTokens:  80000,
				NewTokens:     20000,
				CacheHitRate:  80.0,
			},
		},
		{
			StepIndex: 3,
			Scope:     core.ScopeLocalExecution,
			Type:      core.StepTypeRunCommand,
			Tokens: core.TokenBreakdown{
				TotalTokens: 0,
			},
		},
		{
			StepIndex: 4,
			Scope:     core.ScopeCloudInference,
			Type:      core.StepTypeModelResponse,
			Tokens: core.TokenBreakdown{
				OfficialModel: "claude-3.7-sonnet",
				TotalTokens:   50000,
				CachedTokens:  40000,
				NewTokens:     10000,
				CacheHitRate:  80.0,
			},
		},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)

	// Total processed: 100k + 50k = 150k
	if metrics.TotalStats.TotalProcessed != 150000 {
		t.Fatalf("TotalProcessed = %d, want 150000", metrics.TotalStats.TotalProcessed)
	}
	// Total cached: 80k + 40k = 120k
	if metrics.TotalStats.TotalCached != 120000 {
		t.Fatalf("TotalCached = %d, want 120000", metrics.TotalStats.TotalCached)
	}
	// Total new: 20k + 10k = 30k
	if metrics.TotalStats.TotalNew != 30000 {
		t.Fatalf("TotalNew = %d, want 30000", metrics.TotalStats.TotalNew)
	}

	// Model 1 (Gemini): Effective = 20k + (80k * 0.25) = 40k. Saved = 60k (60.0%)
	// Model 2 (Claude): Effective = 10k + (40k * 0.10) = 14k. Saved = 36k (72.0%)
	// Weighted total effective = 40k + 14k = 54k
	// Total saved = 150k - 54k = 96k (64.0%)

	if metrics.TotalStats.EffectiveTokens != 54000 {
		t.Errorf("EffectiveTokens = %d, want 54000", metrics.TotalStats.EffectiveTokens)
	}
	if metrics.TotalStats.TokensSaved != 96000 {
		t.Errorf("TokensSaved = %d, want 96000", metrics.TotalStats.TokensSaved)
	}
	if len(metrics.ModelStats) != 2 {
		t.Fatalf("ModelStats count = %d, want 2", len(metrics.ModelStats))
	}
}

func TestExtractTurnTrendSeries(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{
			StepIndex: 1,
			Scope:     core.ScopeCloudInference,
			Type:      core.StepTypeModelResponse,
			Tokens: core.TokenBreakdown{
				TotalTokens:  10000,
				CachedTokens: 0,
				NewTokens:    10000,
				CacheHitRate: 0.0,
			},
		},
		{
			StepIndex: 3,
			Scope:     core.ScopeCloudInference,
			Type:      core.StepTypeModelResponse,
			Tokens: core.TokenBreakdown{
				TotalTokens:  25000,
				CachedTokens: 20000,
				NewTokens:    5000,
				CacheHitRate: 80.0,
			},
		},
	}

	series := core.ExtractTurnTrendSeries(history, 50)
	if len(series.Points) != 2 {
		t.Fatalf("Points len = %d, want 2", len(series.Points))
	}
	if series.MaxContext != 25000 {
		t.Errorf("MaxContext = %d, want 25000", series.MaxContext)
	}
	if series.PeakNew != 10000 {
		t.Errorf("PeakNew = %d, want 10000", series.PeakNew)
	}
	if series.LatestCached != 20000 {
		t.Errorf("LatestCached = %d, want 20000", series.LatestCached)
	}
	if series.AvgHitRate != 40.0 {
		t.Errorf("AvgHitRate = %v, want 40.0", series.AvgHitRate)
	}
}
