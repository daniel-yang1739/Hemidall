# 📊 規格與實作全紀錄：Dashboard 全會話統計、多模型 Token 效率矩陣與雙模響應式佈局 (v3.0 最終版)

> **建立時間**：2026-08-27 17:30:00  
> **狀態**：IMPLEMENTED & VERIFIED (Commit: `d528796`)  
> **目標模組**：`agent-observer/internal/core/` & `agent-observer/internal/ui/`  
> **關聯概念**：[[02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting]]、[[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics]]  
> **前置版本**：`2026-08-27_155000_..._plan_v1.md` (v1.0)、`2026-08-27_162500_..._plan_v2.md` (v2.0)

---

## 🌟 一、架構背景與迭代歷程 (Background & Evolution)

在 Agent 長會話開發中，單純觀察單一步驟的瞬間 Token 消耗已無法滿足整體成本與快取效率的評估需求。使用者需要清楚掌握：
1. **全會話實體處理總量 (Total Processed Tokens)**；
2. **會話層級快取命中總量 (Total Cached Tokens)** 與命中率；
3. **未快取冷啟動進水量 (Uncached Inbound Tokens)**；
4. **考量各模型快取折扣後的「實質等效計費 Token 數 (Effective Tokens)」**；
5. **實質節省的 Token 體積與比例 (Tokens Saved & Net Reduction %)**；
6. **多模型混合調用時各模型的獨立表現矩陣 (Per-Model Breakdown)**。

### 🔄 決策演進鏈條 (Evolution Timeline)
* **v1.0 (15:50)**：提出全會話統計指標與條列式初稿；
* **v2.0 (16:25)**：提出 3 種現代視覺方案（A: 5 KPI Cards, B: Table + Bar, C: Split Panel）與多模型折扣矩陣演算法；使用者選定 **「方案 A」**，並明確指示：*「不換算法幣，以純 Token 數與節省比例為主」*；
* **v3.0 (17:30 實裝反覆迭代與精雕)**：
  1. **移除厚重框線**：原本 5 個獨立的大圓角外框佔據過多空間，改為現代俐落的 **無框 KPI 欄位條 (Borderless KPI Strip)**；
  2. **去除 Title Emoji**：移除 `📊`、`📈` 等表情符號，走向工業級純淨風格；
  3. **暫時隱藏 Trend Sparklines**：因折線圖呈現方式尚需重新企劃，先將面板隱藏，保留底層次像素渲染器；
  4. **寬螢幕留白拉開**：加寬各欄分配，精簡折扣標籤（`0.25x (75% OFF)` $\to$ `0.25x`），欄位之間保持 2~4 格舒適留白；
  5. **半螢幕專屬雙模響應式 (Responsive Dual-Mode)**：
     - **統計欄位**：自動切換為雙欄 4 行結構化網格；
     - **模型表格**：精簡為 5 欄緊湊表格（包含 `Tokens Saved (%)`，總寬 71 欄，零截斷不爆版）；
     - **TRACK 1 遙測**：改為多行階層縮排，徹底解決半螢幕下文字粗暴折行問題。

---

## 🎨 二、終端視覺佈局規格 (Visual Layout Specifications)

---

### 1. 寬螢幕模式 ($\ge 110$ 欄) - 完整 7 欄模型明細與 5 欄 KPI 條

