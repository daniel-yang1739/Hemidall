# 📐 Step Inspector 雙框拆分與雙模式響應式佈局設計方案 (Technical Design & Implementation Plan)

> **Document Type**: Architecture & Layout Specification  
> **Date**: 2026-08-27  
> **Location**: `docs/01_raw/2026-08-27_141800_step_inspector_split_and_responsive_layout_plan.md`  
> **Status**: Ready for Review / Execution  

---

## 🎯 1. 目標概述 (Goal Description)

本重構旨在將 **Step Inspector（步驟檢閱器）** 徹底拆分為兩個職責分明的獨立框格（Panel），並根據終端機寬度自動切換最符合人體工學的 3-Panel 佈局：

1. **`STEP TELEMETRY & METRICS`（狀態與遙測面版）**：
   - 專門承載步驟編號、時間戳、執行範疇、後端模型、Token 統計、快取命中率、5 維度上下文分解、計費狀態與工具調用參數。
   - 固定高度（約 7~9 行），資訊高密度、零干擾。
2. **`CONTENT PAYLOAD`（內容載荷面板）**：
   - 專門承載思考鏈（Thinking/CoT）、實際對話內容、終端機輸出（stdout/stderr）、代碼變更 Diff 與錯誤訊息。
   - 具備獨立捲動（`j`/`k`、`Ctrl+u`/`Ctrl+d`、`g`/`G`）、Visual 模式多行反白（`v`）與剪貼簿複製（`y`）。

---

## 🖥️ 2. 佈局架構設計 (Dual-Mode Responsive Architecture)

### 模式 A：全螢幕模式 (Full-Width Mode: 寬度 $\ge 100$ 欄)
在寬螢幕下，採用 **「左側清單 + 右側上下分割」** 的高效率工作站架構：

```text
╭────────────────────────────────────╮╭──────────────────────────────────────────────────────────────────╮
│ STEPS (6536) < [T:All] [C:All]     ││ STEP TELEMETRY & METRICS                                         │
│ > [4258] MODEL_RESP [HIT 100%]     ││ • Step 4258 (DONE) at 06:00:30 | ☁️ CLOUD INFERENCE TURN          │
│     Model: Gemini 3.7 Flash        ││ • Model  : Gemini 3.7 Flash (High) (Official Telemetry)          │
│   └── [4257] OUTPUT (Local)        ││ • Tokens : Total: 151,479 | Cached: 150,866 (99.6%) | New: 613   │
│         Tool: run_cmd              ││ • 5-Dims : Sys=3,806 | Tools=1,377 | Res=0 | Hist=145,683 | Act= │
│   [4256] TOOL_CALL [HIT 100%]      │╰──────────────────────────────────────────────────────────────────╯
│     Model: Gemini 3.7 Flash        │╭──────────────────────────────────────────────────────────────────╮
│   └── [4255] OUTPUT (Local)        ││ CONTENT PAYLOAD < [Scroll: j/k, Copy: v/y]                       │
│         Tool: run_cmd              ││ # 🎯 統一結構落地！`[編號] OUTPUT (Local)` + Tool Hint            │
│   ...                              ││                                                                  │
│                                    ││ 完全照你的標準對齊！現在所有的步驟（無論是雲端還是本地）都遵循... │
╰────────────────────────────────────╯╰──────────────────────────────────────────────────────────────────╯
```

* **左欄**：固定 `38 欄`，全高度 `innerRowsLimit`。
* **右上框**：寬度 `m.width - 38`，高度固定 8~9 行，展示完整遙測與 Token 數據。
* **右下框**：寬度 `m.width - 38`，佔據剩餘全部高度，獨立捲動載荷內容。

---

### 模式 B：半螢幕模式 (Half-Width Mode: 寬度 $< 100$ 欄)
在 80 欄窄螢幕或半螢幕分頁下，採用 **「上方左右並排 + 下方全寬展開」** 的黃金分割架構：

```text
╭──────────────────────────────╮╭──────────────────────────────────────────────╮
│ STEPS (6536) < [T:All][C:All]││ STEP TELEMETRY                               │
│ > [4258] MODEL_RESP [HIT 100%││ • Step 4258 at 06:00:30 | ☁️ CLOUD INFERENCE  │
│     Model: Gemini 3.7 Flash  ││ • Model : Gemini 3.7 Flash                   │
│   └── [4257] OUTPUT (Local)  ││ • Tokens: Total: 151k | Cached: 150k (99.6%) │
│         Tool: run_cmd        ││ • 5-Dims: Sys=3.8k | Tools=1.3k | Hist=145k  │
╰──────────────────────────────╯╰──────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────╮
│ CONTENT PAYLOAD < [Scroll: j/k, Copy: v/y]                                   │
│ # 🎯 統一結構落地！`[編號] OUTPUT (Local)` + Tool Hint                         │
│                                                                              │
│ 完全照你的標準對齊！現在所有的步驟（無論是雲端還是本地）都遵循**最嚴謹的一致性語法**：│
│   ...                                                                        │
╰──────────────────────────────────────────────────────────────────────────────╯
```

