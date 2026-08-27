# 📊 規格企劃：Dashboard 全會話累計統計、等效計費度量與終端多軌 Unicode 趨勢圖 (Dashboard Aggregates & Multi-Turn Trend Charts Plan)

> **建立時間**：2026-08-27 15:50:00  
> **狀態**：PROPOSED & READY FOR IMPLEMENTATION  
> **目標模組**：`agent-observer/internal/core/` & `agent-observer/internal/ui/`  
> **關聯概念**：[[02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting]]、[[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics]]、[[01_theory/03_Prompt_Caching_Lifecycle]]  

---

## 🔍 一、需求背景與痛點剖析 (Motivation & Problem Statement)

在目前 `agent-observer` 的 `[1] Dashboard` 視圖中，主要聚焦於**「當前最新單一步驟（Latest Step）」的即時遙測與 5 維度切片**。
然而，在長程 AI Agent 實戰中，軟體工程師與技術主管面臨著三大宏觀問題：

1. **會話總量與整體成本盲區 (Session-Wide Cost Blindness)**：
   * 單一步驟雖然顯示當前上下文為 199k Tokens，但整場對話經歷了 50~100 次工具呼叫與推論，**累積到底總共向雲端發送了多少 Tokens？總共被快取保護了多少？又實際被計費了多少？**
2. **快取折扣與等效計費估算缺失 (Effective Billing & ROI Calculation)**：
   * 各家大模型對 Prompt Cache 提供了大幅度的價格折扣（如 Google Gemini 享 75% 折扣，即 $0.25\times$ 價格；Anthropic Claude 享 90% 折扣，即 $0.10\times$ 價格）。
   * 若僅看 Token 總數，無法直觀得出「到底替團隊節省了多少錢（Net Cost Savings）與等效計費 Token 總量」。
3. **多輪次上下文演化與快取斷裂趨勢不可視 (Multi-Turn Trend Visualizer)**：
   * 無法一目了然看見整場會話的上下文增長曲線（Context Expansion）、快取命中維持率（Cache Stability）以及突發的大型 Tool Result 注入點（New Token Spikes）。

---

## 📐 二、數學模型與計費經濟學公式 (Mathematical & Economic Formulas)

### 1. 全會話累計指標 (Cumulative Session Aggregates)
針對整場會話中所有涉及雲端 LLM 推論的步驟集合 $\mathcal{S}_{\text{cloud}} = \{e \in \text{History} \mid e.\text{Scope} == \text{CLOUD} \lor e.\text{IsCloudStep}()\}$：

$$\text{Total Processed Tokens} = \sum_{i \in \mathcal{S}_{\text{cloud}}} \text{Tokens}_i.\text{TotalTokens}$$

$$\text{Total Cached Volume} = \sum_{i \in \mathcal{S}_{\text{cloud}}} \text{Tokens}_i.\text{CachedTokens}$$

$$\text{Total Cold Inbound (New)} = \sum_{i \in \mathcal{S}_{\text{cloud}}} \text{Tokens}_i.\text{NewTokens}$$

$$\text{Cumulative Cache Hit Rate} = \frac{\text{Total Cached Volume}}{\text{Total Processed Tokens}} \times 100\%$$

---

### 2. 官方模型階梯折扣與等效計費 (Effective Billed Tokens Model)
依據各主流大模型廠商的官方 Prompt Caching 定價模型，定義快取折扣係數 $D_{\text{model}} \in [0.0, 1.0]$：

| 模型廠商 / 家族 | 快取折扣率 (Discount) | 計費乘數 ($D_{\text{model}}$) | 說明與標準定價依據 |
| :--- | :--- | :--- | :--- |
| **Google Gemini 2.5 / 3.7** | **75% OFF** | **$0.25\times$** | 快取 Token 僅以標準輸入價格的 25% 計費 |
| **Anthropic Claude 3.5 / 3.7** | **90% OFF** | **$0.10\times$** | 快取 Token 僅以標準輸入價格的 10% 計費 |
| **OpenAI GPT-4o / o1** | **50% OFF** | **$0.50\times$** | 快取 Token 僅以標準輸入價格的 50% 計費 |
| **Local / Ollama** | **100% OFF** | **$0.00\times$** | 地端運算，零外部計費 |

#### 🧮 等效計費 Token 總量 (Effective Billed Tokens)：
$$\text{Effective Tokens} = \text{Total Cold Inbound} + (\text{Total Cached Volume} \times D_{\text{model}})$$

#### 💰 實質節省量與節省比率 (Net Token & Cost Savings)：
$$\text{Tokens Saved} = \text{Total Cached Volume} \times (1 - D_{\text{model}})$$

