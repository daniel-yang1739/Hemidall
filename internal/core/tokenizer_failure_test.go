package core

import (
	"errors"
	"testing"
)

func TestTokenizerFailureRemainsStableAcrossCalls(t *testing.T) {
	_, _ = GetTokenizer()
	savedInstance, savedError := bpeInstance, bpeError
	t.Cleanup(func() { bpeInstance, bpeError = savedInstance, savedError })
	bpeInstance, bpeError = nil, errors.New("encoder unavailable")
	cases := []struct {
		name, text string
		expected   int
	}{{"first", "12345", 2}, {"repeated", "12345678", 2}, {"empty", "", 0}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountTokens(tc.text); got != tc.expected {
				t.Fatalf("fallback tokens: got %d, want %d", got, tc.expected)
			}
			if _, err := GetTokenizer(); err == nil {
				t.Fatal("initialization error was lost")
			}
		})
	}
}
