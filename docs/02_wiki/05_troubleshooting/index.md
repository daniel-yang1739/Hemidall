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
├── 06_Single_Line_Card_Static_Packing_Blank_Gap.md        # 單行卡片靜態除二計算導致清單留白排查
├── 07_Idle_TTL_Masking_by_Local_User_Input_Timestamps.md  # 10 分鐘閒置快取未過期與本地打字時序遮蔽排查
├── 08_Stream_Update_Duplication_and_Step_Counter_Inflation.md # 事件數 10,336 與步驟序號 5,919 重複累加排查
├── 09_Harness_Internal_Plumbing_Filtering_and_SQLite_Gap_Recovery.md # Google 內部管線過濾 31 步驟與全量補齊排查
└── 10_Destructive_History_Filter_vs_Non_Destructive_Jump_Navigation.md # 破壞式過濾 Context 丟失到 Vim 跳轉導航重構
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
| [[07_Idle_TTL_Masking_by_Local_User_Input_Timestamps\|07. 10 分鐘閒置快取未過期之謎]] | 閒置 10 分鐘依然誤顯 `[CACHE HIT 88%]` | 本地打字 `USER_INPUT` 更新了 `LastEventTime`，雲端比對時 $\Delta t=0$，時序遮蔽真實閒置 | 抽離 `LastCloudTurnTime` 專屬時鐘，本地事件嚴禁覆蓋，死守雲端間隔 | [[01_theory/03_Prompt_Caching_Lifecycle\|前綴快取生命週期]] |
| [[08_Stream_Update_Duplication_and_Step_Counter_Inflation\|08. 事件數 10,336 與步驟序號 5,919 脫節]] | 右上角計數器膨脹至 10,639，但清單最新僅 Step #5919 | 串流發送 `RUNNING` 與 `DONE`，接收端無腦 `append` 導致同一步驟重複入庫 | 實裝步驟序號唯一性鎖定與原地覆蓋 (In-Place Update)，Header 改標 `Steps` | [[02_architecture/03_Agent_Storage_and_State_Machine\|SQLite 狀態機]] |
| [[09_Harness_Internal_Plumbing_Filtering_and_SQLite_Gap_Recovery\|09. 消失的 31 個步驟與跳號之謎]] | 列表總數 5889 與最大序號 5919 短少 30 筆（出現跳號） | Google 為了防範 Context 污染與介面降噪，在 jsonl 日誌過濾管線與 BLOCKED 步驟 | 實裝 `MergeMissingSQLiteSteps` 全量補齊，賦予 `INTERNAL` 與 `BLOCKED` 徽章 | [[02_architecture/11_Multi_Agent_Hierarchy_and_Subagent_Token_Economics\|Multi-Agent 階層]] |
| [[10_Destructive_History_Filter_vs_Non_Destructive_Jump_Navigation\|10. 歷史搜尋 Context 丟失與 Vim 跳轉導航]] | 按 `/` 搜尋後列表被裁切，按 Enter 無法上下瀏覽前後步驟 | 搜尋邏輯採用破壞式 Filter 切片，破壞步驟陣列連續性 | 重構為 Vim Jump-to-Step 導航器，完整保留列表，平滑滾動錨定游標 | [[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics\|TUI 引擎與盒模型]] |

