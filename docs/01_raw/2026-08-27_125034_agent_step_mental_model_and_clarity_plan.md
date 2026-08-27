# 🔬 Agent-Observer 步驟心智模型與雙模態遙測完整企劃書 (Two-Tier State Machine & Step Clarity Plan)

> **版本**：v2.0 (包含完整歷史背景、現狀問題定義、核心需求、使用者提問 Q&A 與技術實作規劃)  
> **關聯文件**：`docs/01_raw/2026-08-27_125034_agent_step_mental_model_and_clarity_plan.md`

---

## 🧭 一、 現狀問題本質定義 (Problem Definition - 為什麼會感到混亂？)

使用者與開發者在目前視圖中感到混亂的核心原因，可歸納為以下三大架構與 UX 矛盾：

### 1. 概念斷層：單一扁平清單 vs. 雙層執行物理 (Linear Step vs. Two-Tier Physics)
* **底層真實**：Agent 存在兩種本質完全不同的步驟：
  1. ☁️ **雲端生成步驟 (Cloud Inference Turn)**：向 Google API 發送請求，消耗 GPU 算力、查詢 KV 快取、生成決策（擁有 `Model Name`、`Total Tokens`、`Cache Hit Rate`）。
  2. 💻 **本地執行步驟 (Local Execution Step)**：在使用者電腦終端實體執行命令或讀寫檔案（在本地離線完成，不消耗 GPU、無雲端 Model）。
* **現有痛點**：目前 UI 在左欄將它們一視同仁地扁平成 `Step 3561`、`Step 3562`、`Step 3563`，導致使用者誤以為每個 Step 都是獨立發送給雲端的請求，因而產生「Step 3562 是 RUN_COMMAND，它到底有沒有送給 Google？為什麼沒有 Model？」的困惑。

### 2. 遙測延遲歸屬未被可視化 (Delayed Telemetry Attribution)
* **底層真實**：本地步驟（如跑測試輸出的 50 tokens）在執行當下（Step 3562）是不計費的，它會在**緊接著的下一個雲端請求（Step 3563）**才被打包進 HTTP Payload 正式被 GPU 接收並計費。
* **現有痛點**：Step 3562 顯示 `Tokens: 0`，Step 3563 顯示 `Tokens: 210900 (New: 50)`，兩者之間缺少一條「這 50 個新 Token 就是來自前一步驟終端輸出」的**因果連結線**。

### 3. 工具呼叫與執行結果脫鉤 (Tool Call & Result Decoupling)
* **現有痛點**：模型發出工具呼叫（Step 3561: `TOOL_CALL`）與終端執行結果（Step 3562: `RUN_COMMAND`）被拆成兩張獨立卡片，缺乏父子階層與因果關聯。

---

## 📋 二、 核心需求定義 (Requirements Definition)

| 編號 | 需求項目 | 使用者故事 (User Story) | 驗收標準 (Acceptance Criteria) |
| :--- | :--- | :--- | :--- |
| **REQ-1** | **步驟本質清晰標註 (Step Origin Badging)** | 作為使用者，我想一眼看出這個步驟是「☁️ 雲端模型決策」還是「💻 本機離線執行」，不需要大腦記憶規則。 | Inspector 與清單清楚標示 `[CLOUD]` / `[LOCAL]`，本地步驟明確標註執行環境（如 `Local Zsh Process`）。 |
| **REQ-2** | **因果關係與 Token 去向追蹤 (Causality & Next-Turn Trace)** | 作為使用者，當我看著一筆本地工具執行結果時，我想知道它到底在「哪一步驟」被送出並計費。 | 在本地步驟的 Inspector 中，明確顯示：<br/>`📍 Pipeline: Executed locally ➔ Packaged & billed in Step #3563 (+50 New Tokens)`。 |
| **REQ-3** | **工具呼叫與結果階層關聯 (Tool Call & Result Linkage)** | 作為使用者，我想知道這個 `RUN_COMMAND` 是誰叫的，以及它的輸出結果是什麼。 | 支援在 Inspector 內互相跳轉或同時呈現 Tool Call 與對應的 Result。 |
| **REQ-4** | **零學習成本自解釋文字 (Self-Explanatory UI)** | 作為初次使用的工程師，我不應該需要查閱文檔才能理解 `Model: n/a` 或快取狀態。 | 每個欄位附帶簡短易懂的輔助說明。 |

