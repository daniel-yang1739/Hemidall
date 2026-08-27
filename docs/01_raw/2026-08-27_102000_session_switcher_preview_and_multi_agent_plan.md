# Quick Switcher v2.0 Implementation Plan: Pure Visual Tabs, Focused Dual-Pane Inspector, and Zero Emoji

> **Status**: Planning  
> **Refined Design Principles**:
> 1. **Pure Visual Tabs (No 'Tabs:' label)**: Seamlessly styled tab buttons (`[Antigravity (6)]` / `[Claude Code (0)]` / `[OpenCode (0)]`) with active background highlight, switched via `[` / `]`.
> 2. **Zero Emojis**: 100% pure ASCII terminal typography, clean brackets, and professional styling.
> 3. **Clean Left List**: Shows only the last 2-3 directory segments (`self/ithome2026`), short hash (`#aa726359`), step count, relative time, and active badge.
> 4. **Focused Right Inspector**: Dedicated purely to `[INITIAL GOAL / FIRST PROMPT]` and `[LATEST PROGRESS / LAST ACTION]` (for distinguishing forked sessions).

---

## 1. Focused Pure-Text UI Layout (Real Visual Tabs)

```text
+------------------------------- SWITCH AGENT SESSION [Ctrl+P] --------------------------------+
|  [Antigravity (6)]   Claude Code (0)   OpenCode (0)                                           |
| Filter: [ithome_                       ] (Type to filter by path, hash, or prompt keywords)   |
|-------------------------------------------+---------------------------------------------------|
| > [AGY] self/ithome2026 (#aa726359)       | [INITIAL GOAL / FIRST PROMPT]                     |
|   307 steps | 0.8 MB | 10m ago | [ACTIVE] |   "In the Iron Man series, plain terminal text    |
|                                           |   logs are hardcore, but if readers can see a     |
|   [AGY] self/bookkeeper (#915575c9)       |   clear Web Dashboard, article appeal doubles..." |
|   12 steps | 0.1 MB | 1d ago              |                                                   |
|                                           |---------------------------------------------------|
|   [AGY] .gemini/antigravity-cli (#8eed)   | [LATEST PROGRESS / LAST ACTION]                   |
|   45 steps | 0.3 MB | 8d ago              |   "Optimize the session search page, show the     |
|                                           |   last 2-3 directories of the workspace path,     |
|   [AGY] self/scratch (#e261c6af)          |   and remove redundant overview block..."         |
|   1 step | 0.0 MB | 8d ago                |                                                   |
+-------------------------------------------+---------------------------------------------------+
  [Up/Down, Ctrl+j/k] Navigate  [Enter] Attach Session  [[ / ]] Agent Tab  [Esc] Close
```

---

## 2. Directory Formatting Logic (Short Path)

Extracts the last 2-3 directories of the workspace path:
* `/Users/daniel_y_yang/Documents/self/ithome2026` -> `self/ithome2026`
* `/Users/daniel_y_yang/Documents/self/bookkeeper` -> `self/bookkeeper`
* `/Users/daniel_y_yang/.gemini/antigravity-cli` -> `.gemini/antigravity-cli`
* Fallback -> `workspace`

---

## 3. Core Architecture & Modules

### 1. Model Extension (`internal/core/session.go`)
```go
package core

import "time"

type AgentType string

const (
    AgentTypeAntigravity AgentType = "antigravity"
    AgentTypeClaudeCode  AgentType = "claudecode"
    AgentTypeOpenCode    AgentType = "opencode"
)

type SessionInfo struct {
    AgentType     AgentType // "antigravity" | "claudecode" | "opencode"
    SessionID     string    // "aa726359-08e2-4687-a15c-073a2f4a705b"
    WorkspaceDir  string    // Full path: "/Users/daniel_y_yang/Documents/self/ithome2026"
    ShortPath     string    // Short path: "self/ithome2026"
    InitialGoal   string    // First user request clean text
    LastPrompt    string    // Latest user request/action clean text
    StepCount     int       // Total steps
    LastModified  time.Time // Last modified timestamp
    SizeMB        float64   // DB/log size in MB
    ModelName     string    // "Gemini 3.7 Flash"
    DBPath        string    // SQLite DB path
    LogPath       string    // transcript.jsonl path
}
```

---

### 2. Discovery Adapter (`internal/adapters/antigravity/discovery.go`)
* **Head Scan**: Parse first lines of `transcript.jsonl` for `InitialGoal`, `WorkspaceDir`, and `ModelName`.
* **Tail Scan**: Parse last lines of `transcript.jsonl` for `LastPrompt`.
* **Short Path**: Compute `ShortPath` using `formatShortPath(fullPath)`.

---

### 3. TUI Dual-Pane Rendering (`internal/ui/switcher.go`)
* **Pure Visual Tabs**: Rendered without any `Tabs:` text label. The active tab has a highlighted background (e.g. `[Antigravity (6)]`) and inactive tabs have muted styling (`Claude Code (0)`).
* **Left Pane**: Clean 2-line cards with `ShortPath`, `#hash`, `steps`, `size`, `time`, and `[ACTIVE]`.
* **Right Pane**: Strictly two blocks:
  - `[INITIAL GOAL / FIRST PROMPT]`
  - `[LATEST PROGRESS / LAST ACTION]`
* **Keybindings**:
  - `[` / `]`: Cycle Agent Tab
  - `Ctrl+j/k` or `Up/Down`: Navigate list
  - `Enter`: Attach session
  - `Esc` or `Ctrl+p`: Close

---

## 4. Verification Plan

### Automated Tests
```bash
go test -v ./internal/adapters/antigravity/...
go test -v ./internal/ui/...
```

### Manual Verification
1. Launch `agent-observer` and press `Ctrl+p`.
2. Verify pure visual tabs on the top bar (no `Tabs:` label).
3. Verify left list shows clean `self/ithome2026 (#aa726359)` with steps and time.
4. Verify right inspector displays only `[INITIAL GOAL]` and `[LATEST PROGRESS]`.
5. Verify `[` / `]` switches Agent Tabs seamlessly.
