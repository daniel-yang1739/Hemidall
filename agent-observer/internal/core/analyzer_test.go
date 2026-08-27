package core

import (
	"testing"
	"time"
)

func TestTokenizerCount(t *testing.T) {
	text := "Hello, this is a test string for TikToken."
	tokens := CountTokens(text)
	if tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
}

func TestPayloadAnalyzerFiveDimensions(t *testing.T) {
	analyzer := NewPayloadAnalyzer()

	// Simulate Step 0: User Input
	event0 := UnifiedAgentEvent{
		SessionID:  "test-session-1",
		StepIndex:  0,
		Type:       StepTypeModelResponse,
		RawContent: "Please write a web server for me.",
	}
	analyzer.AnalyzeStep(&event0)

	if event0.Tokens.SystemTokens <= 0 {
		t.Errorf("expected positive system tokens baseline, got %d", event0.Tokens.SystemTokens)
	}
	if event0.Tokens.ToolsDefTokens <= 0 {
		t.Errorf("expected positive tools def tokens baseline, got %d", event0.Tokens.ToolsDefTokens)
	}
	if event0.Tokens.ActiveTurnTokens <= 0 {
		t.Errorf("expected positive active turn tokens, got %d", event0.Tokens.ActiveTurnTokens)
	}
	if event0.CacheStatus != "WRITE" {
		t.Errorf("expected initial CacheStatus WRITE, got %s", event0.CacheStatus)
	}

	// Simulate Step 1: Model Response with Tool Call (Cloud step)
	event1 := UnifiedAgentEvent{
		SessionID:  "test-session-1",
		StepIndex:  1,
		Type:       StepTypeModelResponse,
		RawContent: "Compilation succeeded with 0 errors.",
	}
	analyzer.AnalyzeStep(&event1)

	if event1.Tokens.ActiveTurnTokens <= 0 {
		t.Errorf("expected positive active turn tokens, got %d", event1.Tokens.ActiveTurnTokens)
	}
	if event1.Tokens.CachedTokens <= 0 {
		t.Errorf("expected positive cached tokens, got %d", event1.Tokens.CachedTokens)
	}
	if event1.CacheStatus != "HIT" {
		t.Errorf("expected CacheStatus HIT, got %s", event1.CacheStatus)
	}
}

func TestReverseSlidingWindow(t *testing.T) {
	analyzer := NewPayloadAnalyzer()

	// Feed 100 historical steps (generating ~10,000 tokens of past history)
	for i := 0; i < 100; i++ {
		stepType := StepTypeModelResponse
		if i%2 == 0 {
			stepType = StepTypeRunCommand
		}
		e := UnifiedAgentEvent{
			SessionID:  "test-sliding-session",
			StepIndex:  i,
			Type:       stepType,
			RawContent: "Some medium length content for step execution and testing history accumulation.",
		}
		analyzer.AnalyzeStep(&e)
	}

	// Now official telemetry arrives for Step 100 with an official active window of 4,000 tokens
	// The total historical log is > 10,000 tokens, but official total is only 4,000!
	eventOfficial := UnifiedAgentEvent{
		SessionID: "test-sliding-session",
		StepIndex: 100,
		Timestamp: time.Now(),
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    4000,
			CachedTokens:   3500,
			OfficialModel:  "gemini-3.7-flash-high",
		},
	}
	analyzer.AnalyzeStep(&eventOfficial)

	// Verify math consistency
	d := eventOfficial.Tokens
	sum := d.SystemTokens + d.ToolsDefTokens + d.ToolResultTokens + d.HistoryTokens + d.ActiveTurnTokens
	if sum != 4000 {
		t.Fatalf("Reverse sliding window sum mismatch! got=%d, expected=4000", sum)
	}
	if d.RawLocalAccumulated <= 4000 {
		t.Fatalf("Expected raw local accumulated to be > 4000, got %d", d.RawLocalAccumulated)
	}
	t.Logf("✅ Reverse Sliding Window successfully extracted %d tokens from raw %d accumulated tokens!",
		sum, d.RawLocalAccumulated)
}

func TestFormatTable(t *testing.T) {
	event := UnifiedAgentEvent{
		SessionID: "test-session-1",
		StepIndex: 5,
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			SystemTokens:     4500,
			ToolsDefTokens:   3200,
			ToolResultTokens: 5000,
			HistoryTokens:    2000,
			ActiveTurnTokens: 300,
			ThinkingTokens:   500,
			TotalTokens:      15500,
			CachedTokens:     14700,
			NewTokens:        800,
			CacheHitRate:     94.8,
		},
		CacheStatus: "HIT",
	}

	tableStr := FormatTokenBreakdownTable(event)
	if tableStr == "" {
		t.Fatalf("expected non-empty table output")
	}
}

