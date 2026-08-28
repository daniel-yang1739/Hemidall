package core

import (
	"testing"
	"time"
)

// ==============================================================================
// FULL CONFUSION MATRIX (TP / TN / FP / FN) & CACHE BOUNDARY TESTS
// Strict Zero Control-Flow in Test Bodies (No for-loops, no branch calculations)
// ==============================================================================

// 1. True Positive (TP): Official L2 Prefix Hit Even When Time Expired (e.g. 7 hours later)
func TestCacheMatrix_TP_OfficialHitEvenWhenTimeExpired(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 2, 59, 0, 0, time.UTC)
	tCloud := t0.Add(7*time.Hour + 21*time.Minute) // 7 hours 21 minutes later

	// Step 0: Previous cloud turn at 02:59
	step0 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-tp",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "Previous work completed.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    160000,
			CachedTokens:   140000,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&step0)

	// Step 1: Inbound request 7 hours later with 151,651 official cached tokens (87.9% Hit)
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-tp",
		StepIndex:  1,
		Timestamp:  tCloud,
		Type:       StepTypeModelResponse,
		RawContent: "Resuming session with persistent prefix.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    172431,
			CachedTokens:   151651,
			OfficialModel:  "gemini-3.7-flash-high",
		},
	}
	analyzer.AnalyzeStep(&step1)

	// MUST be classified as HIT because official telemetry confirmed 87.9% physical cache hit!
	if step1.CacheStatus != "HIT" {
		t.Fatalf("TP Violation: Expected CacheStatus 'HIT' from official 87.9%% telemetry, got '%s'", step1.CacheStatus)
	}
	if step1.Tokens.CachedTokens != 151651 {
		t.Fatalf("Expected CachedTokens 151651, got %d", step1.Tokens.CachedTokens)
	}
	if step1.Tokens.NewTokens != 20780 {
		t.Fatalf("Expected NewTokens 20780, got %d", step1.Tokens.NewTokens)
	}
	if step1.Tokens.CacheHitRate < 87.0 {
		t.Fatalf("Expected CacheHitRate >= 87.0%%, got %.2f%%", step1.Tokens.CacheHitRate)
	}
}

// 2. True Negative (TN): Expired Cold Start When Zero Cached Tokens
func TestCacheMatrix_TN_ExpiredColdStartWhenZeroCached(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	tCloud := t0.Add(15 * time.Minute) // 15 minutes later (> 300s TTL)

	step0 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-tn",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "Initial cloud turn.",
	}
	analyzer.AnalyzeStep(&step0)

	step1 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-tn",
		StepIndex:  1,
		Timestamp:  tCloud,
		Type:       StepTypeModelResponse,
		RawContent: "Cold start turn after 15m idle.",
	}
	analyzer.AnalyzeStep(&step1)

	// MUST be EXPIRED because > 300s elapsed AND 0 cached tokens
	if step1.CacheStatus != "EXPIRED" {
		t.Fatalf("TN Violation: Expected CacheStatus 'EXPIRED', got '%s'", step1.CacheStatus)
	}
	if step1.Tokens.CachedTokens != 0 {
		t.Fatalf("Expected 0 cached tokens on true expired cold start, got %d", step1.Tokens.CachedTokens)
	}
}

// 3. False Positive Defense (FP): Partial Hit Below 80% When Time Expired
func TestCacheMatrix_FP_PartialHitBelow80WhenExpired(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	tCloud := t0.Add(20 * time.Minute)

	step0 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-fp",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "Initial turn.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    50000,
			CachedTokens:   0,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&step0)

	// Massive new file read dilutes cache to 40%
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-fp",
		StepIndex:  1,
		Timestamp:  tCloud,
		Type:       StepTypeModelResponse,
		RawContent: "Reading huge 60k file.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    100000,
			CachedTokens:   40000, // 40.0% Hit
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&step1)

	// MUST be PARTIAL (not EXPIRED, not HIT)
	if step1.CacheStatus != "PARTIAL" {
		t.Fatalf("FP Violation: Expected CacheStatus 'PARTIAL' for 40.0%% hit rate, got '%s'", step1.CacheStatus)
	}
	if step1.Tokens.NewTokens != 60000 {
		t.Fatalf("Expected NewTokens 60000, got %d", step1.Tokens.NewTokens)
	}
}

