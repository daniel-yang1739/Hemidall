package core

import (
	"strings"
	"testing"
	"time"
)

// ==============================================================================
// 1. TTL & CACHE STATE TRANSITIONS (4 Positive + 5 Negative)
// ==============================================================================

func TestTTL_Pos_InitialBootstrapWrite(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	now := time.Now()

	event0 := UnifiedAgentEvent{
		SessionID:  "sess-ttl-pos-1",
		StepIndex:  0,
		Timestamp:  now,
		Type:       StepTypeModelResponse,
		RawContent: "I am ready to help.",
	}
	analyzer.AnalyzeStep(&event0)

	if event0.CacheStatus != "WRITE" {
		t.Fatalf("Expected initial cloud turn CacheStatus 'WRITE', got '%s'", event0.CacheStatus)
	}
	if event0.Tokens.CachedTokens != 0 {
		t.Errorf("Expected 0 cached tokens on initial write, got %d", event0.Tokens.CachedTokens)
	}
	if event0.Tokens.NewTokens != event0.Tokens.TotalTokens {
		t.Errorf("Expected NewTokens (%d) == TotalTokens (%d)", event0.Tokens.NewTokens, event0.Tokens.TotalTokens)
	}
}

func TestTTL_Pos_NormalCacheHitWithin300s(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	t1 := t0.Add(120 * time.Second) // 2 minutes later (within 300s TTL)

	event0 := UnifiedAgentEvent{
		SessionID:  "sess-ttl-pos-2",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "First turn.",
	}
	analyzer.AnalyzeStep(&event0)

	event1 := UnifiedAgentEvent{
		SessionID:  "sess-ttl-pos-2",
		StepIndex:  1,
		Timestamp:  t1,
		Type:       StepTypeModelResponse,
		RawContent: "Second turn continuing conversation.",
	}
	analyzer.AnalyzeStep(&event1)

	if event1.CacheStatus != "HIT" {
		t.Fatalf("Expected CacheStatus 'HIT' within 120s, got '%s'", event1.CacheStatus)
	}
	if event1.Tokens.CacheHitRate < 80.0 {
		t.Errorf("Expected CacheHitRate >= 80.0%%, got %.2f%%", event1.Tokens.CacheHitRate)
	}
}

func TestTTL_Pos_PartialHitBelow80(t *testing.T) {
	status := ClassifyCacheStatus(45.0, 450, 1000, false)
	if status != "PARTIAL" {
		t.Fatalf("Expected 'PARTIAL' for 45%% hit rate, got '%s'", status)
	}
}

func TestTTL_Pos_CheckpointReanchoringHit(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Now()

	// Step 0: User Input (Local)
	step0 := UnifiedAgentEvent{
		SessionID:  "sess-reanchor",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeUserInput,
		RawContent: "Initial instruction.",
	}
	analyzer.AnalyzeStep(&step0)

	// Step 1: Initial Model Response (Cloud)
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-reanchor",
		StepIndex:  1,
		Timestamp:  t0.Add(5 * time.Second),
		Type:       StepTypeModelResponse,
		RawContent: "Starting task...",
	}
	analyzer.AnalyzeStep(&step1)

	// Step 2: Compaction Checkpoint (ScopeSystemCompaction)
	step2 := UnifiedAgentEvent{
		SessionID:  "sess-reanchor",
		StepIndex:  2,
		Timestamp:  t0.Add(10 * time.Second),
		Type:       StepTypeCheckpoint,
		RawContent: "<CONTEXT_SUMMARY>Compacted memory baseline 12.5k tokens</CONTEXT_SUMMARY>",
	}
	analyzer.AnalyzeStep(&step2)

	if step2.Scope != ScopeSystemCompaction {
		t.Errorf("Expected ScopeSystemCompaction, got %v", step2.Scope)
	}
	if step2.CacheStatus != "" {
		t.Errorf("Checkpoint step must have empty CacheStatus, got '%s'", step2.CacheStatus)
	}

	// Step 3: Cloud inference after compaction within 300s
	step3 := UnifiedAgentEvent{
		SessionID:  "sess-reanchor",
		StepIndex:  3,
		Timestamp:  t0.Add(30 * time.Second),
		Type:       StepTypeModelResponse,
		RawContent: "Continuing with reanchored context.",
	}
	analyzer.AnalyzeStep(&step3)

	if step3.CacheStatus != "HIT" {
		t.Errorf("Expected CacheStatus 'HIT' after compaction re-anchoring, got '%s'", step3.CacheStatus)
	}
}

