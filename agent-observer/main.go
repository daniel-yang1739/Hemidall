package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"agent-observer/internal/adapters"
	"agent-observer/internal/adapters/antigravity"
	"agent-observer/internal/core"
	"agent-observer/internal/ui"
)

const (
	defaultTranscriptPath = "/Users/daniel_y_yang/.gemini/antigravity-cli/brain/aa726359-08e2-4687-a15c-073a2f4a705b/.system_generated/logs/transcript_full.jsonl"
	defaultSessionID      = "aa726359-08e2-4687-a15c-073a2f4a705b"
	version               = "v0.4.0-k9s-tui"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultDBPath := filepath.Join(homeDir, ".gemini", "antigravity-cli", "conversations", defaultSessionID+".db")

	filePath := flag.String("file", defaultTranscriptPath, "Path to the agent transcript_full.jsonl file")
	dbPath := flag.String("db", defaultDBPath, "Path to SQLite conversation database (for official Gemini telemetry)")
	port := flag.Int("port", 8080, "HTTP server port for dashboard & health check")
	adapterName := flag.String("adapter", "antigravity", "Adapter to use (antigravity, generic_jsonl)")
	sessionID := flag.String("session", defaultSessionID, "Session ID to track")
	plainMode := flag.Bool("plain", false, "Use plain scrolling log mode instead of full-screen interactive TUI")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize core payload analyzer
	analyzer := core.NewPayloadAnalyzer()

	// Initialize event channel with large buffer for seamless history warmup
	eventChan := make(chan core.UnifiedAgentEvent, 10000)

	// Initialize adapter based on CLI flag
	var adapter adapters.AgentAdapter
	switch *adapterName {
	case "antigravity":
		adapter = antigravity.NewWatcher(*filePath, *sessionID, analyzer, *dbPath)
	default:
		adapter = antigravity.NewWatcher(*filePath, *sessionID, analyzer, *dbPath)
	}

	// Start background HTTP server
	go startHTTPServer(*port)

	// Start adapter event ingestion in background goroutine
	go func() {
		if err := adapter.Start(ctx, eventChan); err != nil && err != context.Canceled {
			fmt.Printf("❌ Adapter error: %v\n", err)
		}
	}()

	// If plain mode requested, run traditional scrolling CLI
	if *plainMode {
		runPlainLogMode(ctx, cancel, *port, *adapterName, *filePath, *dbPath, analyzer, eventChan)
		return
	}

	// ==================== FULL-SCREEN INTERACTIVE TUI (k9s STYLE) ====================
	initialModel := ui.NewModel(*sessionID)
	p := tea.NewProgram(initialModel, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Goroutine to forward events from adapter channel to Bubble Tea program
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-eventChan:
				analyzer.AnalyzeStep(&event)
				p.Send(ui.AgentEventMsg(event))
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("❌ Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runPlainLogMode(ctx context.Context, cancel context.CancelFunc, port int, adapter string, file string, db string, analyzer *core.PayloadAnalyzer, eventChan chan core.UnifiedAgentEvent) {
	printBanner(port, adapter, file, db)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Received shutdown signal. Gracefully exiting...")
		cancel()
	}()

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
			analyzer.AnalyzeStep(&event)
			printEventLine(event)
			if event.Type == core.StepTypeUserInput || event.Type == core.StepTypeModelResponse || event.Type == core.StepTypeToolCall || stepCount%5 == 0 {
				fmt.Print(core.FormatTokenBreakdownTable(event))
				fmt.Println()
			}
		}
	}
}

func printBanner(port int, adapter string, file string, db string) {
	fmt.Println(`
┌────────────────────────────────────────────────────────────┐
│  🐹 AGENT-OBSERVER: Phase 3 (Official Telemetry & Context) │
│  Version   : ` + version + `                          │
│  Adapter   : ` + fmt.Sprintf("%-45s", adapter) + ` │
│  Telemetry : Dual-Track (SQLite gen_metadata + BPE Engine) │
│  HTTP Port : ` + fmt.Sprintf("%-45s", fmt.Sprintf("http://localhost:%d/healthz", port)) + ` │
│  Log File  : ` + fmt.Sprintf("%-45s", truncatePath(file, 45)) + ` │
│  SQLite DB : ` + fmt.Sprintf("%-45s", truncatePath(db, 45)) + ` │
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
