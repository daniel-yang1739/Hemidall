# 💰 企劃書：Dashboard 即時金額換算、多模型計價矩陣與快捷鍵切換 (`$`) 實裝企劃

> **建立時間**：2026-08-27 23:59:00  
> **狀態**：RAW IMPLEMENTATION PLAN (待審核)  
> **目標套件**：`agent-observer/internal/core/`、`agent-observer/internal/ui/`  
> **目標版本**：`v0.9.0`

---

## 🎯 一、目標描述 (Goal Description)

為 `agent-observer` 增加實質財務與金額可視化能力：
1. **多模型官方價格字典 (`pricing.go`)**：維護集中式、可擴充的模型價格體系（支援 Gemini Flash/Pro、Claude Sonnet/Opus、OpenAI GPT-4o、DeepSeek V3/R1），支援每百萬字（$/1M Tokens）的 `Input`、`Cached Input`（享 75%~90% 折扣）與 `Output`（約為 Input 的 4~5 倍價格）獨立計價。
2. **產出 Token (Output Tokens) 獨立結算**：將 Prompt 輸入與模型生成輸出（包含思考過程 Thinking）拆分結算，精準反映產出 Token 較高之單價（如 $0.40/1M vs $0.10/1M）。
3. **Dashboard 互動式單位切換快捷鍵 (`$` / `m`)**：在 View 1 儀表板按下 `$` 或 `m` 鍵，可在 **Tokens (`Tok`)**、**美金 (`$`)**、**新台幣 (`NT$`)** 三種模式間無縫循環切換，頂部 5 大 KPI 與多模型表格自動自適應重繪。

---

## 🔍 二、當前會話花費精算 (Session aa726359 實時結算)

根據目前本會話的物理遙測數據：
* **總處理字數 (Total Processed)**：$250.42\text{M}$ Tokens
* **快取命中量 (Cached Prefix)**：$245.40\text{M}$ Tokens ($98.0\%$ 命中率)
* **冷啟動新進字數 (Uncached Inbound)**：$5.03\text{M}$ Tokens ($2.0\%$)
* **模型生成輸出字數 (Generated Output)**：約 $420\text{k}$ Tokens

### 💵 方案一：Gemini 2.0 / 3.7 Flash 計價標準
| 計費項目 | 計算公式 | 美金費用 (USD) | 台幣折算 (1:32) |
| :--- | :--- | :--- | :--- |
| **冷啟動輸入 (Uncached)** | $5.03\text{M} \times \$0.10/\text{M}$ | $\$0.503$ | $\text{NT\$} 16.1$ |
| **快取命中輸入 (75% OFF)** | $245.40\text{M} \times \$0.025/\text{M}$ | $\$6.135$ | $\text{NT\$} 196.3$ |
| **模型輸出 (Output)** | $0.42\text{M} \times \$0.40/\text{M}$ | $\$0.168$ | $\text{NT\$} 5.4$ |
| **實付帳單 (NET BILLED)** | $\mathbf{\$0.503 + \$6.135 + \$0.168}$ | $\mathbf{\$6.81\text{ USD}}$ | $\mathbf{\text{NT\$} 217.8\text{ 元}}$ |
| **快取幫你省下 (SAVED)** | $245.40\text{M} \times \$0.075/\text{M}$ | $\mathbf{\$18.41\text{ USD}}$ | $\mathbf{\text{NT\$} 589.0\text{ 元}}$ |
| **無快取原始費用** | $250.42\text{M} \times \$0.10 + \$0.168$ | $\$25.21\text{ USD}$ | $\text{NT\$} 806.7\text{ 元}$ |

### 💎 方案二：Gemini 1.5 / 2.0 Pro 計價標準 (若升級 Pro 方案)
| 計費項目 | 計算公式 | 美金費用 (USD) | 台幣折算 (1:32) |
| :--- | :--- | :--- | :--- |
| **冷啟動輸入 (Uncached)** | $5.03\text{M} \times \$1.25/\text{M}$ | $\$6.29$ | $\text{NT\$} 201.2$ |
| **快取命中輸入 (75% OFF)** | $245.40\text{M} \times \$0.3125/\text{M}$ | $\$76.69$ | $\text{NT\$} 2,454.0$ |
| **模型輸出 (Output)** | $0.42\text{M} \times \$5.00/\text{M}$ | $\$2.10$ | $\text{NT\$} 67.2$ |
| **實付帳單 (NET BILLED)** | $\mathbf{\$6.29 + \$76.69 + \$2.10}$ | $\mathbf{\$85.08\text{ USD}}$ | $\mathbf{\text{NT\$} 2,722.6\text{ 元}}$ |
| **快取幫你省下 (SAVED)** | $245.40\text{M} \times \$0.9375/\text{M}$ | $\mathbf{\$230.06\text{ USD}}$ | $\mathbf{\text{NT\$} 7,362.0\text{ 元}}$ |

---

## 🏛️ 三、架構設計全景圖 (Architecture Overview)

