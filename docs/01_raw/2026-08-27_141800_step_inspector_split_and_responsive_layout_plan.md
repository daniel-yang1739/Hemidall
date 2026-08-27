# 📐 Step Inspector 雙框拆分、響應式佈局與動態排版架構方案 (Finalized Specification)

> **文件類型**：UI/UX Architecture & Implementation Specification  
> **日期**：2026-08-27  
> **路徑**：`docs/01_raw/2026-08-27_141800_step_inspector_split_and_responsive_layout_plan.md`  
> **狀態**：✅ IMPLEMENTED & VERIFIED (全量測試通過，二進位就緒)  

---

## 🎯 一、 重構目標與核心價值 (Goal & Core Values)

在終端機 Agent Observer 中，**Step Inspector（步驟檢閱器）** 原先將元數據狀態與大量文本內容擠在同一個視窗，造成滾動體驗不佳與窄螢幕排版截斷問題。本架構將 Step Inspector 徹底拆分為兩個職責分明的獨立框格，並根據終端機寬度自動切換最佳的 3-Panel 佈局：

1. **`STEP TELEMETRY & METRICS`（狀態與遙測面版）**：
   - 專門承載步驟編號、本地時區時間戳、執行範疇（Cloud/Local/User）、後端模型、Token 統計、快取命中率、5 維度數據與拓撲父子關聯。
   - **半螢幕模式下進行「結構化減法與二級 Sub-Bullets」**：透過樹狀子條目階層呈現，徹底消除文字被硬切截斷（零溢出保證）。
2. **`CONTENT PAYLOAD`（內容載荷面板）**：
   - 專門承載思考鏈（Thinking/CoT）、實際對話內容、終端機輸出（stdout/stderr）、代碼變更 Diff 與錯誤訊息。
   - 支援獨立捲動（`j`/`k`、`Ctrl+u`/`Ctrl+d`）、Visual 模式反白選取（`v`）與剪貼簿複製（`y`）。

---

## 🖥️ 二、 雙模式響應式佈局架構 (Dual-Mode Responsive Architecture)

### 模式 A：全螢幕模式 (Full-Width Mode: 寬度 $\ge 100$ 欄)
在寬螢幕下，採用 **「左側清單 + 右側上下分割」** 的高效率工作站架構：

```text
╭────────────────────────────────────╮╭──────────────────────────────────────────────────────────────────╮
│ STEPS (6536) <                     ││ STEP TELEMETRY & METRICS                                         │
│ Filters: [T:All] [C:All]           ││ • Step #4258 (DONE) at 14:00:30 | ☁️ CLOUD INFERENCE TURN         │
│ > [4258] MODEL_RESP [HIT 100%]     ││ • Model  : Gemini 3.7 Flash (High) (Official Telemetry)          │
│     Model: Gemini 3.7 Flash        ││ • Tokens : Total: 151,479 | Cached: 150,866 (99.6% HIT) | New:613│
│   └── [4257] OUTPUT (Local)        ││ • 5-Dims : Sys=3,806 | Tools=1,377 | Hist=145,683 | Act=613      │
│         Tool: run_cmd              ││ • Parent : User Request in Step #4243                            │
│   [4256] TOOL_CALL [HIT 100%]      │╰──────────────────────────────────────────────────────────────────╯
│     Model: Gemini 3.7 Flash        │╭──────────────────────────────────────────────────────────────────╮
│   └── [4255] OUTPUT (Local)        ││ CONTENT PAYLOAD < [Scroll: j/k, Copy: v/y]                       │
│         Tool: run_cmd              ││ # 🎯 統一結構落地！`[編號] OUTPUT (Local)` + Tool Hint            │
│   ...                              ││                                                                  │
│                                    ││ 完全照你的標準對齊！現在所有的步驟（無論是雲端還是本地）都遵循... │
╰────────────────────────────────────╯╰──────────────────────────────────────────────────────────────────╯
```

#### 📏 全螢幕排版規範：
* **左欄（38 欄外寬 / 34 欄內淨寬）**：
  * 第 1 行：主標題與計數 `STEPS (6536) <`
  * 第 2 行：過濾器狀態列 `Filters: [T:All] [C:All]`
  * 第 3 行起：依序排列步驟卡片，卡片寬度嚴格 $\le 34$ 欄。