func TestTTL_Neg_ExpiredAfter300s(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	t1 := t0.Add(301 * time.Second) // 301 seconds later (> 300s TTL)

	event0 := UnifiedAgentEvent{
		SessionID:  "sess-ttl-neg-1",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "First turn.",
	}
	analyzer.AnalyzeStep(&event0)

	event1 := UnifiedAgentEvent{
		SessionID:  "sess-ttl-neg-1",
		StepIndex:  1,
		Timestamp:  t1,
		Type:       StepTypeModelResponse,
		RawContent: "Turn after long idle.",
	}
	analyzer.AnalyzeStep(&event1)

	if event1.CacheStatus != "EXPIRED" {
		t.Fatalf("Expected CacheStatus 'EXPIRED' after 301s, got '%s'", event1.CacheStatus)
	}
	if event1.Tokens.CachedTokens != 0 {
		t.Errorf("Expected 0 cached tokens on EXPIRED turn, got %d", event1.Tokens.CachedTokens)
	}
}

func TestTTL_Neg_HumanTypingClockMaskingImmunity(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	tUser := t0.Add(10 * time.Minute) // Human spends 10 minutes typing
	tCloud := tUser.Add(2 * time.Second)

	// Step 0: Cloud turn at t0
	step0 := UnifiedAgentEvent{
		SessionID:  "sess-masking-1",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "What is your next request?",
	}
	analyzer.AnalyzeStep(&step0)

	// Step 1: User types prompt 10 minutes later (Local offline)
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-masking-1",
		StepIndex:  1,
		Timestamp:  tUser,
		Type:       StepTypeUserInput,
		RawContent: "Please refactor the database module.",
	}
	analyzer.AnalyzeStep(&step1)

	if step1.Scope != ScopeUserInteraction {
		t.Errorf("Expected ScopeUserInteraction, got %v", step1.Scope)
	}

	// Step 2: Next Cloud turn right after user submits
	step2 := UnifiedAgentEvent{
		SessionID:  "sess-masking-1",
		StepIndex:  2,
		Timestamp:  tCloud,
		Type:       StepTypeModelResponse,
		RawContent: "Starting refactoring...",
	}
	analyzer.AnalyzeStep(&step2)

	// MUST be EXPIRED because 10 minutes elapsed since last cloud turn!
	if step2.CacheStatus != "EXPIRED" {
		t.Fatalf("CRITICAL BUG: Human typing masked cloud TTL! Expected 'EXPIRED', got '%s'", step2.CacheStatus)
	}
}

func TestTTL_Neg_LocalToolClockMaskingImmunity(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	tTool := t0.Add(8 * time.Minute) // Local compile takes 8 minutes
	tCloud := tTool.Add(3 * time.Second)

	// Step 0: Cloud turn at t0
	step0 := UnifiedAgentEvent{
		SessionID:  "sess-masking-2",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "Running make build.",
	}
	analyzer.AnalyzeStep(&step0)

	// Step 1: Local tool execution takes 8 minutes
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-masking-2",
		StepIndex:  1,
		Timestamp:  tTool,
		Type:       StepTypeRunCommand,
		RawContent: "Compilation finished in 8m02s with 0 errors.",
	}
	analyzer.AnalyzeStep(&step1)

	if step1.Scope != ScopeLocalExecution {
		t.Errorf("Expected ScopeLocalExecution, got %v", step1.Scope)
	}

	// Step 2: Next Cloud turn
	step2 := UnifiedAgentEvent{
		SessionID:  "sess-masking-2",
		StepIndex:  2,
		Timestamp:  tCloud,
		Type:       StepTypeModelResponse,
		RawContent: "Build verified.",
	}
	analyzer.AnalyzeStep(&step2)

	// MUST be EXPIRED because 8 minutes elapsed since last cloud turn!
	if step2.CacheStatus != "EXPIRED" {
		t.Fatalf("CRITICAL BUG: Local tool masked cloud TTL! Expected 'EXPIRED', got '%s'", step2.CacheStatus)
	}
}

