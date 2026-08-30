package core

import (
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

var (
	defaultEncoding = "cl100k_base"
	bpeInstance     *tiktoken.Tiktoken
	bpeOnce         sync.Once
)

// GetTokenizer retrieves the singleton Tiktoken BPE encoder
func GetTokenizer() (*tiktoken.Tiktoken, error) {
	var err error
	bpeOnce.Do(func() {
		bpeInstance, err = tiktoken.GetEncoding(defaultEncoding)
	})
	return bpeInstance, err
}

// CountTokens estimates tokens with the configured local BPE encoding.
func CountTokens(text string) int {
	if text == "" {
		return 0
	}
	enc, err := GetTokenizer()
	if err != nil {
		// Fallback: estimate 1 token per 4 characters if tokenizer fails
		return (len(text) + 3) / 4
	}
	tokens := enc.Encode(text, nil, nil)
	return len(tokens)
}

// EncodeTokens encodes text into an array of Token IDs for LCP prefix comparison
func EncodeTokens(text string) []int {
	if text == "" {
		return nil
	}
	enc, err := GetTokenizer()
	if err != nil {
		return nil
	}
	return enc.Encode(text, nil, nil)
}
