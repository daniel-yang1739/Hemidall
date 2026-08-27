# 🔬 Agent-Observer 步驟因果關聯狀態機與 `p`/`n` 階層跳轉完整實作計畫 (Step Causality & Parent-Child Navigation Plan)

> **版本**：v5.0 (包含 Universal 4 Scopes、確定性狀態機關聯、Inspector 雙軌管線可視化、`p`/`n` 階層跳轉與完整測試計畫)  
> **關聯文件**：`docs/01_raw/2026-08-27_130909_step_causality_linkage_and_parent_child_navigation_plan.md`

---

## 🧭 一、 目標與功能全貌 (Goal & Architectural Overview)

本計畫旨在將 `agent-observer` 的步驟觀測從「無因果的扁平日誌傾倒」升級為 **「具備父子因果鏈條 (Causality Chain) 與自解釋雙軌管線」** 的專業 Agent 觀測中樞：

1. **通用四分法範疇 (Universal 4 Scopes)**：
   * `[👤 USER]` (使用者意圖)
   * `[⚙️ SYSTEM]` (系統開局/歷史)
   * `[☁️ CLOUD]` (雲端推理/決策/計費)
   * `[💻 LOCAL]` (本機離線執行/0元暫存)
2. **確定性狀態機關聯 (Deterministic FSM Linkage)**：
   * $O(N)$ 線性單次遍歷，自動建立 `ParentStepIdx`、`PackagedInStepIdx` 與 `ConsumedStepIndices`。
3. **即時 (LIVE) 與歷史 (Playback) 雙模態**：
   * LIVE 當下：顯示 `Staged (~50 tok, Pending Next Turn ⏳)`。
   * 結算後：顯示 `Packaged in Step #3563 (+50 Official tok ✅)`。
4. **焦點在右側欄 (Inspector) 時支援 `p` / `n` 階層極速跳轉**：
   * **按 `p` (Parent)**：直接跳轉至當前步驟的父層（例如從 `RUN_COMMAND` 跳回觸發它的 `TOOL_CALL`，或從 `TOOL_CALL` 跳回初始 `USER_INPUT`）；
   * **按 `n` (Next / Child / Consumed)**：直接跳轉至當前步驟的子層（例如從 `TOOL_CALL` 跳至執行的 `RUN_COMMAND`，或從 `RUN_COMMAND` 跳至打包計費它的 `PLANNER_RESPONSE`）。

---

## 🎨 二、 STEP INSPECTOR 雙軌渲染效果

### 💻 案例 A：檢視本地執行步驟 (例如 Step 3562 `RUN_COMMAND`)
```text
╭──────────────────────────────────────────────────────────────────────────────╮
│ STEP INSPECTOR                                                               │
│ • Step 3562 (DONE) at 11:11:23 | 💻 LOCAL EXECUTION STEP                     │
│ • Origin : Local Process (zsh on macOS)                                      │
│ • Parent : Triggered by Model Tool Call in Step #3561         [p] Jump       │
│ • Billing: Offline (0 tok) ➔ Packaged in Step #3563 (+50 tok) [n] Jump       │
│ ──────────────────────────────────────────────────────────────────────────── │
│ Terminal Output (stdout/stderr):                                             │
│ Created At: 2026-08-27T11:11:20+08:00                                        │
│ PASS: TestDiscoverAllSessions (0.70s)                                        │
╰──────────────────────────────────────────────────────────────────────────────╯
```

### ☁️ 案例 B：檢視雲端生成步驟 (例如 Step 3563 `TOOL_CALL`)
```text
╭──────────────────────────────────────────────────────────────────────────────╮
│ STEP INSPECTOR                                                               │
│ • Step 3563 (DONE) at 11:11:24 | ☁️ CLOUD INFERENCE TURN                     │
│ • Model  : Gemini 3.7 Flash (High) (Google Official Billing Telemetry)       │
│ • Parent : Triggered by User Request in Step #3560            [p] Jump       │
│ • Input  : Consumed Local Step #3562 (+50 New Tokens)         [n] Jump       │
│ • Tokens : Total: 210,900 | Cached: 210,850 (100.0%) | New: 50              │
│ • 5-Dims : Sys=3806 | Tools=1377 | Res=50 | Hist=205667 | Act=0              │
│ ──────────────────────────────────────────────────────────────────────────── │
│ Model Tool Call: run_command                                                 │
│ Args: {"CommandLine": "go build -o bin/agent-observer main.go ..."}          │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## 🛠️ 三、 模組變更與技術實作細節

### 1. `internal/core/types.go` [MODIFY]
```go
// StepScope defines universal origin of agent events
type StepScope string

const (
    ScopeUserInteraction StepScope = "USER"   // 👤 User Intent / Prompts
    ScopeCloudInference  StepScope = "CLOUD"  // ☁️ Cloud LLM Inference / Decisions
    ScopeLocalExecution  StepScope = "LOCAL"  // 💻 Local Machine Process (Offline)
    ScopeSystemBootstrap StepScope = "SYSTEM" // ⚙️ System Init / History
)

// Add fields to UnifiedAgentEvent:
type UnifiedAgentEvent struct {
    // ... existing fields ...
    Scope               StepScope `json:"scope"`
    ParentStepIdx       int       `json:"parent_step_idx,omitempty"`
    PackagedInStepIdx   int       `json:"packaged_in_step_idx,omitempty"`
    ConsumedStepIndices []int     `json:"consumed_step_indices,omitempty"`
}