```text
╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ SESSION TOKEN AGGREGATES & MULTI-MODEL EFFICIENCY                                                                           │
│                                                                                                                             │
│   TOTAL PROCESSED        CACHE HIT VOLUME       UNCACHED INBOUND       EFFECTIVE TOKENS       TOKENS SAVED (%)              │
│   1302.22M Tok           696.51M Tok            605.71M Tok            779.83M Tok            522.39M Tok                   │
│   7912 Cloud Turns       53.5% Hit Rate         46.5% Cold In          59.9% of Raw           40.1% Net Saved               │
│                                                                                                                             │
│  MULTI-MODEL TOKEN & SAVINGS BREAKDOWN:                                                                                     │
│  Model Name                Turns     Processed       Cached (Hit %)        Uncached    Effective (Factor)    Tokens Saved (%)   │
│  gemini-3.7-flash           5845       916.24M      696.11M (76.0%)         220.13M       394.15M (0.25x)     522.08M (57.0%)   │
│  gemini-3.7-flash-safety    2065       385.54M            0 ( 0.0%)         385.54M       385.54M (0.25x)           0 ( 0.0%)   │
│  gemini-3.7-flash-high         2        443.1k       401.6k (90.6%)           41.6k        141.9k (0.25x)      301.2k (68.0%)   │
│  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────  │
│  TOTAL SUMMARY              7912     1302.22M       696.51M (53.5%)         605.71M       779.83M (0.25x)     522.39M (40.1%)   │
╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ TRACK 1: OFFICIAL TELEMETRY & STEP CONTEXT ANATOMY                                                                          │
│   • Backend Model         : gemini-3.7-flash-high  (Step #5144 | Status: DONE | 2026-08-27 17:01:46)                        │
│   • Step Active Context   : 251.7k Tokens ( 98.3% of 256k Window)  [CACHE HIT 99.4%]                                        │
│   • 5-Dimension Breakdown : Sys: 3.8k (1.5%) | Tools: 1.4k (0.5%) | Res: 0 (0.0%) | Hist: 245.0k (97.3%) | Act: 1.5k (0.6%)│
╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

---

### 2. 半螢幕模式 ($< 100$ 欄，如 75~80 欄) - 緊湊適配零溢出

```text
╭─────────────────────────────────────────────────────────────────────────────╮
│ SESSION TOKEN AGGREGATES & MULTI-MODEL EFFICIENCY                           │
│                                                                             │
│   TOTAL PROCESSED                     CACHE HIT VOLUME                      │
│   1302.22M Tok (7912 Turns)           696.51M Tok (53.5% Hit)               │
│                                                                             │
│   UNCACHED INBOUND                    TOKENS SAVED (%)                      │
│   605.71M Tok (46.5% Cold)            522.39M Tok (40.1% Saved)             │
│                                                                             │
│  MULTI-MODEL TOKEN & SAVINGS BREAKDOWN:                                     │
│  Model Name          Turns  Processed   Cached (Hit %)   Tokens Saved (%)   │
│  gemini-3.7-flash     5845    916.24M  696.11M (76.0%)    522.08M (57.0%)   │
│  gemini-3.7-flash-sa  2065    385.54M        0 ( 0.0%)          0 ( 0.0%)   │
│  gemini-3.7-flash-hi     2     443.1k   401.6k (90.6%)     301.2k (68.0%)   │
│  ────────────────────────────────────────────────────────────────────────   │
│  TOTAL SUMMARY        7912   1302.22M  696.51M (53.5%)    522.39M (40.1%)   │
╰─────────────────────────────────────────────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────────────────────╮
│ TRACK 1: OFFICIAL TELEMETRY & STEP CONTEXT ANATOMY                          │
│   • Backend Model   : gemini-3.7-flash-high  (Step #5144)                   │
│   • Active Context  : 251.7k Tokens (98.3% of 256k) [CACHE HIT 99.4%]       │
│   • Status & Time   : Status: DONE | 2026-08-27 17:01:46                    │
│   • Context Anatomy :                                                       │
│     Sys: 3.8k (1.5%) | Tools: 1.4k (0.5%) | Res: 0 (0.0%)                   │
│     Hist: 245.0k (97.3%) | Act: 1.5k (0.6%)                                 │
╰─────────────────────────────────────────────────────────────────────────────╯
```

---

## 🧮 三、多模型計費與折扣矩陣演算法 (Per-Model Accounting Engine)

### 1. 官方計費折扣對照表 (`internal/core/stats.go`)

| 模型系列 (Model Family) | 快取折扣率 ($\text{Discount}$) | 計費係數 ($\text{PriceFactor}$) | 標籤 (`DiscountLabel`) |
| :--- | :--- | :--- | :--- |
| **Claude 系列** (`claude-3-7-sonnet`, `claude-3-5-sonnet`) | 90% OFF ($0.90$) | $0.10\times$ | `0.10x` |
| **Gemini 系列** (`gemini-3.7-flash`, `gemini-2.5-pro`) | 75% OFF ($0.75$) | $0.25\times$ | `0.25x` |
| **OpenAI 系列** (`gpt-4o`, `o1`, `o3-mini`) | 50% OFF ($0.50$) | $0.50\times$ | `0.50x` |
| **DeepSeek 系列** (`deepseek-r1`, `deepseek-v3`) | 90% OFF ($0.90$) | $0.10\times$ | `0.10x` |

### 2. 核心計算公式
對於任意模型 $M$ 與第 $i$ 次調用事件：
$$\text{EffectiveTokens}_i = \text{NewTokens}_i + (\text{CachedTokens}_i \times \text{PriceFactor}_M)$$
$$\text{TokensSaved}_i = \text{TotalTokens}_i - \text{EffectiveTokens}_i = \text{CachedTokens}_i \times \text{DiscountRate}_M$$
$$\text{SavingsPercentage}_M = \frac{\sum \text{TokensSaved}}{\sum \text{TotalProcessed}} \times 100\%$$

---

## 💻 四、程式碼實作結構對照

1. **`internal/core/types.go`**：
   * `ModelTokenStats`：單一模型的輪次、處理量、快取量、等效 Token 與節省量結構體；
   * `SessionAggregateMetrics`：包含 `TotalStats` 與 `ModelStats []ModelTokenStats`；
   * `TurnTrendPoint` & `TurnTrendSeries`：多輪遙測趨勢點資料結構。
2. **`internal/core/stats.go`**：
   * `GetModelDiscount(modelName string)`：模型折扣查詢函式；
   * `ComputeSessionAggregateMetrics(history []UnifiedAgentEvent)`：全會話多模型聚合計算；
   * `ExtractTurnTrendSeries(history []UnifiedAgentEvent, maxPoints int)`：趨勢折線資料抽樣。
3. **`internal/ui/sparkline.go`**：
   * `RenderSparkline(values []float64, maxVal float64, width int, style lipgloss.Style)`：8 階 Unicode Block 折線圖生成器。
4. **`internal/ui/views.go`**：
   * `renderBorderlessKpiStrip(tot core.ModelTokenStats, width int)`：雙模響應式 KPI 欄位條；
   * `renderModelBreakdownTable(models []core.ModelTokenStats, total core.ModelTokenStats, width int)`：雙模響應式多模型表格；
   * `renderDashboardView()`：組裝 Panel 0（統計矩陣）與 Panel 1（遙測與五維結構）。

---

## 🧪 五、驗收與測試指標 (Verification Metrics)

* ✅ **全單元測試 100% 通過**：
  * `TestGetModelDiscount`
  * `TestComputeSessionAggregateMetrics`
  * `TestExtractTurnTrendSeries`
  * `TestDashboardSparklinesAndKpiRendering`
  * `TestDashboardHalfWidthResponsiveRendering`
  * `TestCJKAndLongPayloadZeroHeightVariation`
* ✅ **多解析度高度與寬度守恆**：在 80x24、100x30、120x35、140x40 終端下 100% 遵守 $H$ 守恆，半螢幕下零溢出。
