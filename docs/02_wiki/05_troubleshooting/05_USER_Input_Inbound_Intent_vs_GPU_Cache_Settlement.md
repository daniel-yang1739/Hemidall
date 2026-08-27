---
title: 實戰排查：USER 步驟誤標 MISS 與時序結算錯位 (User Input Intent vs GPU Cache Settlement)
type: troubleshooting
created: 2026-08-27
updated: 2026-08-27
status: completed
tags: [troubleshooting, prompt-caching, user-input, telemetry, billing-cycle, bug-fix, runbook]
aliases: [User Step Fake Miss, Inbound Intent Timing, 意圖輸入與雲端結算解耦, Prompt Staged Telemetry]
---

# 🛡️ 實戰排查：USER 步驟誤標 MISS 與時序結算錯位 (User Input Intent vs GPU Cache Settlement)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 在排查使用者輸入步驟時，開發者發現 TUI 清單中 `USER_INPUT` 竟然被標記為紅色的 `[MISS]`，且遙測面板顯示 `Cached: 0 | New: 162,713`，引發「使用者輸入每次都造成數十萬快取全毀」的恐慌與誤解。
> 真相溯源證實：**人類在鍵盤上打字輸入意圖（Intent）的當下，雲端 GPU 根本尚未啟動推論，自然不可能在該瞬間計算出快取命中率**。真正的 Google 官方帳單與 199k (90.57%) 的快取命中，是在下一個步驟（`TOOL_CALL` / `MODEL_RESPONSE`）中結算。
> 透過將 `USER_INPUT` 的語意定義為 **「暫存待結算意圖 (Staged Intent)」** 並移除虛假的 `[MISS]` 標籤，徹底還原了計費時序的真實物理鏈路！

---

## 🔗 對應核心架構概念
* [[02_architecture/03_Agent_Storage_and_State_Machine|工業級 Agent 儲存架構與狀態機]]：Universal 4 態 FSM 與 Scope 分類。
* [[02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting|雙軌遙測引擎與倒推滑動窗口]]：官方 Protobuf 遙測結算週期。
* [[02_architecture/09_History_Explorer_and_Causality_Graph|歷史步進瀏覽器與因果拓撲圖譜]]：因果括號封裝與步驟語意。

---

## 🔍 一、現象與問題定義 (Symptom & Problem Definition)

### 🚨 異常現象描述：
在 TUI 歷史瀏覽器中，使用者輸入步驟（`[4433] 👤 USER`）右側赫然顯示紅色的 `[MISS]` 標籤。點入查看右側狀態遙測面板時，顯示：
* `• Prefix Cache Hit : 0 Tokens (0.0%) [CACHE MISS / BROKEN]`
* `• New Input Tokens : 162,713 Tokens`

工程師因此產生嚴重焦慮：「為什麼我的每一句對話，都會把前面幾十萬 Token 的快取全部打穿？」

```text
╭────────────────────────────────────╮╭────────────────────────────────────────────────────────────────╮
│ STEPS (6054) <                     ││ STEP TELEMETRY & METRICS                                       │
│ Filters: [T:All] [C:All]           ││   • Type                  : 👤 USER_INPUT                         │
│   ...                              ││   • Active Context        : 162,713 Tokens                         │
│ ┌[4440] 🤖 MODEL [HIT 100%]        ││   • Prefix Cache Hit      : 0 Tokens (  0.0%)  [CACHE MISS]        │
│ └[4433] 👤 USER [MISS]             ││   • New Input Tokens      : 162,713 Tokens                         │
╰────────────────────────────────────╯╰────────────────────────────────────────────────────────────────╯
```

---

## 🔬 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. 深入 SQLite 底層真相調查：
直接查詢真實的 SQLite 資料庫 `conversations/<id>.db`：

```bash
sqlite3 ~/.gemini/antigravity-cli/brain/aa726359/conversations/aa726359.db \
  "SELECT idx, length(data) FROM gen_metadata WHERE idx IN (4433, 4434, 4440);"
```
* **查詢結果**：
  * Step #4433 (`USER_INPUT`)：**在 `gen_metadata` 表中根本沒有資料 (0 Rows)**！
  * Step #4440 (`MODEL_RESPONSE`)：`gen_metadata` 包含 680KB 的 Protobuf BLOB，官方真實結算為 `Cached: 199,280 (90.57%)`！

### 2. 根因分析 (Temporal Cause & Effect Disconnection)：
* **物理事實**：`USER_INPUT` 發生於使用者終端機按下 Enter 鍵的瞬間（$T_0$）。此時尚未發起任何網路請求，雲端 GPU 尚未收到封包；
* **代碼漏洞**：先前的解析代碼在找不到 `gen_metadata` 時，自動將 `CachedTokens` 預設為 `0`，並將累積上下文（16 萬 Token）全數歸入 `NewTokens`，進而觸發了 `formatShortCache()` 的合成 `[MISS]` 標籤！

---

## 🛠️ 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. UI 列表層語意修復 (`formatShortCache`)：
將本地步驟（`ScopeLocalExecution`）與人類輸入步驟（`USER_INPUT`）顯式排除在快取標籤渲染之外：

```go
func formatShortCache(e core.UnifiedAgentEvent) string {
    // USER_INPUT 與本地執行步驟不具備獨立的雲端快取結算，不顯示合成 MISS
    if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution || e.Type == core.StepTypeUserInput {
        return ""
    }
    // 僅對雲端推論步驟渲染 [HIT 100%] / [PART 85%] / [EXPIRED] / [MISS]
    ...
}
```

### 2. 遙測面板語意對齊 (`buildTelemetryPanelLines`)：
對於 `USER_INPUT`，將誤導性的 `Cached: 0 | New: 162713` 替換為真實的意圖暫存語意：

```go
if e.Type == core.StepTypeUserInput {
    lines = append(lines,
        fmt.Sprintf("  • %-22s: %s", "Inbound Prompt", fmt.Sprintf("~%d Prompt Tokens", e.Tokens.ActiveTurnTokens)),
        fmt.Sprintf("  • %-22s: %s", "GPU Cache Billing", "Staged (Settled in next Cloud Step)"),
    )
}
```

---

## 📋 四、總結、抗體防禦與 Runbook SOP (Diagnostic Runbook)

### 💡 核心收穫與通用設計抗體：
* **因果時序與計費邊界鐵律**：在分散式 Agent 觀測中，必須嚴格區分 **本地意圖產生 (Event Ingestion)** 與 **雲端推論結算 (Inference Settlement)**，嚴禁將尚未發生的雲端狀態強行投影至本地輸入節點上！

### 🩺 1 分鐘快速診斷 Runbook SOP：
1. **驗證 `USER_INPUT` 步驟無虛假 MISS 標籤**：
   * 進入 `[2] History`，按 `t` 切換至 `[T:User]`，確認所有 `👤 USER` 步驟右側保持乾淨，無任何 `[MISS]` 標記；
2. **驗證右側遙測面板正確展示暫存語意**：
   * 選取任一 `USER` 步驟，確認 Telemetry 面板清晰顯示 `• Payload: ~N Prompt Tokens` 與 `• GPU Cache Billing: Staged ➔ Settled in Step #N+1`。