func TestClassifyCacheStatus_SingleSourceOfTruth(t *testing.T) {
	cases := []struct {
		hitRate      float64
		cachedTokens int
		totalTokens  int
		isExpired    bool
		expected     string
	}{
		{hitRate: 0.0, cachedTokens: 0, totalTokens: 0, isExpired: false, expected: ""},
		{hitRate: 0.0, cachedTokens: 0, totalTokens: 1000, isExpired: false, expected: "MISS"},
		{hitRate: 50.0, cachedTokens: 500, totalTokens: 1000, isExpired: true, expected: "EXPIRED"},
		{hitRate: 74.0, cachedTokens: 740, totalTokens: 1000, isExpired: false, expected: "PARTIAL"},
		{hitRate: 79.9, cachedTokens: 799, totalTokens: 1000, isExpired: false, expected: "PARTIAL"},
		{hitRate: 80.0, cachedTokens: 800, totalTokens: 1000, isExpired: false, expected: "HIT"},
		{hitRate: 98.5, cachedTokens: 985, totalTokens: 1000, isExpired: false, expected: "HIT"},
	}

	for _, c := range cases {
		actual := ClassifyCacheStatus(c.hitRate, c.cachedTokens, c.totalTokens, c.isExpired)
		if actual != c.expected {
			t.Errorf("For rate=%.1f cached=%d total=%d expired=%v: expected '%s', got '%s'",
				c.hitRate, c.cachedTokens, c.totalTokens, c.isExpired, c.expected, actual)
		}
	}
}

func TestCheckpointScopeAndContextReanchoring(t *testing.T) {
	analyzer := NewPayloadAnalyzer()

	// Step 0: User Input (Local)
	step0 := UnifiedAgentEvent{
		SessionID:  "sess-compaction-test",
		StepIndex:  0,
		Type:       StepTypeUserInput,
		RawContent: "Hello assistant!",
	}
	analyzer.AnalyzeStep(&step0)
	if step0.Scope != ScopeUserInteraction {
		t.Errorf("Expected ScopeUserInteraction, got %v", step0.Scope)
	}
	if step0.CacheStatus != "" {
		t.Errorf("User step should have empty CacheStatus, got '%s'", step0.CacheStatus)
	}

	// Step 1: Model Response (Cloud - Cold write)
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-compaction-test",
		StepIndex:  1,
		Type:       StepTypeModelResponse,
		RawContent: "Hello! How can I help you today?",
	}
	analyzer.AnalyzeStep(&step1)
	if step1.Scope != ScopeCloudInference {
		t.Errorf("Expected ScopeCloudInference, got %v", step1.Scope)
	}
	if step1.CacheStatus != "WRITE" {
		t.Errorf("Initial turn expected WRITE, got '%s'", step1.CacheStatus)
	}

	// Step 2: CHECKPOINT (Compaction Injection)
	step2 := UnifiedAgentEvent{
		SessionID:  "sess-compaction-test",
		StepIndex:  2,
		Type:       StepTypeCheckpoint,
		RawContent: "{{ CHECKPOINT 1 }} # Truncated history summary with 500 tokens of decisions.",
	}
	analyzer.AnalyzeStep(&step2)
	if step2.Scope != ScopeSystemCompaction {
		t.Errorf("Expected ScopeSystemCompaction, got %v", step2.Scope)
	}
	if step2.CacheStatus != "" {
		t.Errorf("Checkpoint step must have empty CacheStatus, got '%s'", step2.CacheStatus)
	}

	// Step 3: Next Model Response after compaction
	step3 := UnifiedAgentEvent{
		SessionID:  "sess-compaction-test",
		StepIndex:  3,
		Type:       StepTypeModelResponse,
		RawContent: "Continuing after checkpoint...",
	}
	analyzer.AnalyzeStep(&step3)
	if step3.Scope != ScopeCloudInference {
		t.Errorf("Expected ScopeCloudInference, got %v", step3.Scope)
	}
	if step3.CacheStatus != "HIT" {
		t.Errorf("Next turn after checkpoint should hit re-anchored cache, got '%s'", step3.CacheStatus)
	}
}

