---
title: 全螢幕 TUI 引擎架構、終端機盒模型與 CJK 渲染物理 (TUI Engine & Terminal Layout Mechanics)
type: architecture
created: 2026-08-26
updated: 2026-08-26
status: completed
tags: [architecture, tui, bubbletea, lipgloss, terminal-layout, box-model, cjk-rendering, go-runewidth, overflow-hidden, concrete-walkthrough]
aliases: [TUI Engine, 終端機佈局物理, Lipgloss 盒模型, CJK 2 列寬度, Terminal Overflow Hidden, 虛擬滾動緩衝區]
---

# 🖥️ 全螢幕 TUI 引擎架構、終端機盒模型與 CJK 渲染物理 (TUI Engine & Terminal Layout Mechanics)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 現代 AI 觀測工具（如 `agent-observer`）採用 **Bubbletea (Elm 架構) 與 Lipgloss 樣式引擎** 打造全螢幕互動式 TUI。然而，終端機開發面臨著三大傳統 Web CSS 所沒有的底層物理陷阱：
> 1. **等寬網格限制 (Monospace Grid)**：終端機只有離散的行與欄，零像素插值；
> 2. **Lipgloss 盒模型內外距疊加**：`.Width(W)` 設定內部尺寸，疊加 `.Border()` (+2) 與 `.Padding()` (+2) 後實際外徑達 $W+4$；
> 3. **中文字元 (CJK) 2 倍列寬隱形折行**：Go 的 `len([]rune)` 算 1 個字元，但在終端機佔用 2 欄，導致隱形換行撐爆容器。
> 透過 **`go-runewidth` 視覺寬度硬切、統一虛擬滾動緩衝區（Unified Scroll Buffer）與像素級垂直行數鎖定**，徹底實現終端機級的 `overflow: hidden; white-space: nowrap`！

---

## 🔍 一、技術背景：終端機渲染的特殊物理約束

與現代瀏覽器渲染引擎（Blink/WebKit）不同，終端機模擬器（如 iTerm2, macOS Terminal, Alacritty）是一張由 **$W$ 欄 (Columns) $\times$ $H$ 行 (Rows)** 組成的二維字元網格：

```text
(0,0) ┌────────────────────────────────────────────────────────────┐
      │ Monospace Character Grid (80 x 24 / 120 x 35)               │
      │ 每個英文字元佔用 1 欄 (Width=1)                            │
      │ 每個中文字元 (CJK) 或 Emoji 佔用 2 欄 (Width=2)             │
      │ 若單行總視覺寬度 > W，終端機會「強制向下折行 (Line Wrap)」  │
      └────────────────────────────────────────────────────────────┘ (W-1, H-1)
```

### 🚨 終端機自動折行 (Line Wrap) 的連鎖災難：
若在 80 欄終端機中輸出一行 81 欄的字串，終端機會自動將第 81 字元換到下一行，導致整張畫面的行數從 24 行膨脹為 25 行。在全螢幕模式下，這 1 行的膨脹會導致**全螢幕向下位移滾動，游標落到螢幕外，底部快捷鍵被推擠出視野！**

---

## 🏛️ 二、TUI 垂直行數預算與盒模型幾何架構圖

```mermaid
flowchart TD
    subgraph Screen ["🖥️ 全螢幕終端機預算 (Height = H, Width = W)"]
        direction TB
        L0["Line 0: 頂部狀態列 Header (1 行)"]
        L1["Line 1: 雙欄頂邊框線 ╭───────────╮╭───────────────────────╮ (1 行)"]
        L2["Line 2 ~ H-3: 雙欄可視內容 (嚴格等於 H - 4 行，零折行)"]
        LH2["Line H-2: 雙欄底邊框線 ╰───────────╯╰───────────────────────╯ (1 行)"]
        LH1["Line H-1: 底部快捷鍵列 Footer (1 行，頂在螢幕最底行)"]
        
        L0 --> L1
        L1 --> L2
        L2 --> LH2
        LH2 --> LH1
    end
    
    style L0 fill:#e8f4f8,stroke:#17a2b8
    style LH2 fill:#d4edda,stroke:#28a745
    style LH1 fill:#fff3cd,stroke:#ffc107
```

### 📐 嚴格垂直行數守恆公式：
$$\text{總垂直行數} = 1 \text{ (Header)} + 1 \text{ (Top Border)} + (H - 4) \text{ (Content)} + 1 \text{ (Bottom Border)} + 1 \text{ (Footer)} \equiv \mathbf{H \text{ 行}}$$

---

## 🎯 三、Lipgloss 盒模型內外距算術 (Box Model Arithmetic)

在 Lipgloss 中，樣式的寬高設定是針對 **內部可繪製區域 (Inner Dimension)**：

