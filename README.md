```text
██╗  ██╗███████╗██╗███╗   ███╗██████╗  █████╗ ██╗     ██╗     
██║  ██║██╔════╝██║████╗ ████║██╔══██╗██╔══██╗██║     ██║     
███████║█████╗  ██║██╔████╔██║██║  ██║███████║██║     ██║     
██╔══██║██╔══╝  ██║██║╚██╔╝██║██║  ██║██╔══██║██║     ██║     
██║  ██║███████╗██║██║ ╚═╝ ██║██████╔╝██║  ██║███████╗███████╗
╚═╝  ╚═╝╚══════╝╚═╝╚═╝     ╚═╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝
```

# 🛡️ HEIMDALL: The All-Seeing AI Agent Observability & Telemetry Engine

> *"I can see nine realms, nine trillion souls, and every step across the stars."*  
> — **Heimdall, Guardian of the Bifrost**

---

## 👁️ What is Heimdall?

In Norse myth and Marvel lore, **Heimdall** is the all-seeing guardian of the Bifrost bridge, gifted with golden eyes that perceive every soul, every energy fluctuation, and every hidden movement across the cosmos.

**Heimdall for AI Agents** is a local, read-only terminal observability tool for Antigravity sessions. It connects persisted usage, visible context, and tool activity so developers can investigate resource consumption and establish a baseline for future optimization experiments.

```
┌───────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    🌐 THE ALL-SEEING ENGINE                                       │
│                                                                                                   │
│  [Agent Memory Stream]  ──┐                                                                       │
│  [SQLite WAL Protobuf]  ──┼──>  🛡️ HEIMDALL  ──>  [5-Dim Context Anatomy | KV Cache Forensics]   │
│  [Inbound User Intent]  ──┘          (TUI)        [Time-Machine Replay  | Multiverse Accounting] │
└───────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## ⚡ Core Capabilities

### 🔍 1. 5-Dimension Context Anatomy
Never wonder what was sent to the model. Heimdall breaks down complex outbound payloads in real time:
- **Persisted System Prompt and Identity Subsection** (`<identity>` is a subsection, not a synonym for the full prompt)
- **Constitutional Rules** (`AGENTS.md`, `GEMINI.md`, Guidelines)
- **Persisted Tool Definitions and MCP Attribution State** (tool entries are observed; MCP ownership can be unknown)
- **Conversation History & Compaction Checkpoints** (Active window vs. truncated turns)
- **Active Inbound Prompt & Buffers** (User requests and staged tool outputs)

### 👑 2. Dual-Track Telemetry & KV Cache Forensics
Heimdall directly decodes **Google Cloud gRPC Protobufs** from local SQLite WAL without requiring intrusive HTTP proxies or custom CA certificates:
- **Cache Input Observations**: Separates recorded cached input, uncached input, and explicitly labelled inferred defaults. Local artifacts do not establish GPU cache residency or an invoice.
- **Observed Generation Telemetry**: Displays values decoded from local generation metadata, with provenance and explicit fallback status.

### ⏳ 3. Time Machine Replay & Causality Graph
- **`k9s`-style Terminal Navigation**: Instant search (`/`), category filters, and step causality links.
- **Non-Destructive Time Travel**: Inspect the exact state of the agent at Step #100 or Step #5,000 without mutating runtime state.
- **Vim Command Mode**: Press `:` for `:q` (safe quit) or `:w` (payload export), and press `y` to yank payload directly to system clipboard.

### 4. Session Insights and Reproducible Reports

Press `5` for Insights. Use `h`/`l` to switch between large tool outputs, context growth, token consumers, and reference-cost consumers. Use `j`/`k` to select a ranked item and `Enter` to inspect its transcript evidence. The TUI shows the first ten entries; exported reports retain the full rankings.

Use `:report` to export the displayed analysis revision to a new `heimdall-report-<timestamp>-<suffix>.json` file in the current directory. Files are created exclusively with owner-only permissions. Reports contain measurements, pricing assumptions, and evidence locations, but omit raw prompts and tool outputs.

For a one-shot report without starting the TUI or a monitor:

```bash
./bin/heimdall -report -session SESSION_ID
./bin/heimdall -report -session SESSION_ID -format json
./bin/heimdall -report -session example -file /path/transcript_full.jsonl -db /path/conversation.db -format json
```

JSON reports use `schema_version: 1`. Measurements distinguish observed, derived, locally estimated, and unavailable values. Missing model prices remain unavailable; incomplete cost totals are labelled partial. Rates are captured with each priced generation so a saved estimate can be reproduced.

Context growth compares adjacent, linked generations of the same model and source, without crossing checkpoints or known generation gaps. Tool output sizes are local text estimates: they do not prove repeated transmission, precise per-tool billing, or compressor savings. Historical reconstruction never borrows a future context snapshot. The current milestone provides observation and analysis; it does not run a compressor or call an LLM.

Source health is separate from agent activity. Idle sources remain healthy. Read failures retain the last successful snapshot and retry with bounded backoff; failed session switches retain the active session. Insights displays the last successful read, and source failures remain visible in the footer.

---

## 🚀 Quickstart

### Prerequisites
- Go 1.26+ (the version declared in `go.mod`)
- A terminal on macOS or Linux. Antigravity artifacts must exist locally; explicit source paths can be supplied for offline analysis.

### Build & Run

```bash
# Clone the repository
git clone git@github.com:daniel-yang1739/heimdall.git
cd heimdall

# Run immediately
make run

# Or compile the standalone binary
make build
./bin/heimdall
```

---

## ⌨️ Keyboard Shortcuts Reference

| Keybinding | Action | Description |
| :--- | :--- | :--- |
| `1`, `2`, `3`, `4`, `5` | Switch Views | Jump to Dashboard, History, Context, Docs, or Insights |
| `Tab` / `Shift+Tab` | Cyclic Switch | Cycle forward and backward through all views |
| `j` / `k` (or `↑`/`↓`) | Navigate Items | Move selection up and down in tree and step lists |
| `l` / `h` (or `Enter`/`Esc`) | Focus Pane | Switch focus between left list and right inspector |
| `r` | Dual-Mode Toggle | Toggle between **Refined Analysis** and **Decoded Evidence JSON** |
| `y` or `c` | Clipboard Yank | Copy currently inspected payload directly to system clipboard |
| `:` | Vim Command Bar | Type `:q` to quit, `:w` to export JSON, `:h` for help |
| `Ctrl+P` | Session Switcher | Fuzzy switch across active and historical agent sessions |
| `?` | Shortcuts Modal | Display interactive keyboard cheat sheet |
| `:report` | Analysis Export | Export the displayed session report as JSON |

---

## 🏛️ Architecture

Heimdall separates provider artifacts from analysis and presentation:

- `internal/core/`: Session domain, immutable query snapshots, token estimates, shared usage accounting, and versioned analysis reports.
- `internal/agent_adapters/antigravity/`: Read-only discovery, transcript/SQLite parsers, and transactional session refresh.
- `internal/runtime/`: Session activation epochs, monitoring, health reporting, retry, and snapshot delivery.
- `internal/ui/`: Bubble Tea views and background analysis orchestration. Provider schemas and database reads stay outside the UI.

### Verification

```bash
go test ./...
go test -race ./...
go vet ./...
go test ./internal/core -run '^$' -bench BenchmarkSessionAnalysis -benchmem
```

Tests use synthetic transcript and SQLite fixtures. CI runs tests, race checks, and vet on macOS and Linux using the Go version from `go.mod`.

---

## 📜 License
MIT License. Crafted for transparent, observable, and trustworthy AI Agents.
