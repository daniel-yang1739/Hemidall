package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/pkoukk/tiktoken-go"
)

func main() {
	filePathFlag := flag.String("file", "", "Path to text/markdown file (leave empty to pass positional argument or read from stdin)")
	flag.Parse()

	targetPath := *filePathFlag
	if targetPath == "" && flag.NArg() > 0 {
		targetPath = flag.Arg(0)
	}

	var data []byte
	var err error

	if targetPath != "" {
		data, err = os.ReadFile(targetPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", targetPath, err)
			os.Exit(1)
		}
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Println("Usage: go run ./cmd/count-tokens <path-to-file.md>")
			fmt.Println("   or: go run ./cmd/count-tokens -file <path-to-file.md>")
			fmt.Println("   or: cat <file.md> | go run ./cmd/count-tokens")
			os.Exit(1)
		}
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}

	text := string(data)
	charCount := utf8.RuneCountInString(text)
	byteCount := len(data)
	lineCount := strings.Count(text, "\n")
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		lineCount++
	}

	fmt.Println("================================================================================")
	fmt.Println("  📊 TOKEN & CHARACTER COUNTER")
	fmt.Println("================================================================================")
	if targetPath != "" {
		fmt.Printf("  • Target File      : %s\n", targetPath)
	} else {
		fmt.Println("  • Target Source    : STDIN (Pipe)")
	}
	fmt.Printf("  • Total Bytes      : %d Bytes\n", byteCount)
	fmt.Printf("  • Total Characters : %d Runes\n", charCount)
	fmt.Printf("  • Total Lines      : %d Lines\n", lineCount)
	fmt.Println("--------------------------------------------------------------------------------")

	// 1. cl100k_base (ChatGPT / GPT-4 / Claude reference BPE)
	encCl100k, err1 := tiktoken.GetEncoding("cl100k_base")
	if err1 == nil {
		toks := encCl100k.Encode(text, nil, nil)
		fmt.Printf("  • cl100k_base Tokens (Standard BPE) : %d tokens\n", len(toks))
	}

	// 2. o200k_base (GPT-4o / Modern BPE)
	encO200k, err2 := tiktoken.GetEncoding("o200k_base")
	if err2 == nil {
		toks := encO200k.Encode(text, nil, nil)
		fmt.Printf("  • o200k_base Tokens (Omni BPE)     : %d tokens\n", len(toks))
	}

	// 3. p50k_base (Legacy Codex / GPT-3)
	encP50k, err3 := tiktoken.GetEncoding("p50k_base")
	if err3 == nil {
		toks := encP50k.Encode(text, nil, nil)
		fmt.Printf("  • p50k_base Tokens (Legacy BPE)   : %d tokens\n", len(toks))
	}

	fmt.Println("================================================================================")
}
