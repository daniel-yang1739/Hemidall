---
title: 事件計數 10,336 與步驟序號 5,919 脫節之謎：串流狀態躍遷重複累加問題
type: troubleshooting
created: 2026-08-28
updated: 2026-08-28
status: completed
tags: [troubleshooting, deduplication, streaming, bubbletea, tui, sqlite]
aliases: [Stream_Duplication_Incident, Step_Counter_Inflation_Bug]
---

# 事件計數 10,336 與步驟序號 5,919 脫節之謎：串流狀態躍遷重複累加問題

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**  
> * **異常現象**：Observer 右上角計數器膨脹至 `Events: 10639`，但列表中的最新步驟序號卻只有 `Step #5919`，兩者數字嚴重失調。
> * **根本根因**：Google CLI 在執行步驟時採用串流寫入（開始時發送 `RUNNING`，結束時追加 `DONE`）。舊版 Observer 在 Bubbletea 事件接收循環中無條件執行 `append`，導致同一個步驟被重複記錄兩次。
> * **架構修復**：實裝「**步驟序號唯一性鎖定與原地覆蓋更新 (In-Place Deduplication)**」，收到事件時先以 `(SessionID, StepIndex)` 查找，若存在則原地更新狀態與 Payload；並將 Header 統一更名為 `Steps: 59xx`。
> * **雙向鏈接**：對應架構概念參見 [[03_Agent_Storage_and_State_Machine]] 與 [[07_TUI_Engine_and_Terminal_Layout_Mechanics]]。

---

## 一、現象與問題定義 (Symptom & Problem Definition)

### 1. 觸發情境與異常現象
在長時間運行觀測時，隨著使用者與 Agent 持續互動，右上角的事件計數器呈現瘋狂膨脹（從 5,800 一路飆升至 10,336 乃至 10,639）；然而切換至 History View 時，底下的步驟清單最新序號卻僅為 `#5919`。

### 2. 使用者困惑與概念矛盾
使用者無法理解：「*這代表有 1 萬多個步驟，但系統遺失了 4,000 多步？還是一個步驟被重複計算了？*」

---

## 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. Step（步驟實體）vs Event（狀態事件）的本質差異
* **Step**：Agent 狀態機前進的原子實體，全局唯一（Primary Key `idx`）；
* **Event**：Step 生命週期中的狀態跳變通知。

### 2. 出錯代碼路徑溯源 (`internal/ui/model.go`)
```go
// ❌ 錯誤的舊版代碼：盲目 append
case AgentEventMsg:
    event := core.UnifiedAgentEvent(msg)
    m.latestEvent = event
    m.eventCount++ // 盲目累加
    m.history = append(m.history, event) // 致命盲點：同一個 StepIndex 被塞了多次
```

### 3. 日誌與記憶體行為比對
* 當 Step #500 啟動時：Watcher 讀取到第一行 JSON（`status: RUNNING`），發送 `AgentEventMsg` $\to$ `m.history` 增加 1 筆；
* 當 Step #500 完成時：Watcher 讀取到第二行 JSON（`status: DONE`），發送 `AgentEventMsg` $\to$ `m.history` 又增加 1 筆；
* 👉 **結果：5,700 個實體步驟，產生了 $5700 \times 2 \approx 10,639$ 次串流通知，造成記憶體虛胖！**

---

## 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. 核心修復：In-Place 原地覆蓋更新機制
在 `model.go` 接收 `AgentEventMsg` 時，先向前掃描是否已有相同 `(StepIndex, SessionID)`：

```go
// 🟢 修復後的代碼 (internal/ui/model.go)
case AgentEventMsg:
    event := core.UnifiedAgentEvent(msg)
    m.latestEvent = event
    m.lastActivity = time.Now()

    // 檢查歷史紀錄中是否已存在此步驟序號（如 RUNNING 轉 DONE 或串流更新）
    existingIdx := -1
    for i := len(m.history) - 1; i >= 0; i-- {
        if m.history[i].StepIndex == event.StepIndex && m.history[i].SessionID == event.SessionID {
            existingIdx = i
            break
        }
    }

    if existingIdx >= 0 {
        m.history[existingIdx] = event // 🟢 原地覆蓋更新，長度不變！
    } else {
        m.history = append(m.history, event) // 全新步驟才往後追加
    }
```

### 2. UI 頂部標籤標準化
在 `renderHeader` 中，將易引發誤解的 `Events: %d` 徹底改為：
```go
stepsInfo := lipgloss.NewStyle().Foreground(ColorLightText).Render(fmt.Sprintf("Steps: %d", len(m.history)))
```

---

## 四、總結、抗體防禦與 Runbook SOP (Key Takeaways & Diagnostic SOP)

### 1. 通用設計準則 (Design Invariants)
* **狀態機的冪等性 (Idempotency)**：接收端必須以「實體主鍵 (Entity Primary Key)」為基礎維護記憶體狀態，而非以「訊息傳輸次數 (Message Arrival Frequency)」做無腦累加。

### 2. 1 分鐘快速診斷 Runbook SOP
1. **驗證記憶體與檔案對齊**：
   在 TUI 中查看右上角 `Steps: N`，比對左側列表頂部的最新序號 `Step #N`；
2. **驗證一致性**：兩者數字必須在扣除 0-indexed 偏移後 **100% 絕對相等**。
