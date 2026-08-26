---
title: 30 天每日詳細拆解與交付物矩陣 (30 Days Breakdown v2.0 旗艦版)
type: planning
created: 2026-08-18
updated: 2026-08-27
status: completed
tags: [planning, ithome2026, breakdown, schedule, syllabus, universal-agent, grand-finale]
aliases: [30 Days Breakdown v2, 每日詳細拆解 v2, 30天大綱清單, v2旗艦大綱]
---

# 📅 30 天每日詳細拆解與交付物矩陣 (30 Days Breakdown v2.0)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 本文件規劃了 2026 iThome 鐵人賽 Day 01 至 Day 30 每一天的核心論點、實作交付物、視覺化配圖與 `02_wiki/` 知識卡片映射。
> 開篇以 **「至少直到最後一刻，我與 AI 共舞著」** 展開破曉自白，終篇以 **「一個軟體工程師的時代殘響」** 首尾呼應，兼具深邃人文情感與頂級硬核技術縱深。
> *(註：如需查閱專案初期的歷史第一版大綱，請參見：[[03_planning/v1/02_30_Days_Breakdown|30 天每日詳細拆解 (v1.0 初版存檔)]])*

---

## 🎭 開篇與終章的情感共鳴錨點 (Narrative Anchors)

* **🌅 Day 01 開篇 Hook**：
  > *「當全世界都在預告軟體工程師的末日，我選擇不閉上眼睛。我經歷過熱愛、抗拒、接受、擁抱到超越。在那一天真正到來之前，我要親手做一把聽診器，看清它的每一行記憶 —— 至少直到最後一刻，我與 AI 共舞著。」*
* **🌌 Day 30 終章殘響 (Grand Finale)**：
  > *「未來不一定有我，但這個時代的根基我有貢獻，我有見證。我依然可以無比驕傲地說我是軟體工程師，歷經這個灰暗時代、卻未曾低頭的軟體工程師。—— 至少直到最後一刻，我與 AI 共舞著。」*

---

## 📑 30 天詳細大綱與知識武器庫映射表 (v2.0)

### ⚡ 第一週：解構記憶底層 —— 注意力機制與 KV Cache 物理極限 (Day 01 ~ 07)
* **Day 01｜至少直到最後一刻，我與 AI 共舞著 —— 寫在黑盒時代前夕的工程自白**
  * *核心論點*：大模型上下文的「百萬謊言」與顯存殘酷現實；工程師的心路歷程與系列價值主張。
  * *交付物*：全系列架構總覽圖、核心問題清單 $\to$ 引用 [[03_planning/v2/01_Master_Plan]], [[01_theory/02_KV_Cache_Mechanics]]
* **Day 02｜【基礎課】Self-Attention 的幾何直覺：Q, K, V 投影與注意力矩陣演算**
  * *核心論點*：從 Token Embedding、線性投影到 Scaled Dot-Product Attention 的視覺化幾何推導。
  * *交付物*：$3 \times 3$ 最小注意力矩陣數值演算卡片 $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]
* **Day 03｜【基礎課】從 MHA 到 GQA：多頭注意力的顯存妥協與演進哲學**
  * *核心論點*：Multi-Head (MHA)、Multi-Query (MQA) 到 Grouped-Query Attention (GQA) 的架構權衡。
  * *交付物*：Head 分組記憶體對比圖表 $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]
* **Day 04｜Prefill vs. Decode：為什麼長上下文生成會從「算力密集」墜入「顯存帶寬地獄」？**
  * *核心論點*：GEMM vs GEMV 物理瓶頸、Arithmetic Intensity (算術強度) 與帶寬牆。
  * *交付物*：Prefill/Decode 雙階段微架構對比圖、3-Token 演繹 $\to$ 引用 [[01_theory/01_Transformer_Prefill_vs_Decode]]
* **Day 05｜算算你的顯存：KV Cache 記憶體推導公式與 128k OOM 實機演算**
  * *核心論點*：$2 \times 2 \times n_{\text{layers}} \times d_{\text{model}}$ 顯存公式精確推導；128k 上下文 $40GB 顯存 OOM 實例。
  * *交付物*：顯存佔用速查表與 Python/Go 試算腳本 $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]
* **Day 06｜Prompt Caching 的物理真相：前綴快取生命週期、TTL 淘汰與 Partial Hit 稀釋機制**
  * *核心論點*：前綴哈希命中、TTL 顯存置換、Full Hit vs Partial Hit 稀釋公式與 5 大狀態徽章。
  * *交付物*：Prompt Caching 生命週期時序圖 $\to$ 引用 [[01_theory/03_Prompt_Caching_Lifecycle]]
* **Day 07｜【第一週實戰】手刻最小 Transformer 注意力與 KV Cache 視覺化模擬器**
  * *核心論點*：用最小代碼實現單層 Attention 前向傳播與 KV Cache 累積視覺化。
  * *交付物*：Go/Python 最小模擬器開源代碼 $\to$ 引用 [[01_theory/02_KV_Cache_Mechanics]]