func TestTTL_Neg_ZeroTimestampAndOutdatedClockFallback(t *testing.T) {
	analyzer := NewPayloadAnalyzer()

	// Zero timestamp event
	eventZero := UnifiedAgentEvent{
		SessionID:  "sess-zero-ts",
		StepIndex:  0,
		Type:       StepTypeModelResponse,
		RawContent: "No timestamp event.",
	}
	analyzer.AnalyzeStep(&eventZero)

	if eventZero.CacheStatus != "WRITE" {
		t.Errorf("Expected WRITE on zero timestamp, got '%s'", eventZero.CacheStatus)
	}

	// Negative clock skew event (clock moved backwards)
	eventPast := UnifiedAgentEvent{
		SessionID:  "sess-zero-ts",
		StepIndex:  1,
		Timestamp:  time.Now().Add(-1 * time.Hour),
		Type:       StepTypeModelResponse,
		RawContent: "Past timestamp event.",
	}
	// Should not panic or crash
	analyzer.AnalyzeStep(&eventPast)
	if eventPast.Tokens.TotalTokens <= 0 {
		t.Errorf("Expected positive tokens on past event, got %d", eventPast.Tokens.TotalTokens)
	}
}

func TestTTL_Neg_ModelSwitchTriggersWrite(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	now := time.Now()

	// Step 0 on Gemini Flash
	event0 := UnifiedAgentEvent{
		SessionID: "sess-model-switch",
		StepIndex: 0,
		Timestamp: now,
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    5000,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&event0)

	// Step 1 switched to Claude Sonnet (within 10 seconds)
	event1 := UnifiedAgentEvent{
		SessionID: "sess-model-switch",
		StepIndex: 1,
		Timestamp: now.Add(10 * time.Second),
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    6000,
			OfficialModel:  "claude-3-7-sonnet",
		},
	}
	analyzer.AnalyzeStep(&event1)

	// Model switch MUST trigger WRITE because GPU KV Cache is not shared across models!
	if event1.CacheStatus != "WRITE" {
		t.Fatalf("Expected 'WRITE' on model switch, got '%s'", event1.CacheStatus)
	}
}

// ==============================================================================
// 2. STEP TYPE & SCOPE CLASSIFICATIONS (7 Positive + 7 Negative)
// ==============================================================================

func TestStep_Pos_UserInputScopeUser(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-step-pos",
		StepIndex:  0,
		Type:       StepTypeUserInput,
		RawContent: "Hello world prompt.",
	}
	analyzer.AnalyzeStep(&event)

	if event.Scope != ScopeUserInteraction {
		t.Errorf("Expected ScopeUserInteraction, got %v", event.Scope)
	}
	if event.IsCloudStep() {
		t.Errorf("UserInput must NOT be a cloud step")
	}
}

func TestStep_Pos_ModelReasoningScopeCloud(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-step-pos",
		StepIndex:  1,
		Type:       StepTypeModelResponse,
		RawContent: "I have thought about the architecture.",
		Thinking:   "First check the dependencies...",
	}
	analyzer.AnalyzeStep(&event)

	if event.Scope != ScopeCloudInference {
		t.Errorf("Expected ScopeCloudInference, got %v", event.Scope)
	}
	if !event.IsCloudStep() {
		t.Errorf("ModelResponse must be a cloud step")
	}
}

