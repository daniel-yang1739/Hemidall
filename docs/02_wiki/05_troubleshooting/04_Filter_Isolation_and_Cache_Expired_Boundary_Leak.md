---
title: 實戰排查：EXPIRED 步驟洩漏至 MISS 過濾結果之邊界漏洞 (Filter Boundary Leak Postmortem)
type: troubleshooting
created: 2026-08-27
updated: 2026-08-27
status: completed
tags: [troubleshooting, prompt-caching, filter-isolation, bug-fix, runbook, boundary-condition]
aliases: [Filter Isolation Leak, 快取過期混入未命中, Cache Status Boundary, 互斥過濾守衛]
---

# 🛡️ 實戰排查：EXPIRED 步驟洩漏至 MISS 過濾結果之邊界漏洞 (Filter Boundary Leak Postmortem)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 在使用 `agent-observer` 的歷史步驟過濾器切換至 `[C:Miss]` 時，工程師驚訝地發現黃色的 `[EXPIRED]` 步驟竟然夾雜在紅色 `[MISS]` 清單中。
> 經過代碼溯源，根本原因在於 `matchCacheFilter()` 早期判定邏輯中，將「快取命中率為 0 且非寫入步驟」無條件落入 Fallback 判定，而未對 `CacheStatus == "EXPIRED"` 與 `"TTL_EXPIRED"` 設立嚴格的先制排除守衛。
> 透過導入 **顯式互斥守衛 (Strict Exclusion Guard)**，徹底保證了 `[C:Miss]` 與 `[C:Expired]` 兩大快取生命週期狀態的 100% 隔離！

---

## 🔗 對應核心架構概念
* [[02_architecture/09_History_Explorer_and_Causality_Graph|歷史步進瀏覽器與因果拓撲圖譜]]：雙軌過濾引擎架構與篩選邊界。
* [[01_theory/03_Prompt_Caching_Lifecycle|Prompt Caching 生命週期與 5 分鐘 TTL]]：雲端 GPU 顯存閒置淘汰機制。
* [[02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting|雙軌遙測引擎與倒推滑動窗口]]：快取狀態標記標準。

---

## 🔍 一、現象與問題定義 (Symptom & Problem Definition)

### 🚨 異常現象描述：
在 TUI 的歷史瀏覽器中，按下 `c` 鍵循環切換至 `[C:Miss]`（預期僅查看因冷啟動或前綴被破壞而發生未命中的步驟），清單中卻出現了帶有黃色標籤的 `[EXPIRED]` 步驟：

```text
╭────────────────────────────────────╮
│ STEPS (42) <                       │
│ Filters: [T:All] [C:Miss]          │
│   ...                              │
│ ┌[2302] 🛠️ TOOL [EXPIRED]          │  <-- 🚨 異常！黃色 EXPIRED 混入紅色的 MISS 過濾清單中
│ │  Model: Gemini 3.7 Flash         │
│ │[2301] 💻 OUTPUT (Local)         │
│ └  Tool: edit_file                 │
│ ┌[2180] 🤖 MODEL [MISS]            │  <-- 正常 MISS 步驟
│ └[2179] 👤 USER                   │
╰────────────────────────────────────╯
```

### 🎯 復現路徑：
1. 連續對話超過 5 分鐘，觸發 Google GPU 雲端顯存的 5-min TTL 淘汰，產生標記為 `[EXPIRED]` 的步驟；
2. 啟動 `agent-observer` 並進入 `[2] History` 視圖；
3. 按 `c` 切換快取過濾器至 `[C:Miss]`，觀察清單內容。

---

## 🔬 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. 排除的錯誤假設：
* **假設 A：底層 SQLite Protobuf 寫入錯誤** $\to$ **排除**。檢查 SQLite 原生二進制資料，`CacheStatus` 正確標記為 `EXPIRED`。
* **假設 B：UI 顏色渲染函式誤判** $\to$ **排除**。`formatShortCache()` 正確將 `EXPIRED` 上色為黃色、`MISS` 上色為紅色。

### 2. 具體出錯代碼溯源 (`agent-observer/internal/ui/model.go`)：
檢查過濾器比對函式 `matchCacheFilter`：

```go
// 🐛 出錯前的漏洞代碼：
case CacheFilterMiss:
    if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
        return false
    }
    // ⚠️ 漏洞點：僅排除了 "WRITE"，但未排除 "EXPIRED" 與 "TTL_EXPIRED"！
    if e.CacheStatus == "WRITE" {
        return false
    }
    // 由於 EXPIRED 步驟的 CachedTokens == 0，條件命中，導致 EXPIRED 被當作 MISS 放行！
    return e.CacheStatus == "MISS" || (e.Tokens.TotalTokens > 0 && e.Tokens.CachedTokens == 0 && e.Tokens.NewTokens > 0)
```

當一個步驟因 5 分鐘閒置過期時，其 `Tokens.CachedTokens == 0` 且 `Tokens.NewTokens > 0`。上述邏輯在 `CacheStatus != "WRITE"` 的條件下直接命中 Fallback，導致過期步驟洩漏進 `[C:Miss]`！

---

## 🛠️ 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. 核心修復代碼：
在 `CacheFilterMiss` 導入嚴格的顯式互斥排除守衛：

```go
case CacheFilterMiss:
    if e.IsLocalStep() || e.Scope == core.ScopeLocalExecution {
        return false
    }
    // 🛡️ 修復：顯式排除 EXPIRED, TTL_EXPIRED 與 WRITE
    if e.CacheStatus == "EXPIRED" || e.CacheStatus == "TTL_EXPIRED" || e.CacheStatus == "WRITE" {
        return false
    }
    return e.CacheStatus == "MISS" || (e.Tokens.TotalTokens > 0 && e.Tokens.CachedTokens == 0 && e.Tokens.NewTokens > 0)
```

### 2. 不變量防護 (Invariant Assurance)：
* `[C:Miss]` $\cap$ `[C:Expired]` $\equiv \emptyset$（兩者交集嚴格為空集合）；
* `[C:Miss]` 專注反映 **前綴結構破壞或首輪冷啟動**；
* `[C:Expired]` 專注反映 **閒置超過 5 分鐘造成的時間淘汰**。

---

## 📋 四、總結、抗體防禦與 Runbook SOP (Diagnostic Runbook)

### 💡 核心收穫與通用設計抗體：
* **多狀態機枚舉過濾鐵律**：在枚舉型狀態過濾（如 Cache 狀態、訂單狀態、任務生命週期）中，**嚴禁使用寛鬆的 Fallback 兜底條件**，所有非目標狀態必須在前置 Guard 階段顯式攔截！

### 🩺 1 分鐘快速診斷 Runbook SOP：
1. **執行單元測試檢查過濾純度**：
   ```bash
   go test -v ./internal/ui -run TestHistoryFilteringAndStepSearch
   ```
2. **驗證日誌中的快取狀態分佈**：
   ```bash
   sqlite3 ~/.gemini/antigravity-cli/brain/<session-id>/conversations/<session-id>.db \
     "SELECT idx, length(data) FROM gen_metadata LIMIT 10;"
   ```
3. **終端機驗證**：進入 `agent-observer`，按 `c` 循環切換 `[C:Expired]` 與 `[C:Miss]`，確認兩者條目無重疊。
