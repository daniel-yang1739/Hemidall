package core

import (
	"testing"
)

func TestGetAgentRole(t *testing.T) {
	tests := []struct {
		name     string
		event    UnifiedAgentEvent
		expected string
	}{
		{
			name: "Explicit AgentRole",
			event: UnifiedAgentEvent{
				AgentRole: "CUSTOM_ROLE",
			},
			expected: "CUSTOM_ROLE",
		},
		{
			name: "Subagent flag true",
			event: UnifiedAgentEvent{
				IsSubagent: true,
			},
			expected: "SUBAGENT",
		},
		{
			name: "Scope Subagent",
			event: UnifiedAgentEvent{
				Scope: ScopeSubagent,
			},
			expected: "SUBAGENT",
		},
		{
			name: "System Source",
			event: UnifiedAgentEvent{
				Source: "SYSTEM",
			},
			expected: "INTERNAL",
		},
		{
			name: "Scope SystemBootstrap",
			event: UnifiedAgentEvent{
				Scope: ScopeSystemBootstrap,
			},
			expected: "INTERNAL",
		},
		{
			name: "Scope SystemCompaction",
			event: UnifiedAgentEvent{
				Scope: ScopeSystemCompaction,
			},
			expected: "INTERNAL",
		},
		{
			name:     "Default Main",
			event:    UnifiedAgentEvent{},
			expected: "MAIN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.GetAgentRole(); got != tt.expected {
				t.Errorf("UnifiedAgentEvent.GetAgentRole() = %v, want %v", got, tt.expected)
			}
		})
	}
}
