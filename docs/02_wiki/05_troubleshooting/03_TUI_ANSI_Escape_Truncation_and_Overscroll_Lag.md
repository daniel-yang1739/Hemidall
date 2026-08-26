---
title: 實戰排查：終端機 ANSI 字元隱形佔位腰斬折行與過度滾動卡頓排查 (Terminal ANSI Truncation & Overscroll Lag)
type: troubleshooting
created: 2026-08-27
updated: 2026-08-27
status: completed
tags: [troubleshooting, postmortem, rca, runbook, tui, ansi-escape, lipgloss, overscroll, hanging-indent]
aliases: [ANSI Truncation Postmortem, 終端機排版三大黑天鵝排查, ANSI 字元隱形佔位修復, 虛擬過度滾動延遲解決, TUI Layout RCA]
---

# 🛠️ 實戰排查：終端機 ANSI 字元隱形佔位腰斬折行與過度滾動卡頓排查 (Terminal ANSI Truncation & Overscroll Lag)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 在開發全螢幕 TUI Docs 頁面時，曾遭遇兩大經典終端機黑天鵝：
> 1. **說明文字半路腰斬並留下 50 格空白**：整段繁體中文在螢幕第 60 欄突然被切斷，右半部完全留白；
> 2. **滾動到底後按 `k` 產生嚴重卡頓延遲**：按住 `j` 到底後，按 `k` 往上滾動需連按數十下畫面才會動。
> 本篇以標準 SRE 四段式事後覆盤（Postmortem）完整還原：
> * **根因 1 (ANSI 隱形膨脹)**：字串上色後的 ANSI 轉義碼（`\x1b[38;2;...m`）被寬度截斷函式當作 20 個實體欄寬，引發提早折行與二次誤殺截斷；
> * **根因 2 (虛擬過度滾動)**：`docsScroll` 缺乏上限約束，數值飆至數百，導致回滾時在可視窗口外「空轉」；
> * **架構修復**：實作 **ANSI 感知狀態機**、**29 格懸掛縮排 (Hanging Indent)** 與 **`getDocsMaxScroll` 邊界約束**。

---

## 📌 一、現象與問題定義 (Symptom & Trigger Conditions)

### 1. 異常現象 A：文字半路腰斬留白 (Premature Line Truncation)
```text
╭─────────────────────────────────────────────────────────────────────────────────────────────╮
│   1. System Instruction    : 基礎系統提示詞、開發者規範、專案憲法與安全限制，引導 Agent 核心行為                             │
│   2. MCP Tools Schema      : 工具函式呼叫的 JSON Schema 定義，包含所有可用工具、參數型別、欄位描                             │
╰─────────────────────────────────────────────────────────────────────────────────────────────╯
```
後方的「...準則。」等字消失，右側邊框前留下了整整 50 格空白！

### 2. 異常現象 B：`k` 鍵回滾卡頓 (Overscroll Empty Spin)
當使用者在 Docs 視圖按住 `j` 到底（或按 `G` 跳到底）後，按下第一下 `k`，畫面毫無反應；必須快速連按 **50~70 次 `k`**，畫面才終於開始往上移動。

---

## 🔬 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. 根因 1：`truncateVisualWidth` 缺乏 ANSI 轉義序列感知
```go
// ❌ 錯誤舊代碼：直接對 Styled 字串逐字計算 RuneWidth
func truncateVisualWidth(s string, maxVisualWidth int) string {
    w := 0
    for _, r := range []rune(s) {
        rw := runewidth.RuneWidth(r)
        if w+rw > maxVisualWidth { break } // 🚨 致命：ANSI bytes 累加了 w！
        w += rw
    }
}
```
* **連鎖反應**：
  1. `KeyStyle.Render` 與 `ColorLightText.Render` 在文字中注入了 `\x1b[38;2;108;92;231m` 等轉義碼；
  2. 每個顏色代碼約佔用 19 個不可見字元，`RuneWidth` 將其誤算為 19 欄寬；
  3. 整行文字累積了 40~50 欄的「隱形假寬度」，導致文字才印到第 70 欄，`w` 就飆到了 120 欄上限，觸發 `break` 將後續真實文字硬生生截斷拋棄！

### 2. 根因 2：`docsScroll` 數值空轉溢出 (Overscroll Overflow)
```go
// ❌ 錯誤舊代碼：無上限累加
case "j", "down":
    m.docsScroll++ // 🚨 致命：連按 100 次，數值變成 100！
case "G", "end":
    m.docsScroll = 9999
```
* 雖然渲染層有做 clamp 限制在 25 行，但記憶體中的變數坐在 100；
* 使用者按 `k` 時，數值從 100 $\to$ 99 $\to$ 98... 直到減回 24，畫面才會動，造成卡頓延遲之假象。

---

## 🛠️ 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. 核心修復 1：ANSI 感知狀態機 (ANSI-Aware State Machine)
在走訪字元時，遇到 `\x1b` 啟動 `inAnsi = true`，完整保留樣式字元但視覺寬度累加量嚴格為 0：

```go
// ✅ 修復後代碼：0 ANSI 隱形寬度損耗
for i := 0; i < len(runes); i++ {
    r := runes[i]
    if r == 0x1b {
        inAnsi = true
        res = append(res, r)
        continue
    }
    if inAnsi {
        res = append(res, r)
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
            inAnsi = false // 遇到終止字母，退出 ANSI 狀態
        }
        continue
    }
    // 正常累加真實可視寬度
    rw := runewidth.RuneWidth(r)
    if w+rw > maxVisualWidth { break }
    res = append(res, r)
    w += rw
}
```

### 2. 核心修復 2：純文字折行 + 29 格懸掛縮排 (Hanging Indent)
在純文字階段以 `descMaxWidth = contentWidth - 29` 進行折行，第 1 行頂滿右邊界，第 2+ 行縮排 29 格，兼具美感與邊界利用率！

### 3. 核心修復 3：動態最大滾動邊界約束 (`getDocsMaxScroll`)
所有滾動按鍵（`j`, `k`, `Ctrl+d`, `G`）嚴格受限於動態計算出的 `maxScroll`，確保按 `j` 到底後按下**第一下 `k` 立即 100% 毫秒級即時往上移動**！

---

## 📋 四、總結、抗體防禦與 Runbook 診斷 SOP

### 1. 長效架構抗體
* **格式化與樣式分離**：幾何尺寸計算（寬度、折行、邊界）必須在純文字階段進行，樣式上色僅在最後一層渲染；
* **狀態機嚴格邊界守恆**：所有 TUI 滾動變數必須在鍵盤事件處理層即刻進行 `[min, max]` 範圍鉗夾（Clamping）。

### 2. 故障診斷 Runbook SOP
當發現終端機文字提前折行或滾動卡頓時：
1. **執行 TUI 綜合單元測試**：
   ```bash
   go test -v ./internal/ui -run TestView3DocsPageRenderingAndSearch
   ```
2. **驗證項目**：
   * 驗證長句子完整包含關鍵字（如 `Active Turn / CoT`、`[CACHE HIT]`）；
   * 驗證模擬連按 200 次 `j` 後按 1 次 `k`，`docsScroll` 立即精確遞減 1。

---

## 🔗 五、相關概念與延伸閱讀
* [[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics|全螢幕 TUI 引擎與終端機盒模型物理]]：盒模型算術與 CJK 寬度。
* [[02_architecture/08_Interactive_Session_Switching_and_Anti_Jitter|互動式會話快切與防抖動機制]]：防抖動鎖定。
