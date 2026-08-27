# 🌲 步驟清單因果逆轉與零縮排括號連接器架構規範 (Bottom-Up Causality & Bracket Connectors)

> **文件類型**：UI Architecture & Causality Topology Specification  
> **日期**：2026-08-27  
> **路徑**：`docs/01_raw/2026-08-27_150200_bottom_up_causality_and_bracket_connector_layout_plan.md`  
> **狀態**：📝 PROPOSED & READY FOR IMPLEMENTATION  

---

## 🎯 一、 設計動機與視覺哲學

### 1. 傳統水平縮排的痛點
傳統檔案樹或對話樹習慣在子項目左側加入 2~4 個空白縮排（例如 `  └── ...`）。但在終端機雙欄/多欄佈局中，左側側邊欄通常只有 38 欄寬度：
* 額外的縮排直接吞噬了珍貴的寬度，導致模型名稱（`Gemini 3.7 Flash`）或工具名稱被迫截斷。
* 無法直觀表達時間遞進與因果關聯。

### 2. 時間流向與「因下果上」物理本質
* 在歷史清單中，**時間是由下往上（由舊到新）流動**。
* **因（Cause / Inbound Input）**：
  * 使用者輸入（`👤 USER`）或本地工具輸出（`💻 OUTPUT`）在時間上先發生，物理位置處於**下方**。
* **果（Effect / Cloud Result）**：
  * 雲端模型決策（`🛠️ TOOL`）或最終回覆（`🤖 MODEL`）在時間上後發生，物理位置處於**上方**。

### 3. 零縮排括號連接器 (`┌──` / `└──`)
* **不浪費任何左側空間（零縮排）**，所有卡片齊頭排列。
* 利用上下成對的括號連接符號，將一個因果輪次（Inbound Cause $\to$ Cloud Effect）完美成組綁定：
  * 上層（果）：`┌── [4446] 🛠️ TOOL [HIT 100%]`
  * 中間（線）：`│   Model: Gemini 3.7 Flash`
  * 下層（因）：`└── [4445] 💻 OUTPUT (Local)`

---

## 📐 二、 視覺效果前後對比

### Before（舊有水平縮排模式）
```text
╭────────────────────────────────────╮
│ STEPS (6054) <                     │
│ Filters: [T:All] [C:All]           │
│   ...                              │
│ > [4446] 🛠️ TOOL [HIT 100%]         │  <-- 無連接符號
│     Model: Gemini 3.7 Flash (High) │
│   └── [4445] 💻 OUTPUT (Local)     │  <-- 浪費左側 2 格縮排
│         Tool: edit_file            │
│   [4444] 🛠️ TOOL [HIT 100%]         │
│     Model: Gemini 3.7 Flash (High) │
│   └── [4443] 💻 OUTPUT (Local)     │
│         Tool: view_file            │
│   [4440] 🤖 MODEL [HIT 100%]       │
│     Model: Gemini 3.7 Flash (High) │
│   [4433] 👤 USER                   │
╰────────────────────────────────────╯
```

### After（因下果上 ＋ 單橫線連續括號封裝模式）
```text
╭────────────────────────────────────╮
│ STEPS (6054) <                     │
│ Filters: [T:All] [C:All]           │
│   ...                              │
│ ┌─ [4446] 🛠️ TOOL [HIT 100%]        │  <-- 果 (Cloud Effect: Top of bracket)
│ │     Model: Gemini 3.7 Flash      │  <-- Cloud Hint (Vertical stem)
│ │  [4445] 💻 OUTPUT (Local)       │  <-- 因 (Local Cause: Inside stem, [ aligned)
│ └─   Tool: edit_file               │  <-- Local Hint (Bottom of bracket)
│ ┌─ [4444] 🛠️ TOOL [HIT 100%]        │  <-- 果 (Cloud Effect: Top of bracket)
│ │     Model: Gemini 3.7 Flash      │  <-- Cloud Hint (Vertical stem)
│ │  [4443] 💻 OUTPUT (Local)       │  <-- 因 (Local Cause: Inside stem, [ aligned)
│ └─   Tool: view_file               │  <-- Local Hint (Bottom of bracket)
│ ┌─ [4440] 🤖 MODEL [HIT 100%]       │  <-- 果 (Cloud Effect: Top of bracket)
│ │     Model: Gemini 3.7 Flash      │  <-- Cloud Hint (Vertical stem)
│ └─ [4433] 👤 USER                  │  <-- 因 (Local Cause: Bottom of bracket, [ aligned)
╰────────────────────────────────────╯
```

---

## 🏛️ 三、 連接器配對判定演算法

在時序清單中（由下至上為舊至新）：
1. **上層雲端步驟 (`MODEL_RESPONSE` / `TOOL_CALL`)**：
   * 若其下方緊鄰之步驟為本地輸入步驟（`USER_INPUT` 或 `OUTPUT`），則冠以 `┌── `（或選中時為 `> ┌─ `）。
   * 若有第 2 行 Hint（如 `Model: ...`），則以 `│   ` 連接。
2. **下層本地步驟 (`USER_INPUT` / `OUTPUT`)**：
   * 若其上方緊鄰之步驟為消費該輸入之雲端步驟，則冠以 `└── `（或選中時為 `> └─ `）。
   * 移除所有 `  ` 縮排。
3. **獨立步驟 (`CHECKPOINT`, `SYSTEM_INIT`, 尚未有回覆的 Prompt)**：
   * 冠以 `    ` 或 `• `，保持齊頭美感。
