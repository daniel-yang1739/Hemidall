# 📐 Step Inspector 雙框拆分與雙模式響應式佈局設計方案 (v3 精簡核心與零截斷保證)

> **Document Type**: Architecture & Layout Specification  
> **Date**: 2026-08-27  
> **Location**: `docs/01_raw/2026-08-27_141800_step_inspector_split_and_responsive_layout_plan.md`  
> **Status**: Ready for Review / Execution  

---

## 🎯 1. 目標概述 (Goal Description)

本重構旨在將 **Step Inspector（步驟檢閱器）** 徹底拆分為兩個職責分明的獨立框格（Panel），並根據終端機寬度自動切換最符合人體工學的 3-Panel 佈局：

1. **`STEP TELEMETRY & METRICS`（狀態與遙測面版）**：
   - 專門承載步驟編號、時間戳、執行範疇、後端模型、Token 統計、快取命中率與父子關聯。
   - **半螢幕模式下進行「精準減法」**：只呈現最關鍵的 4 大核心焦點，徹底杜絕資訊擁擠與文字截斷。
2. **`CONTENT PAYLOAD`（內容載荷面板）**：
   - 專門承載思考鏈（Thinking/CoT）、實際對話內容、終端機輸出（stdout/stderr）、代碼變更 Diff 與錯誤訊息。
   - 具備獨立捲動（`j`/`k`、`Ctrl+u`/`Ctrl+d`、`g`/`G`）、Visual 模式多行反白（`v`）與剪貼簿複製（`y`）。

---

## 🖥️ 2. 佈局架構與精確視覺模擬 (Dual-Mode Visual Layout & Content Bounds)

### 模式 A：全螢幕模式 (Full-Width Mode: 寬度 $\ge 100$ 欄)
在寬螢幕下，採用 **「左側清單 + 右側上下分割」** 的高效率工作站架構：

```text
╭────────────────────────────────────╮╭──────────────────────────────────────────────────────────────────╮
│ STEPS (6536) < [T:All] [C:All]     ││ STEP TELEMETRY & METRICS                                         │
│ > [4258] MODEL_RESP [HIT 100%]     ││ • Step #4258 (DONE) at 06:00:30 | ☁️ CLOUD INFERENCE TURN         │
│     Model: Gemini 3.7 Flash        ││ • Model  : Gemini 3.7 Flash (High) (Official Telemetry)          │
│   └── [4257] OUTPUT (Local)        ││ • Tokens : Total: 151,479 | Cached: 150,866 (99.6% HIT) | New:613│
│         Tool: run_cmd              ││ • 5-Dims : Sys=3,806 | Tools=1,377 | Hist=145,683 | Act=613      │
│   [4256] TOOL_CALL [HIT 100%]      ││ • Parent : User Request in Step #4243                            │
│     Model: Gemini 3.7 Flash        │╰──────────────────────────────────────────────────────────────────╯
│   └── [4255] OUTPUT (Local)        │╭──────────────────────────────────────────────────────────────────╮
│         Tool: run_cmd              ││ CONTENT PAYLOAD < [Scroll: j/k, Copy: v/y]                       │
│   ...                              ││ # 🎯 統一結構落地！`[編號] OUTPUT (Local)` + Tool Hint            │
│                                    ││                                                                  │
│                                    ││ 完全照你的標準對齊！現在所有的步驟（無論是雲端還是本地）都遵循... │
╰────────────────────────────────────╯╰──────────────────────────────────────────────────────────────────╯
```

#### 📏 全螢幕文字寬度與安全邊距：
* **左欄（38 欄外寬 / 34 欄內淨寬）**：最長行 `> [4258] MODEL_RESP [HIT 100%]`（29 欄）$\le 34$ 欄，**安全裕度 +5 欄**。
* **右上 Telemetry（$\ge 62$ 欄外寬 / $\ge 58$ 欄內淨寬）**：最長行 `Tokens` 明細（52 欄）$\le 58$ 欄，**安全裕度 +6~20 欄**。

---

### 模式 B：半螢幕模式 (Half-Width Mode: 寬度 $< 100$ 欄，以標準 80 欄為例)
在 80 欄窄螢幕或半螢幕分頁下，採用 **「上方左右並排 + 下方全寬展開」**，且右上狀態框**只萃取最關鍵的 4 大核心指標**：

```text
╭────────────────────────────────────╮╭────────────────────────────────────────╮
│ STEPS (6536) < [T:All] [C:All]     ││ STEP TELEMETRY                         │
│ > [4258] MODEL_RESP [HIT 100%]     ││ • Step  : #4258 (DONE) at 06:00:30     │
│     Model: Gemini 3.7 Flash        ││ • Model : Gemini 3.7 Flash             │
│   └── [4257] OUTPUT (Local)        ││ • Cache : 150.8k / 151.4k (99.6% HIT)  │
│         Tool: run_cmd              ││ • Parent: Step #4243 (User Prompt)     │
│   [4256] TOOL_CALL [HIT 100%]      ││                                        │
│     Model: Gemini 3.7 Flash        ││                                        │
│   ...                              ││                                        │
╰────────────────────────────────────╯╰────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────╮
│ CONTENT PAYLOAD < [Scroll: j/k, Copy: v/y]                                   │
│ # 🎯 統一結構落地！`[編號] OUTPUT (Local)` + Tool Hint                         │
│                                                                              │
│ 完全照你的標準對齊！現在所有的步驟（無論是雲端還是本地）都遵循**最嚴謹的一致性語法**：│
│   - 第一行（Step Header）：遵循標準的 [編號] TYPE [標籤]                       │
│   - 第二行（Metadata Hint）：專門提供精確的下層元資料                         │
╰──────────────────────────────────────────────────────────────────────────────╯
```

