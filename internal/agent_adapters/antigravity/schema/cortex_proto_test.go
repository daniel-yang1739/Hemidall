package schema

import (
	"testing"
	"time"
)

func TestMessageRole_StringAndDescription(t *testing.T) {
	cases := []struct {
		role        MessageRole
		expectedStr string
		expectedDesc string
	}{
		{RoleUser, "USER", "User Request / Context Summary"},
		{RoleAssistant, "ASSISTANT", "Model Turn / Tool Call"},
		{RoleToolResult, "TOOL_RESULT", "Tool Execution Output"},
		{RoleSystem, "SYSTEM", "System Prompt"},
		{MessageRole(99), "UNKNOWN_99", "Unspecified Role"},
	}

	for _, c := range cases {
		t.Run(c.expectedStr, func(t *testing.T) {
			if s := c.role.String(); s != c.expectedStr {
				t.Errorf("expected string %q, got %q", c.expectedStr, s)
			}
			if d := c.role.Description(); d != c.expectedDesc {
				t.Errorf("expected desc %q, got %q", c.expectedDesc, d)
			}
		})
	}
}

func TestDecodeDuration(t *testing.T) {
	// Field 1: sec=4 (varint), Field 2: nanos=500000000 (varint)
	// tag 1: (1<<3)|0 = 0x08, val 4 = 0x04
	// tag 2: (2<<3)|0 = 0x10, val 500000000 = 0x80, 0x94, 0xeb, 0xee, 0x01
	data := []byte{0x08, 0x04, 0x10, 0x80, 0xca, 0xb5, 0xee, 0x01}
	dur := decodeDuration(data)
	expected := 4*time.Second + 500*time.Millisecond
	if dur != expected {
		t.Fatalf("expected duration %v, got %v", expected, dur)
	}
}
