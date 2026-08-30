package core_test

import (
	"testing"

	"heimdall/internal/core"
)

func TestGetModelDiscount_GeminiFlash(t *testing.T) {
	disc, fact, label := core.GetModelDiscount("gemini-3.7-flash")
	if disc != 0.90 || fact != 0.10 {
		t.Errorf("GetModelDiscount(gemini-3.7-flash) = (%v, %v), want (0.90, 0.10)", disc, fact)
	}
	if label == "" {
		t.Errorf("GetModelDiscount returned empty label")
	}
}

func TestGetModelDiscount_GeminiPro(t *testing.T) {
	disc, fact, label := core.GetModelDiscount("gemini-2.5-pro")
	if disc != 0.90 || fact != 0.10 {
		t.Errorf("GetModelDiscount(gemini-2.5-pro) = (%v, %v), want (0.90, 0.10)", disc, fact)
	}
	if label == "" {
		t.Errorf("GetModelDiscount returned empty label")
	}
}

func TestGetModelDiscount_ClaudeSonnet(t *testing.T) {
	disc, fact, label := core.GetModelDiscount("claude-3-7-sonnet-20250219")
	if disc != 0.90 || fact != 0.10 {
		t.Errorf("GetModelDiscount(claude-3-7-sonnet) = (%v, %v), want (0.90, 0.10)", disc, fact)
	}
	if label == "" {
		t.Errorf("GetModelDiscount returned empty label")
	}
}

func TestGetModelDiscount_DeepSeekR1(t *testing.T) {
	disc, fact, label := core.GetModelDiscount("deepseek-r1")
	if disc != 0.75 || fact != 0.25 {
		t.Errorf("GetModelDiscount(deepseek-r1) = (%v, %v), want (0.75, 0.25)", disc, fact)
	}
	if label == "" {
		t.Errorf("GetModelDiscount returned empty label")
	}
}

func TestGetModelDiscount_UnknownFallback(t *testing.T) {
	disc, fact, label := core.GetModelDiscount("unknown-model")
	if disc != 0.75 || fact != 0.25 {
		t.Errorf("GetModelDiscount(unknown-model) = (%v, %v), want (0.75, 0.25)", disc, fact)
	}
	if label == "" {
		t.Errorf("GetModelDiscount returned empty label")
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

	// Model 1 (Gemini 3.7 Flash 90% discount): Effective = 20k + (80k * 0.10) = 28k. Saved = 72k
	// Model 2 (Claude 3.7 Sonnet 90% discount): Effective = 10k + (40k * 0.10) = 14k. Saved = 36k
	// Weighted total effective = 28k + 14k = 42k
	// Total saved = 150k - 42k = 108k (72.0%)

	if metrics.TotalStats.EffectiveTokens != 42000 {
		t.Errorf("EffectiveTokens = %d, want 42000", metrics.TotalStats.EffectiveTokens)
	}
	if metrics.TotalStats.TokensSaved != 108000 {
		t.Errorf("TokensSaved = %d, want 108000", metrics.TotalStats.TokensSaved)
	}
	if len(metrics.ModelStats) != 2 {
		t.Fatalf("ModelStats count = %d, want 2", len(metrics.ModelStats))
	}
}

func TestStats_Neg_ZeroCloudTurnsDivisionByZeroGuard(t *testing.T) {
	history := []core.UnifiedAgentEvent{
		{
			StepIndex: 1,
			Scope:     core.ScopeLocalExecution,
			Type:      core.StepTypeRunCommand,
		},
	}

	metrics := core.ComputeSessionAggregateMetrics(history)
	if metrics.TotalStats.TotalProcessed != 0 {
		t.Errorf("Expected 0 total processed, got %d", metrics.TotalStats.TotalProcessed)
	}
	if metrics.TotalStats.CacheHitRate != 0.0 {
		t.Errorf("Expected 0.0 hit rate, got %f", metrics.TotalStats.CacheHitRate)
	}
}
