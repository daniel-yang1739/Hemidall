package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-observer/internal/adapters"
	"agent-observer/internal/adapters/antigravity"
	"agent-observer/internal/core"
)

const (
	defaultTranscriptPath = "/Users/daniel_y_yang/.gemini/antigravity-cli/brain/aa726359-08e2-4687-a15c-073a2f4a705b/.system_generated/logs/transcript_full.jsonl"
	defaultSessionID      = "aa726359-08e2-4687-a15c-073a2f4a705b"
	version               = "v0.2.0-phase2"
)

func main() {
	filePath := flag.String("file", defaultTranscriptPath, "Path to the agent transcript_full.jsonl file")
	port := flag.Int("port", 8080, "HTTP server port for dashboard & health check")
	adapterName := flag.String("adapter", "antigravity", "Adapter to use (antigravity, generic_jsonl)")
	sessionID := flag.String("session", defaultSessionID, "Session ID to track")
	showDetails := flag.Bool("details", true, "Show 5-dimension ASCII Token breakdown table")
	flag.Parse()

	printBanner(*port, *adapterName, *filePath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 監聽系統的中斷信號 (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Received shutdown signal. Gracefully exiting...")
		cancel()
	}()

	// 建立核心 Context 分析器
	analyzer := core.NewPayloadAnalyzer()

	// 建立事件傳輸 Channel
	eventChan := make(chan core.UnifiedAgentEvent, 100)

	// 根據參數挑選 Adapter
	var adapter adapters.AgentAdapter
	switch *adapterName {
	case "antigravity":
		adapter = antigravity.NewWatcher(*filePath, *sessionID, analyzer)
	default:
		fmt.Printf("❌ Unknown adapter '%s', falling back to antigravity\n", *adapterName)
		adapter = antigravity.NewWatcher(*filePath, *sessionID, analyzer)
	}

	// 啟動背景 HTTP Server
	go startHTTPServer(*port)

	// 在 Goroutine 啟動 Adapter 監聽
	go func() {
		if err := adapter.Start(ctx, eventChan); err != nil && err != context.Canceled {
			fmt.Printf("❌ Adapter error: %v\n", err)
		}
	}()

	// 主線程：即時消費事件並於 Console 格式化輸出
	fmt.Println("👀 Watching agent events & analyzing Context in real-time... (Press Ctrl+C to stop)")
	fmt.Println()

	stepCount := 0
	for {
		select {
		case <-ctx.Done():
			fmt.Println("👋 Agent-Observer stopped.")
			return
		case event := <-eventChan:
			stepCount++
			// 1. 核心分析：注入 5 維度 Token 與快取指標
			analyzer.AnalyzeStep(&event)

			// 2. 印出單行摘要
			printEventLine(event)

			// 3. 若為重要回合或開啟 details，印出 ASCII Token 表格
			if *showDetails && (event.Type == core.StepTypeUserInput || event.Type == core.StepTypeModelResponse || event.Type == core.StepTypeToolCall || stepCount%5 == 0) {
				fmt.Print(core.FormatTokenBreakdownTable(event))
				fmt.Println()
			}
		}
	}
}

func printBanner(port int, adapter string, file string) {
	fmt.Println(`
┌────────────────────────────────────────────────────────────┐
│  🐹 AGENT-OBSERVER: Phase 2 (Context Token Analyzer)       │
│  Version   : ` + version + `                                │
│  Adapter   : ` + fmt.Sprintf("%-45s", adapter) + ` │
│  Tokenizer : cl100k_base (BPE) / TikToken-Go               │
│  HTTP Port : ` + fmt.Sprintf("%-45s", fmt.Sprintf("http://localhost:%d/healthz", port)) + ` │
│  File Path : ` + fmt.Sprintf("%-45s", truncatePath(file, 45)) + ` │
└────────────────────────────────────────────────────────────┘`)
}

func printEventLine(e core.UnifiedAgentEvent) {
	timeStr := e.Timestamp.Format("15:04:05")
	typeBadge := fmt.Sprintf("[Step %03d | %-14s]", e.StepIndex, e.Type)
	statusBadge := fmt.Sprintf("[%s]", e.Status)

	fmt.Printf("[%s] %s %-34s %s\n", timeStr, typeBadge, e.Summary, statusBadge)
	if len(e.ToolCalls) > 0 {
		for _, tc := range e.ToolCalls {
			fmt.Printf("           ↳ 🛠️  Tool: %s (args: %d)\n", tc.ToolName, len(tc.Arguments))
		}
	}
}

func startHTTPServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"%s","time":"%s"}`, version, time.Now().Format(time.RFC3339))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	_ = server.ListenAndServe()
}

func truncatePath(p string, maxLen int) string {
	if len(p) <= maxLen {
		return p
	}
	return "..." + p[len(p)-(maxLen-3):]
}