// 4. False Negative Defense (FN): Fresh Turn Within TTL Normal Hit
func TestCacheMatrix_FN_FreshTurnWithinTTLNormalHit(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	t0 := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	tCloud := t0.Add(45 * time.Second) // 45s later (< 300s TTL)

	step0 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-fn",
		StepIndex:  0,
		Timestamp:  t0,
		Type:       StepTypeModelResponse,
		RawContent: "Step 0 turn.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    100000,
			CachedTokens:   0,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&step0)

	step1 := UnifiedAgentEvent{
		SessionID:  "sess-matrix-fn",
		StepIndex:  1,
		Timestamp:  tCloud,
		Type:       StepTypeModelResponse,
		RawContent: "Step 1 fast response.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    105000,
			CachedTokens:   100000, // 95.2% Hit
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&step1)

	if step1.CacheStatus != "HIT" {
		t.Fatalf("FN Violation: Expected CacheStatus 'HIT' for 95.2%% fresh hit, got '%s'", step1.CacheStatus)
	}
}

// 5. Boundary: Exact 80.0% Hit Threshold
func TestCacheMatrix_Boundary_Exact80PercentHitThreshold(t *testing.T) {
	status := ClassifyCacheStatus(80.0, 800, 1000, false)
	if status != "HIT" {
		t.Fatalf("Boundary Failure: Expected 'HIT' at exactly 80.0%% hit rate, got '%s'", status)
	}
}

// 6. Boundary: 79.9% Partial Hit Threshold
func TestCacheMatrix_Boundary_79Point9PercentPartialThreshold(t *testing.T) {
	status := ClassifyCacheStatus(79.9, 799, 1000, false)
	if status != "PARTIAL" {
		t.Fatalf("Boundary Failure: Expected 'PARTIAL' at 79.9%% hit rate, got '%s'", status)
	}
}

// 7. Boundary: Zero Total Tokens
func TestCacheMatrix_Boundary_ZeroTotalTokens(t *testing.T) {
	status := ClassifyCacheStatus(0.0, 0, 0, false)
	if status != "" {
		t.Fatalf("Boundary Failure: Expected empty status for 0 total tokens, got '%s'", status)
	}
}

// 8. Boundary: Zero Cached Tokens Within Normal TTL (Cache Miss)
func TestCacheMatrix_Boundary_ZeroCachedWithinTTLIsMiss(t *testing.T) {
	status := ClassifyCacheStatus(0.0, 0, 1000, false)
	if status != "MISS" {
		t.Fatalf("Boundary Failure: Expected 'MISS' when 0 cached tokens within normal TTL, got '%s'", status)
	}
}

// 9. Model Switch Invalidation: Fallback Mode Gemini to Claude
func TestCacheMatrix_ModelSwitch_Fallback_GeminiToClaudeTriggersWrite(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	now := time.Now()

	// Step 0 on Gemini
	step0 := UnifiedAgentEvent{
		SessionID:  "sess-switch-fallback",
		StepIndex:  0,
		Timestamp:  now,
		Type:       StepTypeModelResponse,
		RawContent: "Hello on Gemini.",
		Tokens: TokenBreakdown{
			IsOfficialData: false,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&step0)

	// Step 1 switched to Claude Sonnet (within 5 seconds)
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-switch-fallback",
		StepIndex:  1,
		Timestamp:  now.Add(5 * time.Second),
		Type:       StepTypeModelResponse,
		RawContent: "Hello on Claude.",
		Tokens: TokenBreakdown{
			IsOfficialData: false,
			OfficialModel:  "claude-3-7-sonnet",
		},
	}
	analyzer.AnalyzeStep(&step1)

	// MUST be WRITE because different model families cannot share GPU KV cache!
	if step1.CacheStatus != "WRITE" {
		t.Fatalf("Model Switch Violation: Expected CacheStatus 'WRITE', got '%s'", step1.CacheStatus)
	}
	if step1.Tokens.CachedTokens != 0 {
		t.Fatalf("Expected 0 cached tokens on cross-family model switch, got %d", step1.Tokens.CachedTokens)
	}
}