func (e UnifiedAgentEvent) IsLocalStep() bool {
    return e.Type == StepTypeRunCommand || e.Type == StepTypeViewFile ||
           e.Type == StepTypeCodeAction || e.Type == StepTypeListDirectory ||
           e.Type == StepTypeToolResult || e.Type == StepTypeAskQuestion
}
```

---

### 2. `internal/core/state_machine.go` [NEW]
實現通用單執行緒狀態機 `StepLinkageTracker`：
```go
package core

type StepLinkageTracker struct {
    lastUserPromptIdx     int
    lastParentToolCallIdx int
    pendingStagedSteps    []int
}

func NewStepLinkageTracker() *StepLinkageTracker {
    return &StepLinkageTracker{}
}

func (t *StepLinkageTracker) ProcessEvent(e *UnifiedAgentEvent) {
    if e.Type == StepTypeUserInput {
        e.Scope = ScopeUserInteraction
        t.lastUserPromptIdx = e.StepIndex
        t.lastParentToolCallIdx = 0
        t.pendingStagedSteps = nil
    } else if e.IsLocalStep() {
        e.Scope = ScopeLocalExecution
        e.ParentStepIdx = t.lastParentToolCallIdx
        t.pendingStagedSteps = append(t.pendingStagedSteps, e.StepIndex)
    } else if e.Type == StepTypeToolCall {
        e.Scope = ScopeCloudInference
        e.ParentStepIdx = t.lastUserPromptIdx
        e.ConsumedStepIndices = t.pendingStagedSteps
        t.lastParentToolCallIdx = e.StepIndex
        t.pendingStagedSteps = nil
    } else if e.Type == StepTypeModelResponse {
        e.Scope = ScopeCloudInference
        e.ParentStepIdx = t.lastUserPromptIdx
        e.ConsumedStepIndices = t.pendingStagedSteps
        t.lastParentToolCallIdx = 0
        t.pendingStagedSteps = nil
    } else {
        e.Scope = ScopeSystemBootstrap
    }
}
```

---

### 3. `internal/adapters/antigravity/` [MODIFY]
* 在 `watcher.go` (即時串流) 與 `discovery.go` (歷史載入)：
  * 實例化 `StepLinkageTracker`，處理每一筆事件；
  * 批次載入歷史時，呼叫狀態機進行反向回填（Backfill），將 `PackagedInStepIdx` 正確賦予對應的 Local Step。

---

### 4. `internal/ui/model.go` & `views.go` [MODIFY]
* **`p` / `n` 鍵盤導航處理 (`focusPane == FocusDetail`)**：
  ```go
  case "p":
      if m.focusPane == FocusDetail {
          if curr, ok := m.getSelectedEvent(); ok && curr.ParentStepIdx > 0 {
              m.jumpToStep(curr.ParentStepIdx)
          }
      }
  case "n":
      if m.focusPane == FocusDetail {
          if curr, ok := m.getSelectedEvent(); ok {
              if len(curr.ConsumedStepIndices) > 0 {
                  m.jumpToStep(curr.ConsumedStepIndices[0])
              } else if curr.PackagedInStepIdx > 0 {
                  m.jumpToStep(curr.PackagedInStepIdx)
              }
          }
      }
  ```
* **`jumpToStep(targetIdx int)` 智能視圖校準**：
  * 在 `m.getFilteredHistory()` 中尋找目標 Step Index；
  * 更新 `m.selectedIdx`、重置 `m.detailScroll = 0`；
  * 動態微調 `m.historyOffset`，保證左欄選取框完整顯露在螢幕可視區內。
* **Inspector 面板渲染**：
  * 輸出彩色 Scope 徽章與 `[p] Jump` / `[n] Jump` 提示字樣。
* **快捷鍵說明 (`shortcuts.go`)**：
  * 在 `?` 視窗中增加 `p` 與 `n` 快捷鍵說明。

---

## 🧪 四、 驗證計畫 (Verification Plan)

### 1. Automated Tests
* `TestUniversalFSMAndLinkageTracker`：
  * 構造 `USER_INPUT` $\to$ `TOOL_CALL` $\to$ `RUN_COMMAND` $\to$ `VIEW_FILE` $\to$ `MODEL_RESPONSE` 完整時序；
  * 驗證 `RUN_COMMAND.ParentStepIdx == TOOL_CALL.StepIndex`；
  * 驗證 `VIEW_FILE.ParentStepIdx == TOOL_CALL.StepIndex`；
  * 驗證 `MODEL_RESPONSE.ConsumedStepIndices == [RUN_COMMAND, VIEW_FILE]`；
  * 驗證 `RUN_COMMAND.PackagedInStepIdx == MODEL_RESPONSE.StepIndex`。
* `TestInspectorParentChildJumpNavigation`：
  * 測試在右欄按 `p` 成功跳轉至 Parent Step；
  * 測試在右欄按 `n` 成功跳轉至 Child / PackagedIn Step。
* 執行完整測試：
  ```bash
  go test -v ./...
  ```

### 2. Manual Verification
* 啟動 `agent-observer`，進入 History View：
  * 按 `l` 切換至 Inspector；
  * 在 Local Step 上按 `p`，左欄瞬間跳轉並選取其父層 `TOOL_CALL`；
  * 在 `TOOL_CALL` 上按 `n`，左欄瞬間跳轉並選取其子層 `RUN_COMMAND`；
  * 確認無任何高度抖動，且文字說明清晰易懂。