$$\text{Net Cost Savings Ratio} = \frac{\text{Tokens Saved}}{\text{Total Processed Tokens}} \times 100\% = \text{Cumulative Hit Rate} \times (1 - D_{\text{model}})$$

#### 💡 實機案例驗算 (以 128 輪對話為例)：
* 總處理量：$2,450,120$ Tokens
* 總快取量：$2,210,000$ Tokens ($90.20\%$ 命中率)
* 總冷輸入：$240,120$ Tokens
* 若使用 Gemini ($D = 0.25$)：
  * $\text{Effective Tokens} = 240,120 + (2,210,000 \times 0.25) = 240,120 + 552,500 = \mathbf{792,620\text{ Tokens}}$
  * $\text{Tokens Saved} = 2,210,000 \times 0.75 = \mathbf{1,657,500\text{ Tokens}}$
  * $\text{Net Cost Savings Ratio} = \frac{1,657,500}{2,450,120} \times 100\% = \mathbf{67.65\%}$（實質帳單總額直接打 3.2 折！）

---

## 📈 三、終端機高密度 Unicode 趨勢圖引擎 (Sparkline Engine Architecture)

### 1. 8 階次像素 Unicode Block 映射 (The 8-Level Block Quantization)
終端機不支援傳統 GUI 的 SVG/Canvas 畫布，但 UTF-8 提供了精確的 8 階垂直等寬字元：

| 階數 (Tier) | Unicode 字元 | 數值區間比例 ($\text{val} / \text{max}$) | 視覺表現 |
| :---: | :---: | :---: | :---: |
| 0 | ` ` (Space) | $0.00 \sim 0.05$ | 底部無高度 |
| 1 | ` ` (U+2581) | $0.05 \sim 0.17$ | $1/8$ 高度 |
| 2 | `▂` (U+2582) | $0.17 \sim 0.29$ | $2/8$ 高度 |
| 3 | `▃` (U+2583) | $0.29 \sim 0.42$ | $3/8$ 高度 |
| 4 | `▄` (U+2584) | $0.42 \sim 0.54$ | $4/8$ 高度 |
| 5 | `▅` (U+2585) | $0.54 \sim 0.67$ | $5/8$ 高度 |
| 6 | `▆` (U+2586) | $0.67 \sim 0.79$ | $6/8$ 高度 |
| 7 | `▇` (U+2587) | $0.79 \sim 0.92$ | $7/8$ 高度 |
| 8 | `█` (U+2588) | $0.92 \sim 1.00$ | $8/8$ 滿格高度 |

### 2. 寬度自適應採樣與壓縮演算法 (Adaptive Series Resampling)
令終端可用繪圖寬度為 $W_{\text{spark}}$（例如 50 欄），會話總雲端輪次為 $N$：
* **當 $N \le W_{\text{spark}}$ 時**：直接以最新點靠右對齊（Right-aligned），左側補空白；
* **當 $N > W_{\text{spark}}$ 時**：提供兩種切換模式：
  1. **尾端滑動窗口 (Latest Window - 預設)**：展示最近 $W_{\text{spark}}$ 次推論輪次的微觀變化；
  2. **最大池化降採樣 (Max-Pooling Resampling)**：將 $N$ 個點等分為 $W_{\text{spark}}$ 個桶（Buckets），每個桶取最大值 $\max(b_k)$，保留尖峰特徵。

### 3. 四軌核心趨勢折線 (The 4 Trend Tracks)
1. **軌道 1：上下文總量曲線 (Context Total - 青色 `ColorSecondary`)**：展示總 Active Context 隨步數的增長與 256k 水位；
2. **軌道 2：快取命中體積 (Cached Volume - 綠色 `ColorSuccess`)**：展示 GPU HBM 中維持的快取規模；
3. **軌道 3：新冷輸入尖峰 (New Inbound Spikes - 橙色 `ColorHighlight`)**：直觀抓出哪一步讀入了大型檔案或大量 Diff；
4. **軌道 4：命中率百分比 (Hit Rate % - 萊姆綠 `ColorSuccess`)**：直觀反映快取是否發生斷裂（`MISS`）或過期（`EXPIRED`）。

---

## 🖥️ 四、Dashboard 終端佈局與 $H$ 守恆盒模型 (UI Layout & Budgeting)

