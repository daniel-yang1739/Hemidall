# 終端機 UI 排版四大黑天鵝排查實錄：ANSI 轉義序列、背景色殘留、分隔線折行與盒模型右邊界間隙

- **建立時間**: 2026-08-27 01:33:00
- **更新時間**: 2026-08-27 01:33:00
- **模組歸屬**: `02_architecture`
- **狀態**: `RAW_INTAKE` (待 wiki-distiller 二次提煉)

---

## 🎯 概述

在構建全螢幕互動式 TUI（基於 Charmbracelet `bubbletea` 與 `lipgloss`）時，終端機的字元網格（Character Grid）與 CSS-in-Terminal 盒模型隱藏了許多與傳統 Web 前端迥異的物理陷阱。

本篇完整記錄我們在本次開發中遇到的**四大排版黑天鵝現象**、其深層底層成因（Root Cause Analysis）與數學級防禦解法。

---

## 🦆 黑天鵝一：ANSI 轉義序列長度污染導致冒號錯位

### 1. 故障現象
在名詞釋義與快捷鍵面板中，每一項的冒號（` : `）嚴重忽左忽右參差不齊，完全無法整齊對齊。

### 2. 底層成因分析
在 Go 語言中，若直接對已經過 Lipgloss 樣式渲染的字串使用標準格式化填充：
```go
// ❌ 錯誤寫法：直接對 Styled String 進行寬度填充
styledKey := KeyStyle.Render("1 / d") // 包含隱形 ANSI bytes: \x1b[38;2;108;92;231m1 / d\x1b[0m
line := fmt.Sprintf("%-24s : %s", styledKey, desc)
```
* `fmt.Sprintf("%-24s")` 是基於 **Byte 長度** 進行計算。
* `\x1b[38;2;108;92;231m` 佔用了 19 個隱形 Byte，但在終端螢幕上**可視寬度為 0**。
* `fmt.Sprintf` 誤以為字串長度已達到 24，因而僅補了少量空格，導致終端渲染時可視寬度嚴重縮水，冒號左移錯位。

### 3. 解決方案：先填充純文字，再進行樣式上色 (Pre-Pad Plain String)
```go
// ✅ 正確寫法：先以純字串補齊 24 格，再送入 Lipgloss 上色
paddedKey := fmt.Sprintf("%-24s", "1 / d")
styledKey := KeyStyle.Render(paddedKey)
line := fmt.Sprintf("%s : %s", styledKey, desc)
```

---

## 🦆 黑天鵝二：子元素 ANSI Reset `\033[0m` 破壞父容器背景色產生斑駁黑塊

### 1. 故障現象
在浮動面板中，冒號後方的字串會出現一截突兀的黑底，而換行後又變回正常背景色，畫面呈現一塊一塊不均勻的黑色斑塊。

### 2. 底層成因分析
* 父容器樣式設定了 `.Background(ColorDarkBg)`。
* 當子元素（如冒號 ` : ` 或不同顏色的說明文字）完成渲染時，Lipgloss 會在結尾輸出 ANSI Reset 序列 `\033[0m`。
* `\033[0m` 不僅重置了文字前景顏色，也**一併將終端背景色重置回終端機預設背景色（純黑）**，導致父容器設定的深灰背景被硬生生切斷，形成黑斑。

### 3. 解決方案
在終端機多層次巢狀佈局中，**避免在跨多行的父容器設置全域 Background**，而是讓各子元素自然繼承終端機預設底色，保持乾淨純粹的 ASCII 終端質感。

---

## 🦆 黑天鵝三：浮動面板內距導致分隔線過長折行（雙重分隔線）

### 1. 故障現象
在 Shortcuts 浮動面板中，頂部與底部的橫向分隔線（`──────`）看起來「過長並折到下一行」，看起來像出現了兩條分隔線。