---

## 🔬 三、 使用者關鍵疑問深度剖析 (Engineering Realities & Q&A)

### Q1: 在 3562 剛跑完、尚未發出 3563 的時候 (LIVE 模式)，會有那句話嗎？要怎麼知道 +50 Tokens？還是跑完回顧才會多出來？

#### 💡 真實解答：採用「雙態轉換機制 (Staged ➔ Billed State Machine)」
在 3562 剛執行的那個瞬間，未來的 3563 確實還不存在。但我們**不需要通靈，也不會造假**：

1. **在 LIVE 當下（3562 剛結束，尚未發送給雲端）**：
   * **本地 Tokenizer 即時算力**：雖然雲端帳單還沒下來，但我們本地的 Tokenizer 已即時分析了這筆輸出的文字長度（估算出 `~50 tokens`）；
   * **Inspector 即時顯示**：
     ```text
     • Billing: Offline (0 tok) ➔ Staged (~50 tok, Pending Next Cloud Turn ⏳)
     ```
   * 清楚告訴使用者：「這筆是本地執行，目前已暫存待發，預估將在下一輪消耗 ~50 tokens」。
2. **在發送完成後 / 歷史回溯（3563 完成並收到 Google API 回應）**：
   * 官方遙測（`gen_metadata`）正式寫入 SQLite，關聯自動結算錨定；
   * **Inspector 更新為官方確切數據**：
     ```text
     • Billing: Offline (0 tok) ➔ Packaged to Cloud in Step #3563 (+50 Official tok ✅)
     ```

---

### Q2: 方案二的 Group 要怎麼知道誰是誰的 Parent？底層有記錄嗎？複雜度與誤判率如何？

#### 💡 真實解答：底層是「嚴格單執行緒阻塞狀態機 (Deterministic ReAct FSM)」，誤判率趨近於 0%！

#### 1. 底層數據結構：
Antigravity（以及 Claude Code、Cursor）的底層執行引擎是嚴格的 **Turn-based ReAct 狀態機**：
$$\text{Step } N\text{ (Tool Call: run\_command)} \xrightarrow{\text{實體執行}} \text{Step } N+1\text{ (Tool Output)} \xrightarrow{\text{作為下一輪 Input}} \text{Step } N+2\text{ (Model Response)}$$

#### 2. 判定演算法 (Linear FSM - $O(N)$ 單次遍歷)：
只需在讀取事件流時維護輕量狀態指針（State Pointer）：
```go
type StepLinkageTracker struct {
    pendingCallStepIdx int   // 當前發出工具呼叫的 Step
    stagedLocalSteps   []int // 待送出的 Local Steps
}

func (t *StepLinkageTracker) ProcessEvent(e *core.UnifiedAgentEvent) {
    if e.Type == core.StepTypeToolCall {
        t.pendingCallStepIdx = e.StepIndex
        t.stagedLocalSteps = nil
        e.ExecutionScope = core.ScopeCloudInference
    } else if e.IsLocalStep() {
        e.ExecutionScope = core.ScopeLocalExecution
        e.ParentStepIdx = t.pendingCallStepIdx // 錨定父層 Tool Call
        t.stagedLocalSteps = append(t.stagedLocalSteps, e.StepIndex)
    } else if e.Type == core.StepTypeModelResponse || e.Type == core.StepTypeToolCall {
        e.ExecutionScope = core.ScopeCloudInference
        e.ConsumedStepIndices = t.stagedLocalSteps // 錨定消耗了哪些 Local Steps
        t.stagedLocalSteps = nil
    }
}
```