// 10. Model Switch Invalidation: Official Mode Incompatible Switch
func TestCacheMatrix_ModelSwitch_Official_IncompatibleModelsTriggersWrite(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	now := time.Now()

	step0 := UnifiedAgentEvent{
		SessionID:  "sess-switch-official",
		StepIndex:  0,
		Timestamp:  now,
		Type:       StepTypeModelResponse,
		RawContent: "Step 0 on Gemini.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    5000,
			CachedTokens:   0,
			OfficialModel:  "gemini-3.7-flash",
		},
	}
	analyzer.AnalyzeStep(&step0)

	step1 := UnifiedAgentEvent{
		SessionID:  "sess-switch-official",
		StepIndex:  1,
		Timestamp:  now.Add(5 * time.Second),
		Type:       StepTypeModelResponse,
		RawContent: "Step 1 on Claude with 0 cached.",
		Tokens: TokenBreakdown{
			IsOfficialData: true,
			TotalTokens:    6000,
			CachedTokens:   0,
			OfficialModel:  "claude-3-7-sonnet",
		},
	}
	analyzer.AnalyzeStep(&step1)

	if step1.CacheStatus != "WRITE" {
		t.Fatalf("Expected CacheStatus 'WRITE' on incompatible model switch, got '%s'", step1.CacheStatus)
	}
	if step1.Tokens.CachedTokens != 0 {
		t.Fatalf("Expected 0 cached tokens, got %d", step1.Tokens.CachedTokens)
	}
}

// 11. Model-Specific TTL Resolution (Claude 5m, OpenAI 10m, DeepSeek 24h, Gemini 5m)
func TestCacheMatrix_GetModelTTL_ProviderMappings(t *testing.T) {
	ttlClaude := GetModelTTL("claude-3-7-sonnet")
	if ttlClaude != 5*time.Minute {
		t.Fatalf("Expected Claude TTL 5m, got %v", ttlClaude)
	}

	ttlOpenAI := GetModelTTL("gpt-4o")
	if ttlOpenAI != 10*time.Minute {
		t.Fatalf("Expected OpenAI TTL 10m, got %v", ttlOpenAI)
	}

	ttlO1 := GetModelTTL("o1-preview")
	if ttlO1 != 10*time.Minute {
		t.Fatalf("Expected o1 TTL 10m, got %v", ttlO1)
	}

	ttlDeepSeek := GetModelTTL("deepseek-reasoner-r1")
	if ttlDeepSeek != 24*time.Hour {
		t.Fatalf("Expected DeepSeek TTL 24h, got %v", ttlDeepSeek)
	}

	ttlGemini := GetModelTTL("gemini-3.7-flash")
	if ttlGemini != 5*time.Minute {
		t.Fatalf("Expected Gemini TTL 5m, got %v", ttlGemini)
	}
}

// 12. OpenAI 10-Minute TTL Window Test (7m hits, 12m expires)
func TestCacheMatrix_OpenAI_10MinTTLWindow(t *testing.T) {
	analyzer := NewPayloadAnalyzer()
	now := time.Now()

	// Step 0 on GPT-4o
	step0 := UnifiedAgentEvent{
		SessionID:  "sess-openai-ttl",
		StepIndex:  0,
		Timestamp:  now,
		Type:       StepTypeModelResponse,
		RawContent: "OpenAI turn 0.",
		Tokens: TokenBreakdown{
			IsOfficialData: false,
			OfficialModel:  "gpt-4o",
		},
	}
	analyzer.AnalyzeStep(&step0)

	// Step 1 at 7 minutes later (within 10m OpenAI TTL)
	step1 := UnifiedAgentEvent{
		SessionID:  "sess-openai-ttl",
		StepIndex:  1,
		Timestamp:  now.Add(7 * time.Minute),
		Type:       StepTypeModelResponse,
		RawContent: "OpenAI turn 1 at 7m.",
		Tokens: TokenBreakdown{
			IsOfficialData: false,
			OfficialModel:  "gpt-4o",
		},
	}
	analyzer.AnalyzeStep(&step1)

	if step1.CacheStatus != "HIT" {
		t.Fatalf("Expected GPT-4o at 7m to be 'HIT' (within 10m window), got '%s'", step1.CacheStatus)
	}

	// Step 2 at 18 minutes later (11 minutes after step 1, > 10m OpenAI TTL)
	step2 := UnifiedAgentEvent{
		SessionID:  "sess-openai-ttl",
		StepIndex:  2,
		Timestamp:  now.Add(18 * time.Minute),
		Type:       StepTypeModelResponse,
		RawContent: "OpenAI turn 2 at 18m.",
		Tokens: TokenBreakdown{
			IsOfficialData: false,
			OfficialModel:  "gpt-4o",
		},
	}
	analyzer.AnalyzeStep(&step2)

	if step2.CacheStatus != "EXPIRED" {
		t.Fatalf("Expected GPT-4o at 11m idle to be 'EXPIRED' (> 10m window), got '%s'", step2.CacheStatus)
	}
}