```mermaid
flowchart TD
    subgraph CoreEngine ["internal/core/"]
        PricingMatrix["pricing.go\nModelPricing{Input, Cached, Output}"]
        StatsAggregator["stats.go\nComputeSessionAggregateMetrics()"]
        Types["types.go\nModelTokenStats + Cost fields"]
    end

    subgraph UIEngine ["internal/ui/"]
        Model["model.go\nUnitDisplayMode (Tokens / USD / TWD)"]
        Views["views.go\nResponsive KPI Strip & Model Table ($ formatted)"]
        Shortcuts["shortcuts.go / Footer\nAdd '$' toggle hint"]
    end

    PricingMatrix --> StatsAggregator
    Types --> StatsAggregator
    StatsAggregator --> Views
    Model --> Views
```

### 4 維度圖表剖析 (4-Dimension Diagram Walkthrough):
1. **核心視圖 (Core View)**：展示官方價格字典 (`pricing.go`) 計算出金錢開銷，並經由 `UnitDisplayMode` 驅動 TUI 介面進行多幣別切換。
2. **逐步路徑 (Step-by-Step Path)**：事件載入 $\to$ 依模型提取 `Input/Cached/Output` 單價 $\to$ 結算淨費用與省下金額 $\to$ UI 按 `$` 鍵即時無縫重繪。
3. **色彩/物理語義 (Color Semantics)**：藍色核心模組計算精確浮點數金額，粉色 UI 模組處理貨幣符號與 CJK 寬度對齊。
4. **底層工程細節 (Engineering Details)**：所有金額以 `float64` 在 `stats.go` 精算，格式化時小於 `$1.00` 保留 4 位小數（如 `$0.503`），大於 `$1.00` 保留 2 位小數（如 `$6.81`）。

---

## 📂 四、預計變更檔案清單 (Proposed File Changes)

### 1. `internal/core/pricing.go` [NEW]
* 定義 `ModelPrice` 結構體與 `GetModelPricing(modelName string) ModelPrice` 函式：
  - `gemini-3.7-flash`, `gemini-2.0-flash`: Input $0.10, Cached $0.025, Output $0.40
  - `gemini-1.5-pro`, `gemini-2.0-pro`: Input $1.25, Cached $0.3125, Output $5.00
  - `claude-3.7-sonnet`, `claude-3.5-sonnet`: Input $3.00, Cached $0.30, Output $15.00
  - `gpt-4o`: Input $2.50, Cached $1.25, Output $10.00
  - `deepseek-v3`: Input $0.14, Cached $0.014, Output $0.28

### 2. `internal/core/types.go` [MODIFY]
* 在 `ModelTokenStats` 擴充金額欄位：
  ```go
  type ModelTokenStats struct {
      // Token 計數...
      TotalOutputTokens int     `json:"total_output_tokens"`
      
      // 金額指標 (USD)
      RawCostUSD        float64 `json:"raw_cost_usd"`
      NetBilledUSD      float64 `json:"net_billed_usd"`
      CostSavedUSD      float64 `json:"cost_saved_usd"`
      InputCostUSD      float64 `json:"input_cost_usd"`
      OutputCostUSD     float64 `json:"output_cost_usd"`
  }
  ```

### 3. `internal/core/stats.go` [MODIFY]
* 更新 `ComputeSessionAggregateMetrics`，引入 `ModelPrice`，計算每個模型與總結算之各項金額。

### 4. `internal/ui/model.go` [MODIFY]
* 新增 `UnitDisplayMode`（`UnitModeTokens`, `UnitModeUSD`, `UnitModeTWD`）。
* 在 `Update()` 綁定 `$` 與 `m` 鍵循環切換單位。

### 5. `internal/ui/views.go` [MODIFY]
* 更新 `renderBorderlessKpiStrip` 與 `renderModelBreakdownTable`，根據當前單位模式格式化輸出：
  - **Token 模式**：`250.42M Tok`, `245.40M Tok (98.0%)`
  - **美金模式**：`$25.21 原始`, `$6.81 實付`, `$18.41 省下 (73.5%)`
  - **台幣模式**：`NT$ 807 原始`, `NT$ 218 實付`, `NT$ 589 省下 (73.5%)`
* 更新底部 Footer 顯示 `[$] Toggle Units (Tok / USD / NT$)`。

---

## 🧪 五、驗證計畫 (Verification Plan)

### 自動化測試
```bash
go test -v -run TestGetModelPricing ./internal/core
go test -v -run TestComputeSessionAggregateMetrics ./internal/core
go test -v -run TestDashboardUnitModeToggle ./internal/ui
go test -v ./...
```

### 手動驗收
1. 啟動 `./bin/agent-observer`；
2. 在 View 1 (Dashboard) 按下 `$` 或 `m` 鍵；
3. 觀察 5 大 KPI 卡片與多模型表格在 **Tokens ➔ 美金 ($) ➔ 新台幣 (NT$)** 之間順暢切換，數值精確且版面無任何破版。
