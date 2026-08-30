package core

import (
	"testing"
)

func TestCountTokens_EmptyString(t *testing.T) {
	result := CountTokens("")
	if result != 0 {
		t.Errorf("CountTokens(\"\") = %v, want 0", result)
	}
}

func TestCountTokens_SimpleText(t *testing.T) {
	result := CountTokens("hello")
	if result <= 0 {
		t.Errorf("CountTokens(\"hello\") = %v, want > 0", result)
	}
}

func TestCountTokens_Sentence(t *testing.T) {
	result := CountTokens("hello world")
	if result <= 0 {
		t.Errorf("CountTokens(\"hello world\") = %v, want > 0", result)
	}
}

func TestCountTokens_ComplexText(t *testing.T) {
	result := CountTokens("The quick brown fox jumps over the lazy dog.")
	if result <= 0 {
		t.Errorf("CountTokens(\"The quick brown fox...\") = %v, want > 0", result)
	}
}