---

### 🏛️ 第二週：打造大腦聽診器 —— 手刻 Agent-Observer 觀測服務 (Day 08 ~ 15)
* **Day 08｜黑盒之外：現代 AI Agent 到底塞了多少東西給大腦？Context 5 維度模型解剖**
  * *核心論點*：解構 System, Tools, Trajectory, CoT, User 五維度載荷及其在多輪對話中的爆炸規律。
  * *交付物*：5 維度堆疊比例色塊圖 $\to$ 引用 [[02_architecture/01_Context_5_Dimensions]]
* **Day 09｜雙軌遙測理念：Track 1 官方帳單 Ground Truth vs Track 2 本地切片解剖**
  * *核心論點*：為什麼單靠 API 回傳不夠？官方帳單與本地 Tokenizer 的雙軌互補架構。
  * *交付物*：雙軌遙測架構圖 $\to$ 引用 [[02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting]]
* **Day 10｜六角架構 (Hexagonal Architecture) 實戰：如何設計可擴充的多 Agent 觀測核心？**
  * *核心論點*：Core Domain 與 Adapters 解耦，為後續支援 Antigravity, Claude Code, OpenCode 奠定基礎。
  * *交付物*：Go 六角架構 UML 與 Port/Adapter 介面定義 $\to$ 引用 [[02_architecture/04_Service_Plan_Agent_Observer]]
* **Day 11｜解剖 Google Antigravity CLI：SQLite WAL 7 表狀態機與 100KB 日誌滾動切片**
  * *核心論點*：逆向分析 Antigravity 內部存儲、`gen_metadata` Protobuf 帳單與 WAL 直讀防鎖。
  * *交付物*：7 表關聯 E-R 圖與切片演算法 $\to$ 引用 [[02_architecture/03_Agent_Storage_and_State_Machine]]
* **Day 12｜【實作】用 Go 實作非阻塞 File-Tailer、BPE 分詞與 LCP 最長前綴快取命中算法**
  * *核心論點*：TikToken BPE 編碼、LCP (Longest Common Prefix) 陣列逐位比對、靜默預熱。
  * *交付物*：`internal/adapters/antigravity/watcher.go`, `core/tokenizer.go` $\to$ 引用 [[02_architecture/02_Token_Calculation_and_LCP]]
* **Day 13｜【實作】Bubbletea + Lipgloss 全螢幕雙軌 TUI：ANSI 狀態機與終端盒模型算術**
  * *核心論點*：k9s 風格 TUI、ANSI 感知狀態機、全寬 29 格懸掛縮排、零高度抖動 (Zero-Jitter)。
  * *交付物*：`internal/ui/` 雙軌 TUI 渲染引擎 $\to$ 引用 [[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics]]
* **Day 14｜【實作】全域會話快切中樞 (Ctrl+p) 與歷史步驟防抖動鎖 (Anti-Jitter Lock)**
  * *核心論點*：動態會話發現、即時熱切換、倒序/正序索引映射抗抖動數學證明。
  * *交付物*：`internal/ui/model.go` 會話快切面板 $\to$ 引用 [[02_architecture/08_Interactive_Session_Switching_and_Anti_Jitter]]
* **Day 15｜【實作】Go embed 雙語架構辭典與 Vim-First 鍵盤流暢導航體驗**
  * *核心論點*：單一二進制打包 (`//go:embed`)、`[t]` 語系切換、`Tab/Shift+Tab` 循環切頁、`h/l` 分欄。
  * *交付物*：`docs_view.go`, `shortcuts.go` $\to$ 引用 [[02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics]]

---

### 🤖 第三週：擴充萬能生態系 —— 征服 Claude Code 與 OpenCode (Day 16 ~ 21)
* **Day 16｜解剖 Anthropic Claude Code：5 分鐘 TTL 快取斷點機制 (Cache Breakpoints) 與日誌逆向**
  * *核心論點*：Claude 3.5 Sonnet Prompt Caching 機制（1024 斷點、5 分鐘 TTL、4 個快取標記限制）。
  * *交付物*：Claude Code 狀態日誌結構圖 $\to$ 引用 [[01_theory/03_Prompt_Caching_Lifecycle]]
* **Day 17｜【實作】開發 Claude Code Adapter：即時捕捉 Claude 3.5 Sonnet 快取端點與 Token 帳單**
  * *核心論點*：為 `agent-observer` 實作 Claude Adapter，捕獲 `cache_read_input_tokens`。
  * *交付物*：`internal/adapters/claudecode/` 適配器代碼
* **Day 18｜解剖 OpenCode / 開源 Agent 生態系：狀態機協議、Tool Schema 注入與 Local LLM (Ollama)**
  * *核心論點*：開源 Coding Agent 的上下文組裝協議、OpenAI API 兼容流與 Ollama/vLLM 本地模型。
  * *交付物*：OpenCode 事件流解析規格