func TestStep_Pos_ToolCallScopeCloud(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID: "sess-step-pos",
		StepIndex: 2,
		Type:      StepTypeToolCall,
		ToolCalls: []ToolCallInfo{
			{ToolName: "view_file", RawArgs: `{"AbsolutePath":"/app/main.go"}`},
		},
	}
	analyzer.AnalyzeStep(&event)

	if event.Scope != ScopeCloudInference {
		t.Errorf("Expected ScopeCloudInference, got %v", event.Scope)
	}
	if !event.IsCloudStep() {
		t.Errorf("ToolCall must be a cloud step")
	}
}

func TestStep_Pos_RunCommandScopeLocal(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-step-pos",
		StepIndex:  3,
		Type:       StepTypeRunCommand,
		RawContent: "Files listed: main.go, go.mod",
	}
	analyzer.AnalyzeStep(&event)

	if event.Scope != ScopeLocalExecution {
		t.Errorf("Expected ScopeLocalExecution, got %v", event.Scope)
	}
	if event.IsCloudStep() {
		t.Errorf("RunCommand must NOT be a cloud step")
	}
	if event.CacheStatus != "" {
		t.Errorf("Local step CacheStatus must be empty, got '%s'", event.CacheStatus)
	}
}

func TestStep_Pos_ViewFileScopeLocal(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-step-pos",
		StepIndex:  4,
		Type:       StepTypeViewFile,
		RawContent: "package main\nfunc main() {}",
	}
	analyzer.AnalyzeStep(&event)

	if event.Scope != ScopeLocalExecution {
		t.Errorf("Expected ScopeLocalExecution, got %v", event.Scope)
	}
}

func TestStep_Pos_CodeActionScopeLocal(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-step-pos",
		StepIndex:  5,
		Type:       StepTypeCodeAction,
		RawContent: "Diff applied: +5 lines, -2 lines",
	}
	analyzer.AnalyzeStep(&event)

	if event.Scope != ScopeLocalExecution {
		t.Errorf("Expected ScopeLocalExecution, got %v", event.Scope)
	}
}

func TestStep_Pos_AskQuestionScopeLocal(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-step-pos",
		StepIndex:  6,
		Type:       StepTypeAskQuestion,
		RawContent: "User selected Option 1.",
	}
	analyzer.AnalyzeStep(&event)

	if event.Scope != ScopeLocalExecution {
		t.Errorf("Expected ScopeLocalExecution, got %v", event.Scope)
	}
}

func TestStep_Neg_CorruptedJsonGracefulFallback(t *testing.T) {
	event := UnifiedAgentEvent{
		SessionID:  "sess-corrupt",
		StepIndex:  10,
		Type:       StepTypeUnknown,
		RawContent: `{"type": "PLANNER_RESPO... truncated EOF`,
	}
	if event.IsCloudStep() {
		t.Errorf("Corrupted StepTypeUnknown must not be classified as CloudStep")
	}
	if !event.IsLocalStep() {
		t.Errorf("Corrupted StepTypeUnknown should fallback to IsLocalStep")
	}
}

func TestStep_Neg_MissingTypeAndSourceDefaults(t *testing.T) {
	event := UnifiedAgentEvent{
		SessionID: "sess-missing-fields",
		StepIndex: 11,
	}
	role := event.GetAgentRole()
	if role != "MAIN" {
		t.Errorf("Expected default role 'MAIN', got '%s'", role)
	}
}

func TestStep_Neg_SecurityBlockedStatus7(t *testing.T) {
	event := UnifiedAgentEvent{
		SessionID: "sess-security",
		StepIndex: 12,
		Type:      StepTypeError,
		Status:    "BLOCKED",
		Scope:     ScopeLocalExecution,
	}
	if event.IsCloudStep() {
		t.Errorf("Security blocked step must NOT be billed as CloudStep")
	}
}

func TestStep_Neg_MassivePayloadTruncationGuard(t *testing.T) {
	massiveStr := strings.Repeat("A", 20000)
	tokens := CountTokens(massiveStr)
	if tokens <= 0 {
		t.Fatalf("Expected positive token count for massive payload, got %d", tokens)
	}
}

