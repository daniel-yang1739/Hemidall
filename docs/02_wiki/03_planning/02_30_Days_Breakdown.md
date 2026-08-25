---
title: 30 天每日詳細拆解與交付物矩陣 (30 Days Breakdown)
type: planning
created: 2026-08-18
updated: 2026-08-26
status: completed
tags: [planning, ithome2026, breakdown, schedule, syllabus]
aliases: [30 Days Breakdown, 每日詳細拆解, 30天大綱清單]
---

# 📅 30 天每日詳細拆解與交付物矩陣 (30 Days Breakdown)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 本文件規劃了 Day 01 至 Day 30 每一天的核心論點、實作交付物、視覺化配圖與 `02_wiki/` 知識卡片映射。每篇文章均以獨立且自洽的技術小品形式呈現，同時在宏觀上形成完整的系統工程閉環。

---

## 📑 30 天詳細大綱與知識武器庫映射表

### ⚡ 第一週：解構 LLM 記憶底層 —— Transformer KV Cache 的物理極限
* **Day 01**：為什麼百萬上下文是個「美麗的謊言」？—— 算力與顯存的殘酷現實 $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]
* **Day 02**：回到注意力機制：Query, Key, Value 的矩陣幾何學 $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]
* **Day 03**：自回歸生成的代價：為什麼 Decode 階段需要 KV Cache？ $\to$ 引用 [[01_theory/01_Transformer_Prefill_vs_Decode]]
* **Day 04**：算算你的顯存：KV Cache 記憶體佔用公式與推導 $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]
* **Day 05**：Attention 架構演進：MHA vs MQA vs GQA $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]
* **Day 06**：Prompt Caching 的誕生：前綴快取原理與生命週期 $\to$ 引用 [[01_theory/03_Prompt_Caching_Lifecycle]]
* **Day 07**：【第一週實戰】手刻最小 Transformer 注意力與 KV Cache 記憶體視覺化工具

---

### 🏛️ 第二週：打造大腦聽診器 —— 手刻 Agent-Observer 觀測服務
* **Day 08**：黑盒之外：現代 AI Agent 到底塞了多少東西給大腦？ $\to$ 引用 [[02_architecture/01_Context_5_Dimensions]]
* **Day 09**：解剖 Antigravity CLI：系統架構與狀態機儲存模式 $\to$ 引用 [[02_architecture/03_Agent_Storage_and_State_Machine]]
* **Day 10**：雙軌日誌與 100KB 切片機制 $\to$ 引用 [[02_architecture/03_Agent_Storage_and_State_Machine]]
* **Day 11**：Context 載荷 4 大板塊與 JSON Schema $\to$ 引用 [[02_architecture/03_Agent_Storage_and_State_Machine]]
* **Day 12**：【實作】用 Go 實作非阻塞 File-Tailer 與即時解析 $\to$ 引用 [[02_architecture/04_Service_Plan_Agent_Observer]]
* **Day 13**：【實作】5 維度 Token 水位解剖器與 BPE 編碼 $\to$ 引用 [[02_architecture/02_Token_Calculation_and_LCP]]
* **Day 14**：【實作】SSE 即時廣播與單一 Binary 嵌入 Web 儀表板 $\to$ 引用 [[02_architecture/04_Service_Plan_Agent_Observer]]
* **Day 15**：【第二週結案】透視真實 Agent 執行的 Token 堆疊與 Cache 命中率

---

### 🧩 第三週：手刻極致壓縮引擎 —— 幾何級 Context 壓縮技術
* **Day 16**：壓縮的哲學：什麼該忘記？什麼該死記？ $\to$ 引用 [[02_architecture/01_Context_5_Dimensions]]
* **Day 17**：語義剪枝 (Semantic Pruning)：淘汰過期工具輸出 $\to$ 引用 [[02_architecture/01_Context_5_Dimensions]]
* **Day 18**：分層摘要 (Hierarchical Summarization)：滾動記憶金字塔
* **Day 19**：差分狀態機 (Delta State Compression)：Diff 壓縮 $\to$ 引用 [[02_architecture/03_Agent_Storage_and_State_Machine]]
* **Day 20**：MCP 工具 Schema 動態過濾 (Just-In-Time Tool Injection)
* **Day 21**：【實作】極致壓縮引擎 Pipeline 搭建
* **Day 22**：【實作】壓縮率與語義保真度評測基準 (Benchmark)
* **Day 23**：【第三週結案】挑戰 80% 壓縮率且 0 語義遺失

---

### 🚀 第四週：長程 Agent 綜合實戰與前瞻突破
* **Day 24**：多 Session 狀態隔離與持久化儲存 $\to$ 引用 [[02_architecture/03_Agent_Storage_and_State_Machine]]
* **Day 25**：Multi-Agent 協作下的記憶爆炸與跨 Agent 快取共享 $\to$ 引用 [[01_theory/03_Prompt_Caching_Lifecycle]]
* **Day 26**：打破 10 萬 Token 天花板：長程任務穩定性實測
* **Day 27**：開源發布：`agent-observer` 架構與使用指南 $\to$ 引用 [[01_Master_Plan]]
* **Day 28**：前瞻架構：PagedAttention, vLLM 與硬體級記憶體虛擬化
* **Day 29**：從 RNN 到 Mamba / Transformer 記憶演進沉思 $\to$ 引用 [[01_theory/01_Transformer_Prefill_vs_Decode]]
* **Day 30**：【完結篇】寫給每位 AI 工程師的 LLM 記憶管理指南 $\to$ 引用 [[01_Master_Plan]]
