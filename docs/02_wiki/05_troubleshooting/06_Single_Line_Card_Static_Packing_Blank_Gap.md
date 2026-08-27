---
title: 實戰排查：單行卡片靜態除二計算導致清單底部大片留白 (Dynamic Line Packing Bug Postmortem)
type: troubleshooting
created: 2026-08-27
updated: 2026-08-27
status: completed
tags: [troubleshooting, tui, line-packing, dynamic-layout, bubbletea, terminal-rendering, bug-fix, runbook]
aliases: [Single Line Blank Gap, 靜態行數除二漏洞, 動態行數打包修復, History List Density Fix]
---

# 🛡️ 實戰排查：單行卡片靜態除二計算導致清單底部大片留白 (Dynamic Line Packing Bug Postmortem)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 在使用 `agent-observer` 的歷史步驟過濾器切換至 `[T:User]`（純使用者輸入）時，在高度高達 30 行的終端機視窗中，清單居然只展示了 8 筆資料，底部留下了整整 14 行大片空蕩蕩的空白區域。
> 根因排查發現，`getHistoryVisibleCards()` 原先採用靜態公式 `visibleCount = availLines / 2`（預設每張卡片皆為 2 行高）。當清單中全部是單行卡片時，靜態計算嚴重低估了可容納卡片數。
> 透過改採 **動態行數打包演算法 (Dynamic Line Packing Loop)**，即時累加單行卡片（1 行）與雙行卡片（2 行）的實際行高，使不同高度的項目在任何篩選條件下皆能 100% 填滿視窗高度！

---

## 🔗 對應核心架構概念
* [[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics|全螢幕 TUI 引擎與終端機盒模型物理]]：動態行數打包與垂直行數守恆。
* [[02_architecture/09_History_Explorer_and_Causality_Graph|歷史步進瀏覽器與因果拓撲圖譜]]：步驟時序渲染與清單視圖。

---

## 🔍 一、現象與問題定義 (Symptom & Problem Definition)

### 🚨 異常現象描述：
在 30 行高度的終端機視窗中，左側步驟清單的可用高度為 26 行。當未啟用過濾時，畫面正常顯示約 13 筆雙行卡片（`TOOL_CALL` + Model Hint）。
然而，當按下 `t` 鍵切換至 `[T:User]`（使用者輸入步驟皆為單行 `└[0012] 👤 USER`）時，畫面**僅顯示了 13 筆單行卡片（佔據 13 行），底部留下了 13 行巨大的無效空白**！

```text
╭────────────────────────────────────╮
│ STEPS (150) <                      │
│ Filters: [T:User] [C:All]          │
│   ...                              │
│ └[0045] 👤 USER                    │
│ └[0040] 👤 USER                    │
│ └[0035] 👤 USER                    │
│ └[0030] 👤 USER                    │
│ └[0025] 👤 USER                    │
│ └[0020] 👤 USER                    │
│ └[0015] 👤 USER                    │
│ └[0010] 👤 USER                    │
│                                    │  <-- 🚨 異常！底部整整 13 行全部空白！
│                                    │
│                                    │
╰────────────────────────────────────╯
```

---

## 🔬 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. 出錯代碼定位 (`agent-observer/internal/ui/model.go`)：
檢查可視卡片計算函式 `getHistoryVisibleCards()`：

```go
// 🐛 出錯前的靜態計算：
func (m Model) getHistoryVisibleCards(availLines int) int {
    // ⚠️ 漏洞點：強制除以 2，假設所有卡片都是雙行！
    cards := availLines / 2
    if cards < 1 {
        cards = 1
    }
    return cards
}
```

### 2. 根因剖析：
* 雙行卡片（如 `TOOL_CALL` 帶有 `Model: ...`、`OUTPUT` 帶有 `Tool: ...`）需要佔用 2 行；
* 單行卡片（如 `USER_INPUT` 或無 Hint 的獨立步驟）僅需佔用 1 行；
* 當使用者透過 `[T:User]` 篩選出全單行序列時，`availLines = 26` 被除以 2 得到 `cards = 13`，渲染 13 筆單行後只用了 13 行，剩餘的 13 行直接被補空字串，造成嚴重的空間浪費！

---

## 🛠️ 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. 動態累加打包演算法實作：
放棄除法估算，改為從游標起點（`historyOffset`）向後遍歷過濾後的事件切片，動態判定每張卡片的實際行高並累加：

```go
// 🛡️ 修復後的動態打包演算法：
func (m Model) getHistoryVisibleCards(filtered []core.UnifiedAgentEvent, availLines int) int {
    if len(filtered) == 0 || availLines <= 0 {
        return 0
    }

    usedLines := 0
    cardCount := 0

    for i := m.historyOffset; i < len(filtered); i++ {
        realIdx := len(filtered) - 1 - i
        if realIdx < 0 || realIdx >= len(filtered) {
            break
        }
        e := filtered[realIdx]

        // 動態計算此步驟需要的行數
        linesNeeded := 1
        modelName := m.getStepModelName(e)
        toolName := m.getLocalToolName(e)
        if modelName != "" || (toolName != "" && toolName != "OUTPUT") {
            linesNeeded = 2
        }

        if usedLines+linesNeeded > availLines {
            break
        }
        usedLines += linesNeeded
        cardCount++
    }

    if cardCount < 1 && len(filtered) > 0 {
        return 1
    }
    return cardCount
}
```

### 2. 空間利用率對比：
* **修復前**：`[T:User]` 下僅顯示 13 筆（空間利用率 $50\%$）；
* **修復後**：`[T:User]` 下完整顯示 26 筆（空間利用率 **$100\%$**，無任何多餘留白）！

---

## 📋 四、總結、抗體防禦與 Runbook SOP (Diagnostic Runbook)

### 💡 核心收穫與通用設計抗體：
* **可變高度列表的打包鐵律**：在任何 TUI 終端機或虛擬滾動列表（Virtual Scrolling）中，**只要列表項存在可變高度（1 行 vs 2 行），就絕不能使用靜態整數除法來估算可視元素個數**，必須使用動態累加演算法（Greedy Line Packing）來填滿可繪製區域！

### 🩺 1 分鐘快速診斷 Runbook SOP：
1. **執行單元測試檢查多解析度高度守恆**：
   ```bash
   go test -v ./internal/ui -run TestAllViewsZeroHeightVariationAcrossSizes
   ```
2. **終端機實測篩選密度**：
   * 進入 `[2] History`，按 `t` 切換至 `[T:User]`，確認步驟清單自頂向下緊密排滿整個邊框，底部邊框 `╰───╯` 緊貼最後一筆資料！