#### 3. 複雜度與誤判率：
* **時間與空間複雜度**：$O(1)$ 額外空間、$O(N)$ 單次線性遍歷，完全零效能負擔。
* **誤判率**：**幾乎為 0% (Near Zero)**。因為 CLI Agent 是**單執行緒循序阻塞（Sequential Blocking）**運作，不存在跨 Session 交叉或併發亂序競爭。

---

## 🎨 四、 介面排版實況示意 (UI Mockups)

### 💻 案例 A：檢視本地執行步驟 (例如 Step 3562 `RUN_COMMAND`)
```text
╭──────────────────────────────────────────────────────────────────────────────╮
│ STEP INSPECTOR                                                               │
│ • Step 3562 (DONE) at 11:11:23 | 💻 LOCAL EXECUTION STEP                     │
│ • Origin : Local Terminal (zsh process on macOS)                             │
│ • Billing: Offline (0 tok) ➔ Packaged to Cloud in Step #3563 (+50 New Tokens)│
│ • Parent : Triggered by Model Tool Call in Step #3561                         │
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
│ • Tokens : Total: 210,900 | Cached: 210,850 (100.0%) | New: 50 (from #3562) │
│ • 5-Dims : Sys=3806 | Tools=1377 | Res=50 | Hist=205667 | Act=0              │
│ ──────────────────────────────────────────────────────────────────────────── │
│ Model Tool Call: run_command                                                 │
│ Args: {"CommandLine": "go build -o bin/agent-observer main.go ..."}          │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## 🛠️ 五、 檔案變更與實作架構 (Proposed Code Changes)

### 1. `internal/core/types.go` [MODIFY]
* 在 `UnifiedAgentEvent` 新增：
  * `ExecutionScope string` (`"CLOUD"` 或 `"LOCAL"`)
  * `ParentStepIdx int` (記錄由哪一步 Tool Call 觸發)
  * `PackagedInStepIdx int` (記錄在哪一步被雲端打包計費)
  * `ConsumedStepIndices []int` (雲端步驟記錄消耗了哪些本地結果)
* 新增輔助方法 `IsLocalStep() bool`。

### 2. `internal/adapters/antigravity/watcher.go` & `discovery.go` [MODIFY]
* 內建 `StepLinkageTracker`，在解析歷史與即時 Stream 時自動關聯 `ParentStepIdx`、`PackagedInStepIdx` 與 `ConsumedStepIndices`。

### 3. `internal/ui/model.go` & `views.go` [MODIFY]
* **Inspector 面板**：依據 `ExecutionScope` 動態渲染雙軌資訊（`CLOUD INFERENCE` vs `LOCAL EXECUTION`，並呈現 `Staged ⏳` / `Packaged in #XXX ✅`）。
* **左欄清單**：本地步驟標註 `(Local)` 或樹狀前綴。

---

## 🧪 六、 驗證計畫 (Verification Plan)

### 1. Automated Tests
* 新增單元測試 `TestStepLinkageAndCausalityTracker`：
  * 測試包含 `USER_INPUT` $\to$ `TOOL_CALL` $\to$ `RUN_COMMAND` $\to$ `PLANNER_RESPONSE` 的完整時序鏈。
  * 斷言 `RUN_COMMAND.ParentStepIdx` 與 `PLANNER_RESPONSE.ConsumedStepIndices` 100% 正確匹配。
* 執行完整測試套件：
  ```bash
  go test -v ./...
  ```

### 2. Manual Verification
* 啟動 `agent-observer`，進入 History Explorer：
  * 選取 `TOOL_CALL` 步驟，確認顯示 `☁️ CLOUD INFERENCE TURN` 與關聯的 Consumed Steps；
  * 選取 `RUN_COMMAND` / `VIEW_FILE` 步驟，確認顯示 `💻 LOCAL EXECUTION STEP` 與 `Billing: ... ➔ Packaged in #...`；
  * 進行即時操作，確認在 Local Step 產生當下顯示 `Staged (~X tok, Pending ⏳)`，雲端返回後自動轉為 `Packaged (+X tok ✅)`。