### 2. 底層成因分析
* 面板外框樣式包含 `.Padding(0, 1)`（左右各 1 格內距）。
* 若外框內部寬度為 `modalInnerWidth`，則文字可用列寬僅有 `contentWidth = modalInnerWidth - 2`。
* 先前代碼使用了 `strings.Repeat("─", modalInnerWidth)`，其長度比可用空間**整整多出 2 個字元**。
* 終端機將這多出的 2 個字元自動換行（Line-Wrap）至下一行，造成視覺上的雙重框線。

### 3. 解決方案
嚴格計算 `contentWidth := modalInnerWidth - 2`，並將內部所有標題、分隔線與說明文字統一截斷至 `contentWidth`：
```go
modalInnerWidth := modalWidth - 4
contentWidth := modalInnerWidth - 2 // 扣除左右各 1 格 Padding

contentLines = append(contentLines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", contentWidth)))
```

---

## 🦆 黑天鵝四：外框右邊界 2~4 列間隙導致未頂到底

### 1. 故障現象
頂部 Header 的 `(aa726359)` 凸出在最右側，而下方 Dashboard 面板與 History 雙欄面板的右邊框線（`╮`、`│`、`╯`）卻向內縮進了 2 ~ 4 個字元，未能與 Header 齊平頂到螢幕最右側。

### 2. 底層成因分析
在 Lipgloss 盒模型中：
* `PanelStyle` 包含 `.Border(lipgloss.RoundedBorder())`（左右各 1 格，共 2 格）與 `.Padding(0, 1)`。
* 當調用 `PanelStyle.Width(W)` 時，`W` 是框線內的總寬度（包含 Padding），加上 2 格 Border 後，**外框總寬度為 $W + 2$**。
* 先前代碼誤把 Padding 重複扣除，設定了 $W = m.width - 4$，使得外框總寬度僅為 $(m.width - 4) + 2 = \mathbf{m.width - 2}$（少了 2 格）；History 左右雙欄各扣了 4 格，導致總寬度僅為 $\mathbf{m.width - 4}$（少了 4 格）。

```text
[Header]  AGENT-OBSERVER   [1] Dashboard...         01:26:08 | Events: 2200 | (aa726359)  <-- Column 120
[Box Top] ╭────────────────────────────────────────────────────────────────────────────╮    <-- Column 118 (Short by 2!)
```

### 3. 解決方案：精確全寬數學對齊
1. **單欄面板（Dashboard / Docs）**：
   $$\text{panelInnerWidth} = m.width - 2 \implies \text{Outer Width} = (m.width - 2) + 2 = \mathbf{m.width}$$
2. **雙欄面板（History）**：
   $$\text{leftOuterWidth} + \text{rightOuterWidth} = m.width$$
   $$\text{listInnerWidth} = \text{leftOuterWidth} - 2, \quad \text{detailInnerWidth} = \text{rightOuterWidth} - 2$$
   $$\text{Total Outer Width} = (\text{leftOuterWidth} - 2 + 2) + (\text{rightOuterWidth} - 2 + 2) = \mathbf{m.width}$$
3. **頂部 Header**：
   $$\text{gapWidth} = m.width - \text{len}(\text{left}) - \text{len}(\text{right}) \implies \text{Total Width} = \mathbf{m.width}$$

---

## 🧪 實機單元測試驗證輸出 (`TestWidthMeasurement`)

在 $m.width = 120$ 的終端環境下，實測每一行的精確像素/字元寬度：
```bash
=== RUN   TestWidthMeasurement
    tui_test.go:339: Header length: 120, string:   AGENT-OBSERVER   [1] Dashboard  [2] History  [3] Docs                             01:26:08 | Events: 2200 | (aa726359)
    tui_test.go:345: Dash line 0 width: 120 | ╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
    tui_test.go:355: Hist line 0 width: 120 | ╭────────────────────────────────────╮╭────────────────────────────────────────────────────────────────────────────────╮
--- PASS: TestWidthMeasurement (0.00s)
```
**所有框線與文字的最右端 100% 絕對齊平坐落在第 120 列！**
