# 📊 規格企劃：Dashboard 全會話統計視覺方案重構、多模型等效計費矩陣與多軌趨勢圖 (v2.0)

> **建立時間**：2026-08-27 16:25:00  
> **狀態**：PROPOSED & READY FOR USER SELECTION  
> **目標模組**：`agent-observer/internal/core/` & `agent-observer/internal/ui/`  
> **關聯概念**：[[02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting]]、[[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics]]  

---

## 🌟 一、問題檢討與重構目標 (Critique & Objectives)

針對原先「純條列式（Bulleted List）」缺乏視覺張力、資訊密度偏低、且未支援「多模型個別等效計費」的問題，本次規劃提出 **三大終端機視覺排版方案**，並將資料模型升級為 **「按模型維度分組統計 (Per-Model Breakdown Matrix)」**：

1. **告別平淡條列，導入高資訊密度現代視覺**（如 KPI 卡片方塊、結構化網格表格、微型比例長條圖）；
2. **多模型分別統計與等效計費 (Per-Model Economics)**：
   * 會話中可能同時使用 `gemini-3.7-flash`（主力）、`gemini-2.5-pro`（複雜任務）或 Subagent 模型；
   * 分別計算各模型的：**調用輪次 (Turns)、總處理量 (Processed)、快取體積 (Cached)、未快取量 (Uncached)、個別快取折扣 (Discount Rate)、個別等效計費 (Effective Tokens) 與實質節省量 (Net Saved)**；
3. **終端多軌 Unicode 趨勢圖 (Trend Sparklines)**：持續保留 4 軌次像素趨勢折線。

---

## 🎨 二、三大視覺排版候選方案 (3 Visual Layout Options)

---

### 🅰️ 方案 A：現代多欄 KPI 卡片方塊 ＋ 多模型明細表格（推薦）
> **特點**：上方 5 顆大字號 KPI 圓角方塊（視覺衝擊力強，一目了然），下方搭配多模型成本明細表格。

```text
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 📊 SESSION TOKEN AGGREGATES & MULTI-MODEL COST EFFICIENCY                                                            │
│                                                                                                                      │
│  ┌─ TOTAL PROCESSED ─┐  ┌─ CACHE HIT VOLUME ─┐  ┌─ UNCACHED INBOUND ─┐  ┌─ EFFECTIVE BILLED ─┐  ┌─ NET SAVINGS / ROI ┐ │
│  │   2,450,120 Tok   │  │   2,210,000 Tok    │  │    240,120 Tok     │  │    792,620 Tok     │  │   1,657,500 Saved  │ │
│  │   128 Cloud Turns │  │   90.2% Hit Rate   │  │    9.8% Cold       │  │    32.3% of Raw    │  │   67.7% OFF 💰     │ │
│  └───────────────────┘  └────────────────────┘  └────────────────────┘  └────────────────────┘  └────────────────────┘ │
│                                                                                                                      │
│  MODEL COST & SAVINGS BREAKDOWN:                                                                                     │
│  Model Name                Turns   Processed     Cached (Hit %)       Uncached     Effective (Discount)   Net Saved  │
│  gemini-3.7-flash-high       120   2,250,120   2,050,000 (91.1%)       200,120      712,620 (75% OFF)     1.54M (68%)│
│  gemini-2.5-pro                8     200,000     160,000 (80.0%)        40,000       80,000 (75% OFF)      120k (60%)│
│  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│  TOTAL SUMMARY               128   2,450,120   2,210,000 (90.2%)       240,120      792,620 (75% OFF)     1.66M (68%)│
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

---

### 🅱️ 方案 B：緊湊網格表格 ＋ 內嵌快取比例量表（精簡緊湊）
> **特點**：以一體化數據網格為主體，直接在表格內嵌入色彩量表（Green/Orange Bar），兼具緊湊度與專業感。

```text
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 📊 TOKEN ECONOMICS & MULTI-MODEL BREAKDOWN                                                                           │
│                                                                                                                      │
│  MODEL / BACKEND           TURNS     PROCESSED       CACHED TOKENS       UNCACHED     EFFECTIVE BILLED   NET SAVINGS │
│  gemini-3.7-flash-high       120     2.25M Tok    2.05M [█████████░] 91%    200k Tok     713k (0.25x)    1.54M (68%) │
│  gemini-2.5-pro                8      200k Tok     160k [████████░░] 80%     40k Tok      80k (0.25x)     120k (60%) │
│  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────── │
│  TOTAL (ALL MODELS)          128     2.45M Tok    2.21M [█████████░] 90%    240k Tok     793k (0.25x)    1.66M (68%) │
│                                                                                                                      │
│  💰 Financial Summary: 792.6k Effective Billed vs 2.45M Raw Context (67.7% Net Savings via Cloud Prompt Caching)     │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

---