#### 📏 半螢幕模式極簡化指標設計 (嚴格 $\le 35$ 欄，100% 絕不截斷)：
| 步驟類型 | 半螢幕 Telemetry 呈現的 4 大核心焦點 | 最長行長度 | 框格可用淨寬 (38 欄) |
| :--- | :--- | :--- | :--- |
| **雲端推理 (Cloud Turn)** | 1. `• Step  : #4258 (DONE) at 06:00:30`<br>2. `• Model : Gemini 3.7 Flash`<br>3. `• Cache : 150.8k / 151.4k (99.6% HIT)`<br>4. `• Parent: Step #4243 (User Prompt)` | **35 欄** | ✅ **剩餘 +3 欄 (零截斷)** |
| **本地執行 (Local Step)** | 1. `• Step  : #4257 (DONE) at 06:00:29`<br>2. `• Action: Tool Output (run_cmd)`<br>3. `• Status: Offline Process (0 Tokens)`<br>4. `• Parent: Triggered by Step #4256` | **34 欄** | ✅ **剩餘 +4 欄 (零截斷)** |
| **使用者提問 (User Input)** | 1. `• Step  : #4243 (DONE) at 05:58:10`<br>2. `• Origin: Human Client Prompt`<br>3. `• Tokens: ~140.2k Inbound Context`<br>4. `• Billed: Cloud Turn #4244` | **33 欄** | ✅ **剩餘 +5 欄 (零截斷)** |

---

## 🎨 3. 資訊可讀性與視覺層級規範 (Readability Guidelines)

1. **Telemetry 面板配色與排版**：
   * **標籤**：`• Step`, `• Model`, `• Cache`, `• Parent` 採用青色粗體（Bold Cyan）。
   * **關鍵狀態**：快取命中率採用亮綠色 `(99.6% HIT)`、部分命中鮮黃色 `(PARTIAL)`、冷啟動紅色 `(MISS)`。
   * **數值簡化**：在半螢幕模式下自動啟用簡明單位（如 `150.8k`），兼顧精確性與排版呼吸感。
2. **Content Payload 面板獨立大畫布**：
   * 在半螢幕下**直接獨享 80 欄全寬**，長行日誌、代碼 Diff、終端輸出不再被強行折成碎片。
   * 思考鏈（Thinking）獨立於內容頂部，以淡紫色斜體呈現。

---

## 🔄 4. 焦點切換與鍵盤互動 (Focus Navigation Matrix)

| 按鍵 | 當前焦點：Step List | 當前焦點：Content Payload |
| :--- | :--- | :--- |
| **`Tab`** / **`Shift+Tab`** | 切換至 `Content Payload` | 切換至 `Step List` |
| **`h` / `l`** | 切換至 `Content Payload` | 切換至 `Step List` |
| **`j` / `k`** | 上下選取步驟 | 上下滾動 Payload 代碼與文字 |
| **`v` / `y`** | （自動切換至 Payload 並開啟選取） | 進入 Visual 模式反白選取 / 複製到剪貼簿 |

---

## 🛠️ 5. 具體程式碼變更計畫 (Proposed Changes)

### 1. `internal/ui/model.go`
* **實作專用生成器**：
  - `buildTelemetryPanelLines(e core.UnifiedAgentEvent, maxWidth int, isCompact bool) []string`：根據 `isCompact` 輸出全螢幕完整版或半螢幕 4 大核心焦點。
  - `buildContentPayloadLines(e core.UnifiedAgentEvent, maxWidth int) []string`：專注於生成思考鏈與長文本載荷。
* **動態卡片數量計算**：
  - 半螢幕模式下：Step List 容納行數由 Top-Left 框高度（8 行內容）決定。
  - 全螢幕模式下：Step List 容納行數由 Full-Height（`m.height - 4`）決定。

### 2. `internal/ui/views.go`
* **實作 `renderHistoryViewHorizontal()`（全螢幕模式）**：
  - 左：Step List（38 欄全高）
  - 右上：Telemetry Panel（完整版）
  - 右下：Content Payload Panel（佔據剩餘高度）
* **實作 `renderHistoryViewVertical()`（半螢幕模式）**：
  - 左上：Step List（38 欄，8 行內容）
  - 右上：Telemetry Panel（極簡 4 核心版，42 欄，8 行內容）
  - 下方：Content Payload Panel（全寬 80 欄，佔據剩餘全部高度）
* **嚴格保證輸出總行數恆等於 `m.height`（零抖動、零高度溢出）**。

### 3. `internal/ui/tui_test.go`
* 擴充全尺寸單元測試（`80x24`、`100x30`、`120x35`、`140x40`），驗證雙模式結構、零字元截斷與零高度溢出。

---

## 🧪 6. 驗證計畫 (Verification Plan)

### 自動化測試 (Automated Tests)
```bash
cd agent-observer && go test -v ./...
```
* 確保 `TestAllViewsZeroHeightVariationAcrossSizes` 在所有尺寸全部通過。

### 手動驗證 (Manual Verification)
1. 啟動 `./bin/agent-observer` 並按 `2` 進入 History 模式。
2. **半螢幕測試**：將終端機縮至 80 欄，確認右上狀態框只展示 4 條清晰核心數據，下方為 80 欄全寬載荷。
3. **全螢幕測試**：拉寬終端機，確認右側切為上下兩塊，展示完整遙測與長文本。