```text
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 📊 SESSION AGGREGATE TOKEN METRICS & COST EFFICIENCY                                                                 │
│   • Total Tokens Processed : 2,450,120 Tokens (128 Cloud Turns)                                                      │
│   • Prefix Cache Hit Volume: 2,210,000 Tokens ( 90.2% Cumulative Hit Rate)                                           │
│   • New / Uncached Tokens  :   240,120 Tokens (  9.8% Cold Inbound)                                                  │
│   • Effective Billed Tokens:   792,620 Tokens [75% Gemini Cache Discount: 2.21M * 0.25 + 240k]                       │
│   • Net Cost / Token Saving: 1,657,500 Tokens Saved ( 67.6% Net Cost Savings vs Non-Cached)                          │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 📈 MULTI-TURN CONTEXT & CACHE HIT TREND (Last 54 Cloud Turns)                                                        │
│   Context Total (Cyan) :   ▂▃▄▅▆▇████████████████████████████████████████████████ [Peak: 219k]                       │
│   Cached Volume (Green):   ▂▃▄▅▆▇████████████████████████████████████████████████ [Curr: 199k]                       │
│   New Input     (Orange):  ▄█▂ ▃ ▂ ▅ ▂  ▂ ▃ ▂ ▂  ▂ ▃ ▂  ▂ ▂  ▂ ▃ ▂ ▂ ▂ ▂ ▂ ▂ ▂ ▂  [Avg: 1.8k]                       │
│   Hit Rate %    (Lime) :  ▂▅▆▇██████████████████████████████████████████████████ [Avg: 90.2%]                       │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 🔍 TRACK 1: LATEST STEP TELEMETRY & CONTEXT ANATOMY (Step #4546)                                                     │
│   • Backend Model         : gemini-3.7-flash-high                                                                    │
│   • Active Step Context   : 199,280 Tokens ( 77.8% of 256k Window)  [CACHE HIT 90.6%]                                │
│   • 5-Dimension Breakdown : Sys: 3.8k (1.9%) | Tools: 1.4k (0.7%) | Hist: 194.1k (97.4%) | Active: 50 (0.0%)         │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 📐 響應式垂直行數守恆預算表 (Height Responsive Allocations)：
* **小螢幕 ($H < 28$)**：展示 Panel 0（統計總覽 6 行）+ Panel 1（當前步驟 5 行），隱藏趨勢圖；
* **標準螢幕 ($28 \le H < 36$)**：展示 Panel 0（統計總覽 6 行）+ Panel 0.5（趨勢圖 5 行）+ Panel 1（當前步驟 6 行）；
* **大螢幕 ($H \ge 36$)**：完整展示 Panel 0（統計 6 行）+ Panel 0.5（趨勢圖 5 行）+ Track 1（官方 7 行）+ Track 2（5 維度 7 行）+ Recent Events（6 行）。

---

## 🛠️ 五、代碼實作計畫 (Implementation Architecture)

### 1. 領域層 (`agent-observer/internal/core/`)
* **`types.go`**：
  * 新增 `SessionTokenStats` 結構體；
  * 新增 `TurnTrendPoint` 與 `TurnTrendSeries` 結構體。
* **`stats.go` [NEW]**：
  * 實作 `ComputeSessionTokenStats(history []UnifiedAgentEvent, discountRate float64) SessionTokenStats`；
  * 實作 `ExtractTurnTrendSeries(history []UnifiedAgentEvent, maxPoints int) TurnTrendSeries`。
* **`stats_test.go` [NEW]**：
  * 針對真實 4,500 步歷史資料進行單元測試，驗證累計數值、等效計費公式與降採樣長度一致性。

### 2. UI 呈現層 (`agent-observer/internal/ui/`)
* **`sparkline.go` [NEW]**：
  * 實作 `RenderSparkline(values []float64, maxVal float64, width int, style lipgloss.Style) string`；
  * 實作 Unicode 8 階字元對齊與色彩渲染。
* **`views.go`**：
  * 重構 `renderDashboardView()`，置入頂部統計總覽面板與趨勢圖面板；
  * 導入動態高度適配，嚴格遵守行數守恆公式。
* **`tui_test.go`**：
  * 擴充 `TestDashboardSparklineRendering` 與多解析度零抖動回歸測試。

---

## 🧪 六、驗收與驗證計畫 (Verification Strategy)

### 1. 自動化測試：
```bash
go test -v ./internal/core -run TestComputeSessionTokenStats
go test -v ./internal/ui -run TestDashboardSparklineRendering
go test -v ./internal/ui -run TestAllViewsZeroHeightVariationAcrossSizes
```

### 2. 實機驗證（真實 4,500 步會話）：
```bash
go build -o bin/agent-observer main.go
./bin/agent-observer
```
* 檢查 `[1] Dashboard` 頂部是否精確顯示全會話總 Tokens、75% 快取折扣等效計費與節省量；
* 檢查 4 軌 Sparkline 是否隨視窗寬度自適應延伸，顏色層次分明；
* 縮放終端視窗（80x24, 100x30, 120x35, 140x40），驗證排版零抖動、零溢出。
