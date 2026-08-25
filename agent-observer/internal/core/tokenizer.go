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

// GetTokenizer 取得單例 Tiktoken BPE 編碼器
func GetTokenizer() (*tiktoken.Tiktoken, error) {
	var err error
	bpeOnce.Do(func() {
		bpeInstance, err = tiktoken.GetEncoding(defaultEncoding)
	})
	return bpeInstance, err
}

// CountTokens 計算純文字的 Token 數量
func CountTokens(text string) int {
	if text == "" {
		return 0
	}
	enc, err := GetTokenizer()
	if err != nil {
		// Fallback: 粗估約每 4 個字元 1 個 token
		return (len(text) + 3) / 4
	}
	tokens := enc.Encode(text, nil, nil)
	return len(tokens)
}

// EncodeTokens 將文字編碼為 Token ID 列表 (供前綴比對 LCP 使用)
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