* **左上框**：寬度 `36~38 欄`，高度約 9~10 行，展示緊湊步驟階層清單。
* **右上框**：寬度 `m.width - (36~38)`，高度與左上框一致（9~10 行），展示步驟元資料與 Token 狀態。
* **下方框**：寬度 **100% 全螢幕 (80 欄)**，高度佔據剩餘所有垂直高度，代碼與終端輸出完全不被橫向擠壓！

---

## 🔄 3. 焦點切換與鍵盤互動 (Focus Navigation Matrix)

我們支援自然直覺的焦點循環與滾動管理：

| 按鍵 | 當前焦點：Step List | 當前焦點：Telemetry | 當前焦點：Content Payload |
| :--- | :--- | :--- | :--- |
| **`Tab`** | 切換至 `Content Payload` | 切換至 `Content Payload` | 切換至 `Step List` |
| **`Shift+Tab`** | 切換至 `Content Payload` | 切換至 `Step List` | 切換至 `Step List` |
| **`h` / `l`** (水平切換) | 在半螢幕切換至 `Telemetry`；全螢幕切換至 `Payload` | 切換回 `Step List` | 切換回 `Step List` |
| **`j` / `k`** (垂直捲動) | 上下移動選取步驟 | 上下滾動遙測（若有超長參數） | 上下流暢捲動 Payload 代碼與文字 |
| **`v` / `y`** | （提示切換至 Payload） | （提示切換至 Payload） | 進入 Visual 模式反白選取 / 複製到剪貼簿 |

---

## 🛠️ 4. 具體程式碼變更計畫 (Proposed Changes)

### 1. `internal/ui/model.go`
* **拆分 Inspector 內容生成器**：
  - `buildTelemetryPanelLines(e core.UnifiedAgentEvent, maxWidth int) []string`：專注於生成結構化 Metadata 與 Token 分解。
  - `buildContentPayloadLines(e core.UnifiedAgentEvent, maxWidth int) []string`：專注於生成思考鏈與長文本載荷。
* **動態計算可視卡片與滾動高度**：
  - 半螢幕模式下：Step List 容納行數由 Top-Left 框高度決定。
  - 全螢幕模式下：Step List 容納行數由 Full-Height 決定。

### 2. `internal/ui/views.go`
* **實作 `renderHistoryViewHorizontal()`（全螢幕模式）**：
  - 左：Step List
  - 右上：Telemetry Panel (`topBox`)
  - 右下：Content Payload Panel (`bottomBox`)
  - `lipgloss.JoinVertical` 組裝右側，再與左側 `lipgloss.JoinHorizontal`。
* **實作 `renderHistoryViewVertical()`（半螢幕模式）**：
  - 左上：Step List
  - 右上：Telemetry Panel
  - `lipgloss.JoinHorizontal` 組裝上方橫排。
  - 下方：Content Payload Panel（全寬）。
  - `lipgloss.JoinVertical` 組裝上下兩層。
* **保證零抖動、零高度溢出（Zero-Height Variation Invariant）**：
  - 嚴格保證無論哪種模式，總輸出高度恆等於 `m.height`。

### 3. `internal/ui/shortcuts.go`
* 更新 Shortcuts 說明表，清晰標註 Full-Width 與 Half-Width 模式下的焦點導航按鍵。

### 4. `internal/ui/tui_test.go`
* 更新並擴充全螢幕與半螢幕的單元測試（包含 `80x24`、`100x30`、`120x35`、`140x40`），驗證 3-Panel 佈局結構與零高度溢出。

---

## 🧪 5. 驗證計畫 (Verification Plan)

### 自動化測試 (Automated Tests)
```bash
cd agent-observer && go test -v ./...
```
* 確保 `TestAllViewsZeroHeightVariationAcrossSizes` 在所有尺寸（80x24 半螢幕到 140x40 全螢幕）全部通過（輸出行數恆等於終端機高度）。
* 驗證 `TestHistoryVimPaneSwitchingHL`、`TestHistoryTreeAndDistinctiveLabels` 等所有 UI 測試 PASS。

### 手動驗證 (Manual Verification)
1. 啟動 `./bin/agent-observer` 並按 `2` 進入 History 模式。
2. **全螢幕測試**：將終端機拉寬（寬度 $> 100$），確認右側清晰分為「上方狀態框」與「下方內容框」。
3. **半螢幕測試**：將終端機拉窄至半屏（寬度 $< 100$），確認上方呈現「左步驟 + 右狀態」，下方呈現「全寬內容框」。
4. 按 `Tab` 與 `j`/`k` 驗證焦點切換與下方面板滾動流暢無卡頓。
