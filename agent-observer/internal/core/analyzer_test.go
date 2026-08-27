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
