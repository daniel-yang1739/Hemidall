---
title: 歷史搜尋 Context 丟失之謎：破壞式過濾搜尋到 Vim 式非破壞跳轉導航重構
type: troubleshooting
created: 2026-08-28
updated: 2026-08-28
status: completed
tags: [troubleshooting, tui, vim-navigation, search, ux, bubbletea]
aliases: [History_Search_Context_Loss, Jump_To_Step_Reconstruction]
---

# 歷史搜尋 Context 丟失之謎：破壞式過濾搜尋到 Vim 式非破壞跳轉導航重構

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**  
> * **異常現象**：在 History View 按下 `/` 搜尋步驟時，每輸入一個字列表就被即時過濾裁切，導致前後上下文（父步驟、工具呼叫順序）全部消失；按下 Enter 後只剩單行符合項，無法繼續按 `j`/`k` 上下瀏覽。
> * **根本根因**：搜尋邏輯採用了「破壞式 Filter 切片機制（Destructive Filtering）」，將搜尋輸入直接作為列表過濾條件，破壞了步驟陣列的連續性。
> * **架構修復**：徹底重構為 **Vim / Pager 式 Jump-to-Step 導航器**：輸入期間完整保留所有步驟，按 Enter 後計算偏移量並平滑滾動游標至目標步驟，找不到時以 `❌ Step '#9999' not found` 紅色橫幅提示；跳轉後可立即按 `j`/`k` 上下瀏覽完整上下文。
> * **雙向鏈接**：對應架構概念參見 [[07_TUI_Engine_and_Terminal_Layout_Mechanics]] 與 [[09_History_Explorer_and_Causality_Graph]]。

---

## 一、現象與問題定義 (Symptom & Problem Definition)

### 1. 觸發情境與異常現象
使用者在有數千個步驟的歷史清單中按下 `/` 嘗試搜尋 `Step #5518`：
* 輸入 `5` $\to$ 列表瞬間被裁切，只留下所有包含數字 5 的步驟；
* 輸入 `5518` $\to$ 列表只剩下一行；
* 按下 `Enter` $\to$ 使用者想看 Step #5517 與 #5519 的因果關係，但按 `j`/`k` 完全動彈不得，必須按 `Esc` 清除搜尋才能看見其他步驟，但按 `Esc` 後游標又重置跑回頂部。

### 2. 人機互動矛盾
搜尋的本質目的是「**跳轉定位以查看上下文**」，而非「**隔離並隱藏其他步驟**」。破壞式過濾完全違反了程式碼審查與時序追蹤的直覺。

---

## 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. 出錯代碼路徑溯源 (`internal/ui/model.go`)
```go
// ❌ 錯誤的舊版代碼：將搜尋字串直接套用於列表過濾
func (m Model) getFilteredHistory() []core.UnifiedAgentEvent {
    // ...
    if m.historyStepQuery != "" {
        // 致命盲點：粗暴過濾切片，破壞前後步驟連續性
        var matched []core.UnifiedAgentEvent
        for _, e := range filtered {
            if strings.Contains(fmt.Sprintf("%d", e.StepIndex), m.historyStepQuery) {
                matched = append(matched, e)
            }
        }
        return matched // 返回只剩少數項的殘缺切片
    }
}
```

---

## 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. 核心修復：非破壞式 Jump 導航引擎
在 [`model.go`](file:///Users/daniel_y_yang/Documents/self/ithome2026/agent-observer/internal/ui/model.go#L1060-L1100) 中：
1. 從 `getFilteredHistory` 中徹底移除 `historyStepQuery`，保證列表永遠完整；
2. 當使用者在 Jump 輸入框按 Enter 時，精確計算索引並平滑錨定：

```go
// 🟢 修復後的代碼 (internal/ui/model.go)
case "enter":
    if m.isHistorySearching {
        m.isHistorySearching = false
        targetQuery := strings.TrimSpace(m.historyStepQuery)
        m.historyStepQuery = ""
        
        filtered := m.getFilteredHistory()
        targetIdx := -1
        
        // 嘗試精確數字匹配或模糊摘要匹配
        if targetSeq, err := strconv.Atoi(targetQuery); err == nil {
            for i, e := range filtered {
                if e.StepIndex == targetSeq {
                    targetIdx = i
                    break
                }
            }
        }
        
        if targetIdx >= 0 {
            // 🟢 平滑跳轉錨定！
            m.selectedIdx = len(filtered) - 1 - targetIdx
            m.historyOffset = m.selectedIdx - (m.height / 4)
            if m.historyOffset < 0 { m.historyOffset = 0 }
            m.historySearchErr = ""
        } else {
            // 🟢 找不到時彈出錯誤橫幅，原列表完好如初
            m.historySearchErr = fmt.Sprintf("Step '%s' not found", targetQuery)
        }
    }
```

---

## 四、總結、抗體防禦與 Runbook SOP (Key Takeaways & Diagnostic SOP)

### 1. 通用設計準則 (Design Invariants)
* **導航不破壞拓撲 (Navigation Preserves Topology)**：在時序因果鏈中，跳轉功能必須維持拓撲圖譜的完整性，嚴禁為了定位目標而抹除周遭的上下文資訊。

### 2. 1 分鐘快速操作驗證 SOP
1. 在 History View 按 **`/`**；
2. 輸入目標序號（如 `5518`）並按 **`Enter`**；
3. 確認游標直接錨定在 Step #5518，且可立即按 **`j` / `k`** 自由上下移動瀏覽相鄰步驟。
