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

	tea "github.com/charmbracelet/bubbletea"

	"heimdall/internal/adapters"
	"heimdall/internal/adapters/antigravity"
	"heimdall/internal/core"
	"heimdall/internal/ui"
)

const (
	disabledHTTPPort          = 0
	eventChannelCapacity      = 10_000
	historyBatchSize          = 500
	historyBatchFlushInterval = 50 * time.Millisecond
	version                   = "v0.5.0-session-switcher"
)

type PlainLogConfig struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	Port      int
	File      string
	DB        string
	EventChan chan core.UnifiedAgentEvent
}

func main() {
	filePath := flag.String("file", "", "Path to the agent transcript_full.jsonl file (leave empty for auto-detection)")
	dbPath := flag.String("db", "", "Path to SQLite conversation database (leave empty for auto-detection)")
	port := flag.Int("port", disabledHTTPPort, "HTTP server port for the optional health check (0 disables it)")
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
	sessions := make([]core.SessionInfo, 0, 1)

	// Resolve only the latest session synchronously. Full catalog discovery runs
	// after the TUI starts so one slow historical database cannot delay first paint.
	if !sessionPassed || targetSessionID == "" {
		if latest, latestErr := antigravity.GetLatestActiveSession(); latestErr == nil {
			sessions = append(sessions, *latest)
			targetSessionID = latest.SessionID
			if targetFilePath == "" {
				targetFilePath = latest.LogPath
			}
			if targetDBPath == "" {
				targetDBPath = latest.DBPath
			}
		}
	}

	openSwitcherOnStart := targetSessionID == ""

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize core payload analyzer
	analyzer := core.NewPayloadAnalyzer()

	// Initialize event channel with large buffer for seamless history warmup
	eventChan := make(chan core.UnifiedAgentEvent, eventChannelCapacity)

	// Initialize Dynamic Watcher Hub
	hub := adapters.NewWatcherHub(ctx, eventChan, analyzer)
	if targetSessionID != "" {
		if err := hub.StartSession(targetSessionID, targetFilePath, targetDBPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start session watcher: %v\n", err)
			os.Exit(1)
		}
	}

	// The TUI is the product surface. Keep the auxiliary health endpoint opt-in
	// so it does not claim a web dashboard or reserve a port by default.
	if *port != disabledHTTPPort {
		go startHTTPServer(*port)
	}

	// If plain mode requested, run traditional scrolling CLI
	if *plainMode {
		if targetSessionID == "" {
			fmt.Fprintln(os.Stderr, "Plain mode requires a discoverable session or an explicit -session value")
			return
		}
		runPlainLogMode(PlainLogConfig{
			Ctx:       ctx,
			Cancel:    cancel,
			Port:      *port,
			File:      targetFilePath,
			DB:        targetDBPath,
			EventChan: eventChan,
		})
		return
	}

	// ==================== FULL-SCREEN INTERACTIVE TUI (k9s STYLE WITH DYNAMIC WATCHER HUB) ====================
	contextBuilder := antigravity.NewContextPayloadBuilder()
	initialModel := ui.NewModelWithData(targetSessionID, openSwitcherOnStart, ui.ModelData{
		Sessions: sessions,
		ContextPayloadBuilder: func(history []core.UnifiedAgentEvent, sessionID string) core.AgentContextPayload {
			return contextBuilder.Build(history, sessionID, "")
		},
	}, hub)
	p := tea.NewProgram(initialModel, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Decode the static snapshot and its visible-text estimate off the UI thread.
	// The result is session-scoped and the builder memoizes the parsed snapshot
	// for later Context-view rendering.
	if targetSessionID != "" {
		go func(sessionID string) {
			payload := contextBuilder.Build(nil, sessionID, "")
			p.Send(ui.ContextEstimateMsg{SessionID: sessionID, Estimate: core.EstimateVisibleContextEvidence(payload)})
		}(targetSessionID)
	}

	// Discover the switcher catalog in the background. This is deliberately
	// independent from live watcher startup and from the first TUI frame.
	go func() {
		catalog, catalogErr := antigravity.DiscoverAllSessions()
		if catalogErr == nil {
			p.Send(ui.SessionCatalogMsg{Sessions: catalog})
		}
	}()

	// Forward transcript hydration in coherent batches. The watcher first emits a
	// 10-event preview, then historical replay arrives in 500-row batches. This
	// avoids one Bubble Tea redraw and Context invalidation per transcript row.
	go func() {
		ticker := time.NewTicker(historyBatchFlushInterval)
		defer ticker.Stop()
		batch := make([]core.UnifiedAgentEvent, 0, historyBatchSize)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			events := make([]core.UnifiedAgentEvent, len(batch))
			copy(events, batch)
			p.Send(ui.HistoryBatchMsg{Events: events})
			batch = batch[:0]
		}

		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case event := <-eventChan:
				batch = append(batch, event)
				if len(batch) >= historyBatchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("❌ Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runPlainLogMode(config PlainLogConfig) {
	printBanner(config.Port, config.File, config.DB)

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
			table := core.FormatTokenBreakdownTable(event)
			fmt.Println(table)
		}
	}
}

func printBanner(port int, file, db string) {
	fmt.Println("================================================================================")
	fmt.Printf("  🐹 HEIMDALL %s (Go Real-Time LLM Context Telemetry)\n", version)
	fmt.Println("================================================================================")
	if port == disabledHTTPPort {
		fmt.Println("  • Optional Health Endpoint : disabled")
	} else {
		fmt.Printf("  • Optional Health Endpoint : http://localhost:%d\n", port)
	}
	fmt.Println("  • Source Adapter    : antigravity")
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
