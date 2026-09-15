package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"heimdall/internal/agent_adapters/antigravity"
	"heimdall/internal/core"
	"heimdall/internal/runtime"
	"heimdall/internal/ui"
)

const (
	disabledHTTPPort   = 0
	healthReadTimeout  = 5 * time.Second
	healthWriteTimeout = 10 * time.Second
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output, diagnostics io.Writer) error {
	flags := flag.NewFlagSet("heimdall", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	transcript := flags.String("file", "", "Path to transcript_full.jsonl")
	database := flags.String("db", "", "Path to conversation SQLite database")
	id := flags.String("session", "", "Session ID (latest session when omitted)")
	port := flags.Int("port", disabledHTTPPort, "Optional health endpoint port (0 disables it)")
	plain := flags.Bool("plain", false, "Use scrolling logs")
	reportMode := flags.Bool("report", false, "Read one session and write an analysis report")
	format := flags.String("format", "text", "Report format: text or json")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("unsupported report format %q", *format)
	}
	if *reportMode && (*plain || *port != disabledHTTPPort) {
		return fmt.Errorf("-report cannot be combined with -plain or -port")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	var sessions []core.SessionInfo
	if *id == "" && *transcript == "" && *database == "" {
		if latest, err := antigravity.DiscoverLatestSession(home); err == nil {
			*id, *transcript, *database = latest.SessionID, latest.LogPath, latest.DBPath
			sessions = append(sessions, *latest)
		}
	} else if *id == "" {
		*id = "local"
	}
	if *reportMode {
		if *id == "" {
			return fmt.Errorf("report requires a discoverable session or explicit sources")
		}
		query, err := runtime.LoadSession(home, *id, *transcript, *database)
		if err != nil {
			return err
		}
		return core.WriteAnalysisReport(output, core.NewAnalysisService(query).Report(), *format)
	}
	if *plain && *id == "" {
		return fmt.Errorf("plain mode requires a discoverable session or explicit sources")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	supervisor, err := runtime.NewSessionSupervisor(ctx)
	if err != nil {
		return err
	}
	if *id != "" {
		go func() { _ = supervisor.StartSession(*id, *transcript, *database) }()
	}
	if *port != disabledHTTPPort {
		go startHTTPServer(ctx, *port)
	}
	if *plain {
		return runPlainLogMode(ctx, supervisor.Updates(), output)
	}
	model := ui.NewModelWithData(*id, *id == "", ui.ModelData{Sessions: sessions}, supervisor)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithContext(ctx))
	go func() {
		catalog, err := antigravity.DiscoverAllSessions(home)
		if err == nil {
			program.Send(ui.SessionCatalogMsg{Sessions: catalog})
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-supervisor.Updates():
				program.Send(ui.SessionUpdateMsg(update))
			}
		}
	}()
	_, err = program.Run()
	return err
}

func runPlainLogMode(ctx context.Context, updates <-chan core.SessionUpdate, output io.Writer) error {
	previous := make(map[int]core.UnifiedAgentEvent)
	var epoch uint64
	var lastHealth core.MonitorState
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if update.SwitchError != "" && !update.Ready {
				return fmt.Errorf("load session: %s", update.SwitchError)
			}
			if update.Health.State != "" && update.Health.State != lastHealth {
				if _, err := fmt.Fprintf(output, "Source: %s %s\n", update.Health.State, update.Health.Error); err != nil {
					return err
				}
				lastHealth = update.Health.State
			}
			if !update.Ready {
				continue
			}
			if epoch != update.Epoch {
				previous = make(map[int]core.UnifiedAgentEvent)
				epoch = update.Epoch
			}
			current := make(map[int]core.UnifiedAgentEvent)
			for _, event := range core.ProjectSessionEvents(update.Session) {
				current[event.StepIndex] = event
				if old, ok := previous[event.StepIndex]; !ok || !reflect.DeepEqual(old, event) {
					if _, err := fmt.Fprintln(output, core.FormatTokenBreakdownTable(event)); err != nil {
						return err
					}
				}
			}
			previous = current
		}
	}
}

func startHTTPServer(ctx context.Context, port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","app":"heimdall","scope":"process"}`)
	})
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux, ReadTimeout: healthReadTimeout, WriteTimeout: healthWriteTimeout}
	go func() { <-ctx.Done(); _ = server.Close() }()
	_ = server.ListenAndServe()
}
