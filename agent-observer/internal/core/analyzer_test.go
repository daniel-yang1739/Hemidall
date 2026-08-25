package core

import (
	"testing"
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

	// 模擬 Step 0: User Input
	event0 := UnifiedAgentEvent{
		SessionID:  "test-session-1",
		StepIndex:  0,
		Type:       StepTypeUserInput,
		RawContent: "Please write a web server for me.",
	}
	analyzer.AnalyzeStep(&event0)

	if event0.Tokens.SystemTokens != 4618 {
		t.Errorf("expected 4618 system tokens baseline, got %d", event0.Tokens.SystemTokens)
	}
	if event0.Tokens.ToolsDefTokens != 3200 {
		t.Errorf("expected 3200 tools def tokens baseline, got %d", event0.Tokens.ToolsDefTokens)
	}
	if event0.Tokens.ActiveTurnTokens <= 0 {
		t.Errorf("expected positive active turn tokens, got %d", event0.Tokens.ActiveTurnTokens)
	}
	if event0.CacheStatus != "WRITE" {
		t.Errorf("expected initial CacheStatus WRITE, got %s", event0.CacheStatus)
	}

	// 模擬 Step 1: Tool Execution Result
	event1 := UnifiedAgentEvent{
		SessionID:  "test-session-1",
		StepIndex:  1,
		Type:       StepTypeRunCommand,
		RawContent: "Compilation succeeded with 0 errors.",
	}
	analyzer.AnalyzeStep(&event1)

	if event1.Tokens.ToolResultTokens <= 0 {
		t.Errorf("expected positive tool result tokens, got %d", event1.Tokens.ToolResultTokens)
	}
	if event1.Tokens.CachedTokens <= 0 {
		t.Errorf("expected positive cached tokens, got %d", event1.Tokens.CachedTokens)
	}
	if event1.CacheStatus != "HIT" {
		t.Errorf("expected CacheStatus HIT, got %s", event1.CacheStatus)
	}
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