func TestStep_Neg_SubagentContextIsolation(t *testing.T) {
	event := UnifiedAgentEvent{
		SessionID:  "sess-main",
		StepIndex:  20,
		IsSubagent: true,
		AgentRole:  "SUBAGENT",
		Scope:      ScopeSubagent,
	}
	if event.GetAgentRole() != "SUBAGENT" {
		t.Errorf("Expected GetAgentRole() 'SUBAGENT', got '%s'", event.GetAgentRole())
	}
}

func TestStep_Neg_LocalStepZeroCacheStatus(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-local-cache",
		StepIndex:  0,
		Type:       StepTypeListDirectory,
		RawContent: "dir1, dir2, file1.go",
	}
	analyzer.AnalyzeStep(&event)

	if event.CacheStatus != "" {
		t.Errorf("Local list_dir must have empty CacheStatus, got '%s'", event.CacheStatus)
	}
	if event.Tokens.CachedTokens != 0 {
		t.Errorf("Local list_dir must have 0 CachedTokens, got %d", event.Tokens.CachedTokens)
	}
}

func TestStep_Neg_ZeroTokenStepHandling(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	event := UnifiedAgentEvent{
		SessionID:  "sess-zero-token",
		StepIndex:  0,
		Type:       StepTypeModelResponse,
		RawContent: "",
	}
	analyzer.AnalyzeStep(&event)

	if event.Tokens.TotalTokens < 0 {
		t.Errorf("TotalTokens cannot be negative, got %d", event.Tokens.TotalTokens)
	}
}

// ==============================================================================
// 3. 5-DIMENSION TOKEN MATH & CONSERVATION (2 Positive + 2 Negative)
// ==============================================================================

func TestTokenMath_Pos_ReverseSlidingWindowConservation(t *testing.T) {
	analyzer := NewPayloadAnalyzer()

	// Accumulate history
	for i := 0; i < 20; i++ {
		e := UnifiedAgentEvent{
			SessionID:  "sess-math-pos",
			StepIndex:  i,
			Type:       StepTypeModelResponse,
			RawContent: "Historical step context payload block.",
		}
		analyzer.AnalyzeStep(&e)
	}

	eventOfficial := UnifiedAgentEvent{
		SessionID: "sess-math-pos",
		StepIndex: 20,
		Timestamp: time.Now(),
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    6000,
			CachedTokens:   5000,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&eventOfficial)

	tb := eventOfficial.Tokens
	sum := tb.SystemTokens + tb.ToolsDefTokens + tb.ToolResultTokens + tb.HistoryTokens + tb.ActiveTurnTokens
	if sum != 6000 {
		t.Fatalf("Conservation violation! Sum (%d) != TotalTokens (6000)", sum)
	}
}

func TestTokenMath_Pos_OfficialTelemetryOverride(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	eventOfficial := UnifiedAgentEvent{
		SessionID: "sess-telemetry-override",
		StepIndex: 0,
		Timestamp: time.Now(),
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    8000,
			CachedTokens:   7000,
			OfficialModel:  "gemini-3.7-pro",
		},
	}
	analyzer.AnalyzeStep(&eventOfficial)

	if eventOfficial.Tokens.TotalTokens != 8000 {
		t.Errorf("Expected official TotalTokens 8000, got %d", eventOfficial.Tokens.TotalTokens)
	}
	if eventOfficial.Tokens.OfficialModel != "gemini-3.7-pro" {
		t.Errorf("Expected official model 'gemini-3.7-pro', got '%s'", eventOfficial.Tokens.OfficialModel)
	}
}

