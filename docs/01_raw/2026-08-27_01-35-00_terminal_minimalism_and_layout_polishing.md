# 極簡無 Emoji 終端工程美學與 Header / Recent Events 佈局重構

- **建立時間**: 2026-08-27 01:35:00
- **更新時間**: 2026-08-27 01:35:00
- **模組歸屬**: `02_architecture`
- **狀態**: `RAW_INTAKE` (待 wiki-distiller 二次提煉)

---

## 🎯 終端機極簡設計哲學 (Terminal Minimalism)

在現代 CLI / TUI 開發中，過多的圖標與 Emoji（例如 📊、⏱️、🧠、🚀、📜、🔍）雖然在 Web 上看似活潑，但在專業終端機環境（如 Alacritty、iTerm2、tmux）中往往會帶來諸多排版與視覺負擔：
1. **寬度計算失真**：部分終端機將 Emoji 視為 1 欄寬，部分視為 2 欄寬，極易引發不同終端下的跳行與邊框破裂；
2. **視覺噪點干擾**：過度花俏的 Emoji 會分散工程師對核心度量數據（Tokens、Hit Rate、Latency）的注意力；
3. **字體相容性**：在無 Emoji 字體的伺服器或 SSH 環境下容易出現亂碼方框（Tofu blocks）。

因此，我們貫徹了「**純粹 ASCII / ANSI 專業終端美學**」，全面重構視覺排版。

---

## 🎨 佈局優化一：頂部 Header 的時序遞進排版

### 1. 演進前 vs 演進後

* **早期版本（視覺繁雜、Tab 與控制混淆）**：
  ```text
  AGENT-OBSERVER   [1] Dashboard  [2] History  [3] Docs  [Ctrl+P] aa726359...     Events: 2200 | 01:14:31
  ```
  * 缺點：`[Ctrl+P]` 夾在 Tab 之間容易誤以為是第 4 個分頁；省略號 `...` 顯得不夠精練；右側資訊未對稱。

* **最終重構版本（時序自然遞進、兩端平衡）**：
  ```text
  AGENT-OBSERVER   [1] Dashboard  [2] History  [3] Docs                     01:23:20 | Events: 2200 | (aa726359)
  ```
  * **左側**：純粹的視圖切換 Tab 列（`[1] Dashboard  [2] History  [3] Docs`）；
  * **右側**：以時序因果自然遞進排列 —— **當前時間** $\to$ **累積事件總數** $\to$ **當前會話短 Hash** `(aa726359)`；
  * 徹底消除省略號，右側 Hash 與下方圓角外框最右邊緣（第 $m.width$ 列）**100% 垂直絕對對齊**。

---

## 📦 佈局優化二：Recent Live Events 內容自適應 (Content-Hugging)

### 1. 痛點與改進
* **舊有機制**：在 Live Dashboard 視圖中，Panel 3（Recent Live Events）過去強制計算剩餘行數並填滿螢幕底端，在只有 1~2 筆事件或超大螢幕（如 50 行）時，會在框內產生大量無意義的空白行；
* **新有機制（Content-Hugging）**：
  1. **硬性上限 6 筆**：最多呈現最新的 6 筆即時事件，確保資訊密度精練；
  2. **自適應內容包覆**：外框高度僅包覆實際事件數（標題 1 行 + $N$ 筆事件 + 2 行邊框），高度不再強行撐滿底端；
  3. **視圖全域定高**：在 `Model.View()` 統一以乾淨的終端底色補齊行數，使最底部的快捷提示列（Footer）永遠穩穩吸附在螢幕最底行。

```text
╭───────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ TRACK 1: OFFICIAL GEMINI TELEMETRY (BILLING GROUND TRUTH)                                                 │
│   • Backend Model         : gemini-3.7-flash-high (Official API)                                          │
│   • Total Active Context  : 130360 Tokens ( 50.9% of 256k Window)                                         │
│   • Prefix Cache Hit      : 109565 Tokens ( 84.0%)  [CACHE HIT 84.0%]                                     │
│   • New Billable Tokens   : 20795 Tokens ( 16.0%)                                                         │
│   • Response / Event Time : 2026-08-27 00:34:18  (Step #1957 | Status: DONE)                             │
╰───────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭───────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ TRACK 2: LOCAL 5-DIMENSION CONTEXT ANATOMY (PAYLOAD ANALYSIS)                                             │
│   1. System Instruction : 3806     Tokens (  2.9%)  [████░░░░░░░░░░░]                                     │
│   2. MCP Tools Schema   : 1377     Tokens (  1.1%)  [██░░░░░░░░░░░░░]                                     │
│   3. Tool Results / Diff: 0        Tokens (  0.0%)  [░░░░░░░░░░░░░░░]                                     │
│   4. Conversation Hist  : 125127   Tokens ( 96.0%)  [██████████████░]                                     │
│   5. Active Turn / CoT  : 50       Tokens (  0.0%)  [░░░░░░░░░░░░░░░]                                     │
╰───────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭───────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ RECENT LIVE EVENTS (Press [Enter] or [2] to inspect history)                                              │
│   [1954|TOOL ] 16:34:00  Search database schema                                                           │
│   [1955|CMD  ] 16:34:02  Querying SQLite WAL metadata                                                     │
│   [1956|MODEL] 16:34:05  Generating payload telemetry diff                                                │
│   [1957|TOOL ] 16:34:15  Run discovery test                                                               │
│   [1958|CMD  ] 16:34:18  Running discovery unit tests                                                     │
│   [1959|MODEL] 16:34:20  Telemetry sync completed                                                         │
╰───────────────────────────────────────────────────────────────────────────────────────────────────────────╯


 [Enter] Inspect  [j/k] Playback  [?] Shortcuts  [3/h] Docs  [Ctrl+p] Switch  [G] LIVE  [q] Quit
```