* **右上 Telemetry（$\ge 62$ 欄外寬 / $\ge 58$ 欄內淨寬，高度固定 8 行）**：展示完整官方 Token 分解、5-Dims 數據與計費真理。
* **右下 Content Payload（佔據剩餘全部高度）**：獨立滾動長文本載荷。

---

### 模式 B：半螢幕模式 (Half-Width Mode: 寬度 $< 100$ 欄，以標準 80 欄為例)
在 80 欄窄螢幕或半螢幕分頁下，採用 **「上方左右並排 + 下方全寬展開」**：

```text
╭────────────────────────────────────╮╭────────────────────────────────────────╮
│ STEPS (6054) <                     ││ STEP TELEMETRY                         │
│ Filters: [T:All] [C:All]           ││ • Step  : #4446 (DONE) at 14:31:13     │
│   ...                              ││ • Model : Gemini 3.7 Flash (High)      │
│ > [4446] TOOL_CALL [HIT 100%]      ││ • Tokens: 217.5k Total Context         │
│     Model: Gemini 3.7 Flash (High) ││   ├ Cached: 217.4k (100.0% HIT)        │
│   └── [4445] OUTPUT (Local)        ││   └ New   : 50 new tokens              │
│         Tool: edit_file            ││ • 5-Dims:                              │
│   [4444] TOOL_CALL [HIT 100%]      ││   ├ Sys: 3.8k | Tools: 1.4k            │
│     Model: Gemini 3.7 Flash (High) ││   └ Hist: 212.3k | Act: 50             │
│   └── [4443] OUTPUT (Local)        ││ • Parent: Step #4433 (User Prompt)     │
│         Tool: view_file            ││ • Tool  : view_file (1 call)           │
╰────────────────────────────────────╯╰────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────╮
│ CONTENT PAYLOAD < [Scroll: j/k, Copy: v/y]                                   │
│ # 🎯 內容載荷獨佔 100% 全螢幕寬度（80 欄）                                   │
│                                                                              │
│ 享受最大面積的長行代碼 Diff、終端機 stdout/stderr 與日誌閱讀！               │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## 🌲 三、 遙測數據二級 Sub-Bullets 階層設計 (Zero Truncation Guarantee)

為了解決單行過長導致末端字元被截斷（例如 `5-Dims: Sys 3.8k | Tools 1.4k | His`）的問題，我們在半螢幕狀態框引入**二級樹狀子條目（Sub-Bullets）**：

| 步驟類型 | 結構化 Sub-Bullets 呈現內容 | 最長行字元長度 | 框格可用淨寬 (38 欄) |
| :--- | :--- | :--- | :--- |
| **雲端推理 (Cloud Turn)** | `• Step  : #4446 (DONE) at 14:31:13`<br>`• Model : Gemini 3.7 Flash (High)`<br>`• Tokens: 217.5k Total Context`<br>`  ├ Cached: 217.4k (100.0% HIT)`<br>`  └ New   : 50 new tokens`<br>`• 5-Dims:`<br>`  ├ Sys: 3.8k \| Tools: 1.4k`<br>`  └ Hist: 212.3k \| Act: 50`<br>`• Parent: Step #4433 (User Prompt)`<br>`• Tool  : view_file (1 call)` | **27 欄** | ✅ **剩餘 +11 欄 (零截斷)** |
| **本地執行 (Local Step)** | `• Step  : #4445 (DONE) at 14:31:12`<br>`• Action: Tool Output (edit_file)`<br>`• Status: Offline Process (0 tok)`<br>`  └ Billed: Packaged in #4446`<br>`• Parent: Triggered by #4444`<br>`• Origin: Local Machine Subprocess` | **33 欄** | ✅ **剩餘 +5 欄 (零截斷)** |
| **使用者提問 (User Input)** | `• Step  : #4433 (DONE) at 14:30:10`<br>`• Origin: Human Client Prompt`<br>`• Tokens: ~217.4k Inbound Context`<br>`  ├ Active: ~227 prompt tokens`<br>`  └ Cached: ~217.2k (99.9%)`<br>`• Status: Inbound to GPU Cluster` | **31 欄** | ✅ **剩餘 +7 欄 (零截斷)** |

