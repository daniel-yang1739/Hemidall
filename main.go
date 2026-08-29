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

	"heimdall/internal/adapters"
	"heimdall/internal/adapters/antigravity"
	"heimdall/internal/core"
	"heimdall/internal/ui"
)

const (
	defaultTranscriptPath = "/Users/daniel_y_yang/.gemini/antigravity-cli/brain/aa726359-08e2-4687-a15c-073a2f4a705b/.system_generated/logs/transcript_full.jsonl"
	defaultSessionID      = "aa726359-08e2-4687-a15c-073a2f4a705b"
	version               = "v0.5.0-session-switcher"
)

type PlainLogConfig struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	Port      int
	Adapter   string
	File      string
	DB        string
	Analyzer  *core.PayloadAnalyzer
	EventChan chan core.UnifiedAgentEvent
}

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultDBPath := filepath.Join(homeDir, ".gemini", "antigravity-cli", "conversations", defaultSessionID+".db")

	filePath := flag.String("file", "", "Path to the agent transcript_full.jsonl file (leave empty for auto-detection)")
	dbPath := flag.String("db", "", "Path to SQLite conversation database (leave empty for auto-detection)")
	port := flag.Int("port", 8080, "HTTP server port for dashboard & health check")
	adapterName := flag.String("adapter", "antigravity", "Adapter to use (antigravity, generic_jsonl)")
	sessionID := flag.String("session", "", "Session ID to track (leave empty for auto-detection of latest active session)")
	plainMode := flag.Bool("plain", false, "Use plain scrolling log mode instead of full-screen interactive TUI")
	flag.Parse()

	sessionPassed := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "session" {
			sessionPassed = true
		}
	})

	targetSessionID := *sessionID
	targetFilePath := *filePath
	targetDBPath := *dbPath

	// Auto-detect latest active session if session flag not passed
	if !sessionPassed || targetSessionID == "" {
		if latest, err := antigravity.GetLatestActiveSession(); err == nil && latest != nil {
			targetSessionID = latest.SessionID
			if targetFilePath == "" {
				targetFilePath = latest.LogPath
			}
			if targetDBPath == "" {
				targetDBPath = latest.DBPath
			}
		} else {
			targetSessionID = defaultSessionID
			if targetFilePath == "" {
				targetFilePath = defaultTranscriptPath
			}
			if targetDBPath == "" {
				targetDBPath = defaultDBPath
			}
		}
	}

	openSwitcherOnStart := false

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize core payload analyzer
	analyzer := core.NewPayloadAnalyzer()

	// Initialize event channel with large buffer for seamless history warmup
	eventChan := make(chan core.UnifiedAgentEvent, 10000)

	// Initialize Dynamic Watcher Hub
	hub := adapters.NewWatcherHub(ctx, eventChan, analyzer)
	_ = hub.StartSession(targetSessionID, core.AgentTypeAntigravity, targetFilePath, targetDBPath)

	// Start background HTTP server
	go startHTTPServer(*port)

	// If plain mode requested, run traditional scrolling CLI
	if *plainMode {
		runPlainLogMode(PlainLogConfig{
			Ctx:       ctx,
			Cancel:    cancel,
			Port:      *port,
			Adapter:   *adapterName,
			File:      targetFilePath,
			DB:        targetDBPath,
			Analyzer:  analyzer,
			EventChan: eventChan,
		})
		return
	}

	// ==================== FULL-SCREEN INTERACTIVE TUI (k9s STYLE WITH DYNAMIC WATCHER HUB) ====================
	initialModel := ui.NewModel(targetSessionID, openSwitcherOnStart, hub)
	p := tea.NewProgram(initialModel, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Goroutine to forward events from adapter channel to Bubble Tea program
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-eventChan:
				p.Send(ui.AgentEventMsg(event))
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("❌ Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runPlainLogMode(config PlainLogConfig) {
	printBanner(config.Port, config.Adapter, config.File, config.DB)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Received shutdown signal. Gracefully exiting...")
		config.Cancel()
	}()

	fmt.Println("👀 Watching agent events & analyzing Context in real-time... (Press Ctrl+C to stop)")
	fmt.Println()

	for {
		select {
		case <-config.Ctx.Done():
			return
		case event := <-config.EventChan:
			config.Analyzer.AnalyzeStep(&event)
			table := core.FormatTokenBreakdownTable(event)
			fmt.Println(table)
		}
	}
}

func printBanner(port int, adapter, file, db string) {
	fmt.Println("================================================================================")
	fmt.Printf("  🐹 HEIMDALL %s (Go Real-Time LLM Context Telemetry)\n", version)
	fmt.Println("================================================================================")
	fmt.Printf("  • Local Server Port : http://localhost:%d\n", port)
	fmt.Printf("  • Active Adapter    : %s\n", adapter)
	fmt.Printf("  • Log File Path     : %s\n", file)
	fmt.Printf("  • SQLite Telemetry  : %s\n", db)
	fmt.Println("================================================================================")
	fmt.Println()
}

func startHTTPServer(port int) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","app":"heimdall"}`))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	_ = server.ListenAndServe()
}