func TestTokenMath_Neg_TotalTokensSmallerThanSystemFloorGuard(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	// Official total is only 50 tokens (smaller than BaseSystem ~3800)
	eventTiny := UnifiedAgentEvent{
		SessionID: "sess-tiny-total",
		StepIndex: 0,
		Timestamp: time.Now(),
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    50,
			CachedTokens:   0,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&eventTiny)

	tb := eventTiny.Tokens
	if tb.SystemTokens < 0 || tb.ToolsDefTokens < 0 || tb.ActiveTurnTokens < 0 {
		t.Errorf("Token dimensions cannot be negative: %+v", tb)
	}
	sum := tb.SystemTokens + tb.ToolsDefTokens + tb.ToolResultTokens + tb.HistoryTokens + tb.ActiveTurnTokens
	if sum != 50 {
		t.Errorf("Tiny window conservation mismatch! Sum (%d) != TotalTokens (50)", sum)
	}
}

func TestTokenMath_Neg_ZeroTotalTokensAllZero(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	eventZero := UnifiedAgentEvent{
		SessionID: "sess-zero-total",
		StepIndex: 0,
		Timestamp: time.Now(),
		Type:      StepTypeModelResponse,
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    0,
			CachedTokens:   0,
		},
	}
	analyzer.AnalyzeStep(&eventZero)

	tb := eventZero.Tokens
	if tb.TotalTokens != 0 {
		t.Errorf("Expected 0 TotalTokens, got %d", tb.TotalTokens)
	}
}

// ==============================================================================
// 4. TOKENIZER HEURISTICS & CJK (2 Positive + 2 Negative)
// ==============================================================================

func TestTokenizer_Pos_MixedCJKAndEnglishCounting(t *testing.T) {
	mixed := "這是一個包含中文 Traditional Chinese 與 English 的混合測試句子！"
	tokens := CountTokens(mixed)
	if tokens <= 0 {
		t.Fatalf("Expected positive tokens for mixed string, got %d", tokens)
	}
}

func TestTokenizer_Pos_CodeIndentationAndBrackets(t *testing.T) {
	code := `
func CalculateSum(a, b int) int {
	// Add numbers
	return a + b
}
`
	tokens := CountTokens(code)
	if tokens <= 0 {
		t.Fatalf("Expected positive tokens for code snippet, got %d", tokens)
	}
}

func TestTokenizer_Neg_EmptyStringAndWhitespaceOnly(t *testing.T) {
	if count := CountTokens(""); count != 0 {
		t.Errorf("Expected 0 tokens for empty string, got %d", count)
	}
	// Whitespace in BPE generates valid tokens without panic
	count := CountTokens("   \t\n   ")
	if count < 0 {
		t.Errorf("Whitespace token count should not be negative, got %d", count)
	}
}

func TestTokenizer_Neg_ExtremeUnicodeAndAnsiEscapeCodes(t *testing.T) {
	ansi := "\x1b[31mError:\x1b[0m Failed with code 500 🚀🔥"
	tokens := CountTokens(ansi)
	if tokens <= 0 {
		t.Fatalf("Expected positive tokens for ANSI/Emoji string, got %d", tokens)
	}
}



func TestGetModelTTL(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected time.Duration
	}{
		{"Claude model lower", "claude-3-5-sonnet", 5 * time.Minute},
		{"Claude model mixed case", "CLAUDE-3-haiku", 5 * time.Minute},
		{"GPT model", "gpt-4o", 10 * time.Minute},
		{"GPT model mixed case", "GPT-3.5-turbo", 10 * time.Minute},
		{"o1 model", "o1-preview", 10 * time.Minute},
		{"o3 model", "o3-mini", 10 * time.Minute},
		{"Deepseek model", "deepseek-coder", 24 * time.Hour},
		{"Deepseek model upper", "DEEPSEEK-R1", 24 * time.Hour},
		{"Gemini model (default)", "gemini-1.5-pro", 5 * time.Minute},
		{"Unknown model (default)", "unknown-model", 5 * time.Minute},
		{"Empty string (default)", "", 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetModelTTL(tt.model)
			if actual != tt.expected {
				t.Errorf("GetModelTTL(%q) = %v, want %v", tt.model, actual, tt.expected)
			}
		})
	}
}