```mermaid
flowchart LR
    subgraph Box ["📦 Lipgloss Bordered Box"]
        direction TB
        Top["Border Top (1 行)"]
        PadL["Left Border (1) + Left Padding (1) = 2 欄"]
        Content["內部文字可視寬度 (listInnerWidth - 2)"]
        PadR["Right Padding (1) + Right Border (1) = 2 欄"]
        Bot["Border Bottom (1 行)"]
    end
```

$$\text{外部總寬度 (Outer Width)} = W_{\text{inner}} + 2 \text{ (Border)} + 2 \text{ (Padding)} = W_{\text{inner}} + 4$$
$$\text{外部總高度 (Outer Height)} = H_{\text{inner}} + 2 \text{ (Top/Bottom Border)} = (H - 4) + 2 = H - 2$$

👉 **避坑鐵律**：若左欄欲分配 $32\%$ 螢幕寬度，則：
$$W_{\text{left\_outer}} = \text{int}(0.32 \times W) \implies W_{\text{list\_inner}} = W_{\text{left\_outer}} - 4$$
文字在排版截斷時，可用的文字物理欄寬為 **$W_{\text{list\_inner}} - 2$**（扣除左右各 1 格 Padding 空白）。

---

## 🔬 四、中文字元 (CJK) 2 倍列寬與 `go-runewidth` 硬切實作

為徹底根除繁體中文、全形符號與 Emojis 造成的隱形折行，必須使用 `go-runewidth` 針對字元終端列寬進行計算：

```go
// truncateVisualWidth 依據終端機物理列寬 (CJK/Emoji 算 2 列, 英文算 1 列) 進行精確硬切
// 徹底實現終端機級 overflow: hidden; white-space: nowrap
func truncateVisualWidth(s string, maxVisualWidth int) string {
    s = strings.ReplaceAll(s, "\n", " ")
    s = strings.ReplaceAll(s, "\r", "")
    s = strings.ReplaceAll(s, "\t", "    ")

    if maxVisualWidth <= 0 {
        return ""
    }

    w := 0
    var res []rune
    for _, r := range []rune(s) {
        rw := runewidth.RuneWidth(r)
        if w+rw > maxVisualWidth {
            break
        }
        res = append(res, r)
        w += rw
    }
    return string(res)
}
```

---

## 📜 五、右欄軟換行虛擬緩衝區 (Soft-Wrapping Virtual Buffer)

在 Inspector 右欄中，長代碼或長段落若直接硬切會遺失資訊。系統採用 **Pre-wrapping 軟換行虛擬緩衝區**：

```go
// wrapVisualLines 將長段落預先切成符合欄寬的虛擬行，保證每一行 visualWidth <= maxWidth
func wrapVisualLines(text string, maxWidth int) []string {
    if maxWidth <= 0 {
        return []string{""}
    }
    var result []string
    lines := strings.Split(text, "\n")
    for _, line := range lines {
        line = strings.ReplaceAll(line, "\r", "")
        line = strings.ReplaceAll(line, "\t", "    ")
        if line == "" {
            result = append(result, "")
            continue
        }
        runes := []rune(line)
        var currentChunk []rune
        currentW := 0
        for _, r := range runes {
            rw := runewidth.RuneWidth(r)
            if currentW+rw > maxWidth {
                if len(currentChunk) > 0 {
                    result = append(result, string(currentChunk))
                    currentChunk = nil
                    currentW = 0
                }
            }
            currentChunk = append(currentChunk, r)
            currentW += rw
        }
        if len(currentChunk) > 0 || len(runes) == 0 {
            result = append(result, string(currentChunk))
        }
    }
    return result
}
```

### 💡 雙重保障：
1. **內容 100% 完整**：長段落完整保留在緩衝區中，可透過 `detailScroll` 垂直滾動閱讀；
2. **高度 100% 鎖死**：視窗切片永遠只取 `innerRowsLimit - 1` 行，左右兩欄高度永遠絕對平齊！

---

## 🗂️ 六、左欄雙行步驟卡片佈局 (2-Line Step Card)

```text
╭─────────────────────────────╮
│ 📜 Steps (Newest First ◀)   │
│ ▶ [035|MODEL] 15:04:05      │  <- Card 1 Line 1 (序號 + 類型 + 時間戳)
│   剛剛你說3x 萬字是為什麼... │  <- Card 1 Line 2 (縮排摘要文字)
│   [034|TOOL ] 15:03:50      │  <- Card 2 Line 1
│   view_file (internal/ui)   │  <- Card 2 Line 2
╰─────────────────────────────╯
```

---

## 🔗 七、相關概念與延伸閱讀
* [[03_Agent_Storage_and_State_Machine]]：SQLite 狀態機與 Protobuf 載荷。
* [[06_Dual_Track_Telemetry_and_Window_Accounting]]：雙軌遙測與倒推滑動窗口。
* [[04_Service_Plan_Agent_Observer]]：`agent-observer` 完整服務架構規劃。