---

## ⚡ 四、 步驟清單動態行數打包算法 (Dynamic Line Packing)

### 📌 歷史問題：
原先系統計算可視卡片數時採用靜態假設 `maxCards = availLines / 2`。當使用者過濾至 **`USER_INPUT`（單行卡片，無 Model/Tool Hint）** 時，30 行的高度只渲染了 15 個步驟，導致清單下方出現 14 行巨大留白。

### 🛠️ 動態打包算法實作 (`getHistoryVisibleCards`)：
1. 從當前捲動起點 `m.historyOffset` 開始遍歷。
2. 針對每一筆步驟，精準判斷其所需行數：
   - 帶有 Hint 的步驟（Cloud/Local）：`linesNeeded = 2`
   - 無 Hint 的步驟（`USER_INPUT` 或單行步驟）：`linesNeeded = 1`
3. 累加行數直到剛好填滿可用行數 `availLines`（並在存在後續項目時預留 1 行放置 `  ...`）。
4. **效果**：無論過濾到哪種步驟，左欄清單永遠緊密填滿至最底行，徹底消除無效留白！

---

## 🌐 五、 主機本地時區全域同步 (Local Timezone Synchronization)

所有時間戳統一轉為主機本地時區：
1. **資料解析層 (`watcher.go`)**：解析 RFC3339 時間戳時執行 `t = t.Local()`。
2. **UI 渲染層 (`views.go`, `model.go`)**：
   - Dashboard Track 1 官方時間：`e.Timestamp.Local().Format("2006-01-02 15:04:05")`
   - Dashboard Panel 3 即時事件：`ev.Timestamp.Local().Format("15:04:05")`
   - Step History 遙測面板：`e.Timestamp.Local().Format("15:04:05")`

---

## 🔄 六、 焦點切換與鍵盤互動矩陣 (Focus Navigation Matrix)

| 按鍵 | 當前焦點：Step List | 當前焦點：Content Payload |
| :--- | :--- | :--- |
| **`Tab`** / **`Shift+Tab`** | 切換至 `Content Payload` | 切換至 `Step List` |
| **`h` / `l`** | 切換至 `Content Payload` (`l`) | 切換至 `Step List` (`h`) |
| **`j` / `k`** | 上下選取步驟（即時更新 Telemetry 與 Payload） | 上下滾動 Payload 長代碼與文本 |
| **`Ctrl+d` / `Ctrl+u`** | 快速翻頁 10 個步驟 | 快速滾動 Payload 10 行 |
| **`t` / `T`** | 循環切換步驟類型過濾（All ➔ Tool ➔ Model ➔ User ➔ Code） | - |
| **`c` / `C`** | 循環切換快取狀態過濾（All ➔ Hit ➔ Partial ➔ Write ➔ Miss） | - |
| **`/`** | 進入步驟編號搜尋模式（`Filter: [#...]`） | - |
| **`v` / `y`** | （自動切換至 Payload 並開啟選取） | 進入 Visual 模式反白選取 / 複製到剪貼簿 |

---

## 🧪 七、 驗證與測試矩陣 (Verification & Test Invariants)

```bash
cd agent-observer && go test -v ./...
```
* ✅ `TestHistoryThreePanelSplitAndZeroTruncation`: 驗證全螢幕與半螢幕 3-Panel 佈局及 `Filters:` 列。
* ✅ `TestAllViewsZeroHeightVariationAcrossSizes`: 在 `80x24`、`100x30`、`120x35`、`140x40` 下輸出總行數恆等於終端機高度，零高度抖動。
* ✅ `TestHistoryVimPaneSwitchingHL`: 驗證 Vim `h`/`l` 焦點切換與滾動邊界。
* ✅ `TestHistoryTreeAndDistinctiveLabels`: 驗證樹狀階層與標籤前綴。
* ✅ `TestWidthMeasurement`: 驗證邊框寬度對齊。

