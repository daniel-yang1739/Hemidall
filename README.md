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

**Heimdall for AI Agents** is an ultra-fast, zero-overhead, non-invasive terminal observability engine (TUI) designed to **crack open the AI Agent Black Box**. It acts as your **Eye of Truth**—illuminating the hidden thoughts, context anatomy, KV cache dynamics, and cloud bills of modern autonomous coding agents.

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
- **L1 GPU HBM vs. L2 Global Prefix Cache**: Explains why requests idle for hours can still achieve an **87.9% Cache Hit rate (0.25x discount)** via persistent prefix hashing.
- **Observed Generation Telemetry**: Displays values decoded from local generation metadata, with provenance and explicit fallback status.

### ⏳ 3. Time Machine Replay & Causality Graph
- **`k9s`-style Terminal Navigation**: Instant search (`/`), category filters, and step causality links.
- **Non-Destructive Time Travel**: Inspect the exact state of the agent at Step #100 or Step #5,000 without mutating runtime state.
- **Vim Command Mode**: Press `:` for `:q` (safe quit) or `:w` (payload export), and press `y` to yank payload directly to system clipboard.

---

## 🚀 Quickstart

### Prerequisites
- Go 1.22+
- macOS, Linux, or Windows Terminal

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
| `1`, `2`, `3`, `4` | Switch Views | Jump to `[1] Dashboard`, `[2] Context`, `[3] History`, `[4] Docs` |
| `Tab` / `Shift+Tab` | Cyclic Switch | Cycle forward and backward through all views |
| `j` / `k` (or `↑`/`↓`) | Navigate Items | Move selection up and down in tree and step lists |
| `l` / `h` (or `Enter`/`Esc`) | Focus Pane | Switch focus between left list and right inspector |
| `r` | Dual-Mode Toggle | Toggle between **Refined Analysis** and **Decoded Evidence JSON** |
| `y` or `c` | Clipboard Yank | Copy currently inspected payload directly to system clipboard |
| `:` | Vim Command Bar | Type `:q` to quit, `:w` to export JSON, `:h` for help |
| `Ctrl+P` | Session Switcher | Fuzzy switch across active and historical agent sessions |
| `?` | Shortcuts Modal | Display interactive keyboard cheat sheet |

---

## 🏛️ Architecture

Heimdall follows a strict **Hexagonal Architecture (Ports & Adapters)**:

- `internal/core/`: Universal Agent Finite State Machine (FSM), BPE Tokenizer, and Reverse Sliding Window Accounting Algorithm.
- `internal/adapters/`: Non-invasive readers (Antigravity SQLite WAL, Protobuf decoders, directory watchers).
- `internal/ui/`: Bubbletea & Lipgloss reactive terminal user interface engine.

---

## 📜 License
MIT License. Crafted for transparent, observable, and trustworthy AI Agents.