* **Day 19｜【實作】開發 OpenCode / OpenAI 通用適配器：讓本地開源模型也擁有雙軌遙測**
  * *核心論點*：支援本地開源 Agent 遙測串流，實現跨雲端與本地的一致觀測。
  * *交付物*：`internal/adapters/opencode/` 通用適配器代碼
* **Day 20｜三大 Agent 實彈對決：Antigravity vs Claude Code vs OpenCode 快取命中率與膨脹率大評測**
  * *核心論點*：同一任務下三大 Agent 的 Token 消耗曲線、快取穩定性與成本對比。
  * *交付物*：三大 Agent 橫向評測對比矩陣與圖表
* **Day 21｜【第三週結案】萬能 AI Agent 觀測平台成型：一鍵切換漫遊三大 Agent 生態**
  * *核心論點*：統一多 Agent 介面，以單一 TUI 監控全工作區所有 AI 程式設計活動。
  * *交付物*：Universal Agent-Observer 跨生態實機展示

---

### 🧩 第四週：極致壓縮引擎、長程壓測與工程師的終章 (Day 22 ~ 30)
* **Day 22｜壓縮的哲學：什麼該忘記？什麼該死記？雙水位線非同步管道與常數收斂模型**
  * *核心論點*：高低水位線觸發、LRU 淘汰與「摘要的摘要」$O(1)$ 常數空間收斂數學模型。
  * *交付物*：壓縮管線狀態機時序圖 $\to$ 引用 [[01_theory/04_Context_Compaction_and_Summarization]]
* **Day 23｜【實作】Tool Output 語義剪枝 (Semantic Pruning)：過期工具輸出的外科手術式剔除**
  * *核心論點*：歷史 Bash/Cat 巨大輸出的無損替換演算法，保留最後狀態並截斷中間冗餘。
  * *交付物*：`internal/compressor/pruner.go` 語義剪枝器
* **Day 24｜【實作】差分狀態機 (Delta State Compression)：用 Diff 取代全量 Prompt 堆疊**
  * *核心論點*：檔案狀態的 Delta 運算，以 Git-like Diff 替代全量程式碼重複發送。
  * *交付物*：`internal/compressor/diff.go` 差分狀態壓縮器
* **Day 25｜【實作】極致壓縮引擎 Pipeline：實測 80% 壓縮率且 0 語義遺失的終極 Benchmark**
  * *核心論點*：整合剪枝、摘要與差分，在真實 SWE-bench 任務中壓測壓縮率與準確率。
  * *交付物*：`internal/compressor/engine.go` 與壓測評測報告
* **Day 26｜長程任務大壓測：連續對話 100 輪以上，Context 如何抵抗指數級膨脹？**
  * *核心論點*：長程任務中的 Memory Leaks 排查、迷失在中間 (Lost in the Middle) 緩解實測。
  * *交付物*：100 輪長程對話 Token 水位對比曲線
* **Day 27｜SRE 實戰覆盤：89 萬字膨脹、開機雙重攝取與 ANSI 截斷之 4 階段 Postmortem 與 Runbook**
  * *核心論點*：真實遭遇的三大黑天鵝故障，從現象、根因、代碼修復到 1 分鐘 SOP Runbook。
  * *交付物*：SRE 實戰故障手冊 $\to$ 引用 [[05_troubleshooting/index]]
* **Day 28｜開源發布：`agent-observer` 架構全景圖、安裝部署與自訂 Adapter 開發指南**
  * *核心論點*：專案開源發布（GitHub/Brew/Go Install）、模組擴充教學與社群貢獻指南。
  * *交付物*：開源 Repository、README、CI/CD 與跨平台 Binary 發布
* **Day 29｜下一代 Agent 記憶架構展望：從靜態 Context Window 到動態外掛記憶神經網絡**
  * *核心論點*：PagedAttention, vLLM 顯存虛擬化、長程圖向量記憶與 Hybrid Attention 趨勢。
  * *交付物*：未來 AI 記憶體架構演進藍圖
* **Day 30｜一個軟體工程師的時代殘響：當黑盒化為透明，至少直到最後一刻，我與 AI 共舞著**
  * *核心論點*：30 天全系列總結；一位軟體工程師身處灰暗時代的浪漫與驕傲。
  * *交付物*：全系列知識地圖 (Master MOC)、完結後記與給未來工程師的信 $\to$ 引用 [[03_planning/v2/01_Master_Plan]]

---

## 🔗 三、模組關聯與延伸索引
* [[03_planning/v2/01_Master_Plan|總體企劃書 (v2.0 旗艦版)]]
* [[03_planning/v2/03_Tech_Stack_Tradeoffs|Go vs. Python 技術選型權衡]]
* [[03_planning/v2/04_Phased_Implementation_Roadmap|Phase 1 ~ 5 實作路線圖 (v2.0)]]
* [[03_planning/v1/02_30_Days_Breakdown|30 天每日詳細拆解 (v1.0 初版存檔)]]
