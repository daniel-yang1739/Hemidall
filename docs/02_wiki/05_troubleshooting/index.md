---
title: 實戰故障排查、QA 問答與 Runbook 手冊 (Troubleshooting & Runbooks)
type: index
created: 2026-08-27
updated: 2026-08-27
status: completed
tags: [troubleshooting, runbook, postmortem, rca, index]
aliases: [Troubleshooting MOC, 排查手冊索引, 故障覆盤中樞]
---

# 🛠️ 實戰故障排查、QA 問答與 Runbook 手冊 (Troubleshooting & Runbooks)

> 本模組收錄在 `agent-observer` 開發與長程對話觀測實戰中所遭遇的重大工程故障、底層黑天鵝、QA 問答與標準化事後覆盤（SRE 4-Stage Postmortem）。
> 每一篇卡片均包含 **「現象定義 $\to$ 根因代碼溯源 $\to$ 架構修復方案 $\to$ 總結與 Runbook 診斷 SOP」**，並與對應的架構概念頁面深度互聯。

---

## 🌲 故障排查與 Runbook 知識地圖

```text
05_troubleshooting/
├── 01_Context_Inflation_and_Intermediate_Compounding.md  # 89 萬字膨脹與中間步驟基線污染排查
├── 02_Startup_Warmup_Double_Ingestion_and_Cache_Lag.md   # 開機預熱雙重分析與歷史遙測誤用排查
├── 03_TUI_ANSI_Escape_Truncation_and_Overscroll_Lag.md   # ANSI 字元隱形佔位腰斬折行與滾動延遲排查
├── 04_Filter_Isolation_and_Cache_Expired_Boundary_Leak.md # EXPIRED 步驟洩漏至 MISS 過濾結果排查
├── 05_USER_Input_Inbound_Intent_vs_GPU_Cache_Settlement.md # USER 步驟誤標 MISS 與時序結算錯位排查
└── 06_Single_Line_Card_Static_Packing_Blank_Gap.md        # 單行卡片靜態除二計算導致清單留白排查
```

---

## 📑 核心排查卡片導覽

| 卡片名稱 | 核心故障現象 | 根本原因 (RCA) | 架構修復與長效抗體 | 關聯架構概念 |
| :--- | :--- | :--- | :--- | :--- |
| [[01_Context_Inflation_and_Intermediate_Compounding\|01. 89 萬字膨脹與基線污染]] | Context 暴增至 893,834 Tokens，隨後卡死在 256k | 本地工具步驟連續執行 `state.PrevTotalTokens = totalTokens` 滾雪球覆寫基線 | 確立中間步驟非遞增基線鐵律，施加 256k 物理上限約束 | [[02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting\|雙軌遙測與窗口會計]] |
| [[02_Startup_Warmup_Double_Ingestion_and_Cache_Lag\|02. 開機預熱雙重分析與歷史遙測誤用]] | 早期步驟誤顯會話結尾 25 萬字，事件數雙倍膨脹至 4,800 | `NewWatcher` 未預先 Poll SQLite，`parseLine` 誤用全局最後一筆遙測，`main.go` 雙重分析 | 單一攝入責任鏈，開機預先載入 700+ 世代紀錄，移除全局 fallback | [[02_architecture/03_Agent_Storage_and_State_Machine\|SQLite 7 表與狀態機]] |
| [[03_TUI_ANSI_Escape_Truncation_and_Overscroll_Lag\|03. ANSI 字元隱形佔位與滾動卡頓]] | 終端文字半路腰斬留白 50 格，按 `j` 到底後按 `k` 延遲 75 下 | ANSI 轉義碼誤算為 20 欄實體寬度觸發提前截斷，`docsScroll` 數值溢出空轉 | 實作 ANSI 感知狀態機、29 格懸掛縮排與 `getDocsMaxScroll` 邊界約束 | [[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics\|TUI 引擎與終端盒模型]] |
| [[04_Filter_Isolation_and_Cache_Expired_Boundary_Leak\|04. EXPIRED 洩漏至 MISS 篩選漏洞]] | 按 `c` 切至 `[C:Miss]` 時，黃色 `[EXPIRED]` 步驟混入清單中 | `matchCacheFilter` 僅排除 `WRITE`，未能先制攔截 `EXPIRED`，命中 Fallback 條件 | 實作顯式互斥排除守衛，確保各狀態篩選集嚴格正交隔離 | [[02_architecture/09_History_Explorer_and_Causality_Graph\|歷史步進瀏覽器圖譜]] |
| [[05_USER_Input_Inbound_Intent_vs_GPU_Cache_Settlement\|05. USER 誤標 MISS 與結算錯位]] | 使用者打字步驟顯示紅色 `[MISS]`，面板顯示 `Cached: 0 \| New: 162k` | 人類輸入當下尚未發起推論，適配器在無 Protobuf 時合成全未命中帳單 | 移除 USER 合成標籤，遙測改標 `Staged Intent ➔ Settled in Step #N+1` | [[02_architecture/03_Agent_Storage_and_State_Machine\|SQLite 狀態機]] |
| [[06_Single_Line_Card_Static_Packing_Blank_Gap\|06. 單行卡片靜態除二清單大片留白]] | 篩選 `[T:User]` 時，30 行視窗僅顯示 8 筆卡片，底部留白 14 行 | `getHistoryVisibleCards` 靜態執行 `availLines / 2`，嚴重低估單行卡片容量 | 改採動態累加行數打包演算法，依實際卡片高度 100% 填滿可用行數 | [[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics\|TUI 引擎與盒模型]] |

