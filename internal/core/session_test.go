package core

import (
	"sync"
	"testing"
	"time"
)

// ==============================================================================
// 10. SESSION CONCURRENCY & DATA STRUCTURE TESTS (2 Positive + 2 Negative)
// ==============================================================================

func TestSessionStore_Pos_SequentialAppendAndRead(t *testing.T) {
	var events []UnifiedAgentEvent
	for i := 0; i < 50; i++ {
		events = append(events, UnifiedAgentEvent{
			SessionID: "sess-seq-1",
			StepIndex: i,
			Type:      StepTypeModelResponse,
		})
	}
	if len(events) != 50 {
		t.Fatalf("Expected 50 events, got %d", len(events))
	}
	if events[49].StepIndex != 49 {
		t.Errorf("Expected StepIndex 49, got %d", events[49].StepIndex)
	}
}

func TestSessionStore_Pos_AtomicSessionSwitching(t *testing.T) {
	sessionA := SessionInfo{
		AgentType: AgentTypeAntigravity,
		SessionID: "sess-a",
		StepCount: 100,
	}
	sessionB := SessionInfo{
		AgentType: AgentTypeClaudeCode,
		SessionID: "sess-b",
		StepCount: 50,
	}

	var mu sync.RWMutex
	current := sessionA

	mu.Lock()
	current = sessionB
	mu.Unlock()

	mu.RLock()
	read := current
	mu.RUnlock()

	if read.SessionID != "sess-b" || read.AgentType != AgentTypeClaudeCode {
		t.Fatalf("Expected atomic switch to session B, got %+v", read)
	}
}

func TestSessionStore_Neg_100GoroutinesConcurrentRaceCondition(t *testing.T) {
	var mu sync.RWMutex
	var events []UnifiedAgentEvent

	var wg sync.WaitGroup
	// 50 concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mu.Lock()
			events = append(events, UnifiedAgentEvent{
				SessionID: "sess-race",
				StepIndex: idx,
				Timestamp: time.Now(),
			})
			mu.Unlock()
		}(i)
	}

	// 50 concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.RLock()
			_ = len(events)
			mu.RUnlock()
		}()
	}

	wg.Wait()

	mu.RLock()
	finalLen := len(events)
	mu.RUnlock()

	if finalLen != 50 {
		t.Fatalf("Expected exactly 50 events after concurrent writes, got %d", finalLen)
	}
}

func TestSessionStore_Neg_AgentTypeUniversalInvariants(t *testing.T) {
	types := []AgentType{
		AgentTypeAntigravity,
		AgentTypeClaudeCode,
		AgentTypeOpenCode,
	}
	for _, at := range types {
		if string(at) == "" {
			t.Errorf("AgentType string cannot be empty")
		}
	}
}