### 🅲 方案 C：雙欄財務看板（左側大字號財務總結 ＋ 右側模型明細矩陣）
> **特點**：左側以獨立的高亮總覽框凸顯財務 ROI，右側展開多模型明細，適合寬螢幕掃描。

```text
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 📊 SESSION TOKEN ECONOMICS & MULTI-MODEL COST ANALYSIS                                                               │
│                                                                                                                      │
│  ┌─ OVERALL COST SAVINGS ─────────┐  MODEL BREAKDOWN MATRIX:                                                         │
│  │  💰 67.7% Net Cost Saved       │  Model Name            Turns  Processed    Cached   Uncached  Effective (Disc) │
│  │  • Raw Processed : 2.45M Tok   │  gemini-3.7-flash-high   120     2.25M      2.05M     200k      713k (75% OFF) │
│  │  • Cached Volume : 2.21M (90%) │  gemini-2.5-pro            8      200k       160k      40k       80k (75% OFF) │
│  │  • Effective Bill: 793k Tok    │  ─────────────────────────────────────────────────────────────────────────────── │
│  │  • Net Saved     : 1.66M Tok   │  TOTAL (All Models)      128     2.45M      2.21M     240k      793k (68% SAVED│
│  └────────────────────────────────┘                                                                                  │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

---

## 🧮 三、多模型計費與折扣矩陣演算法 (Per-Model Accounting Engine)

### 1. 模型折扣係數對照表 (Model Discount Registry)
```go
func getModelCacheDiscount(modelName string) (discountRate float64, priceFactor float64, label string) {
    name := strings.ToLower(modelName)
    switch {
    case strings.Contains(name, "claude"):
        return 0.90, 0.10, "90% OFF"
    case strings.Contains(name, "gemini"):
        return 0.75, 0.25, "75% OFF"
    case strings.Contains(name, "gpt-4") || strings.Contains(name, "o1") || strings.Contains(name, "o3"):
        return 0.50, 0.50, "50% OFF"
    case strings.Contains(name, "deepseek"):
        return 0.90, 0.10, "90% OFF"
    default:
        return 0.75, 0.25, "75% OFF" // Default Gemini standard
    }
}
```

### 2. 多模型分組彙總結構體 (`internal/core/types.go`)
```go
type ModelTokenStats struct {
    ModelName         string  `json:"model_name"`
    TurnCount         int     `json:"turn_count"`
    TotalProcessed    int     `json:"total_processed"`
    TotalCached       int     `json:"total_cached"`
    TotalNew          int     `json:"total_new"`
    CacheHitRate      float64 `json:"cache_hit_rate"`
    DiscountRate      float64 `json:"discount_rate"` // e.g. 0.75
    PriceFactor       float64 `json:"price_factor"`  // e.g. 0.25
    DiscountLabel     string  `json:"discount_label"`
    EffectiveTokens   int     `json:"effective_tokens"`
    TokensSaved       int     `json:"tokens_saved"`
    SavingsPercentage float64 `json:"savings_percentage"`
}

type SessionAggregateMetrics struct {
    TotalStats  ModelTokenStats   `json:"total_stats"`
    ModelStats  []ModelTokenStats `json:"model_stats"`
}
```

---

## 📈 四、終端多軌趨勢折線圖 (Multi-Turn Trend Sparklines)

與統計面板垂直堆疊，4 軌次像素 Unicode Block 折線圖：

```text
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 📈 MULTI-TURN CONTEXT & CACHE HIT TREND (Last 54 Cloud Turns)                                                        │
│   Context Total (Cyan) :   ▂▃▄▅▆▇████████████████████████████████████████████████ [Peak: 219k]                       │
│   Cached Volume (Green):   ▂▃▄▅▆▇████████████████████████████████████████████████ [Curr: 199k]                       │
│   New Input     (Orange):  ▄█▂ ▃ ▂ ▅ ▂  ▂ ▃ ▂ ▂  ▂ ▃ ▂  ▂ ▂  ▂ ▃ ▂ ▂ ▂ ▂ ▂ ▂ ▂ ▂  [Avg: 1.8k]                       │
│   Hit Rate %    (Lime) :  ▂▅▆▇██████████████████████████████████████████████████ [Avg: 90.2%]                       │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

---

## 🧪 五、驗證與驗收計畫 (Verification Plan)

1. **單元測試 (`internal/core/stats_test.go`)**：
   - 驗證單一會話多模型（例如同時出現 `gemini-3.7-flash` 與 `gemini-2.5-pro`）時的分組正確性；
   - 驗證等效計費公式：$\sum \text{Effective} = \sum (\text{New} + \text{Cached} \times \text{Factor})$；
   - 驗證加總列與個別列 100% 數學一致。
2. **終端機畫面與多尺寸適配 (`internal/ui/tui_test.go`)**：
   - 測試 80 欄、100 欄、120 欄、140 欄下的表格對齊與截斷，確保零換行與零抖動。
