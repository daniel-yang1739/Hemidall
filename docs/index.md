# 🗺️ LLM 記憶中樞總目錄 (Global Vault Map of Content)

> [!NOTE]
> 歡迎來到 **LLM 記憶中樞 (LLM Memory Hub) 永久知識庫**。
> 本目錄作為整個 Obsidian Vault 的最高導覽中樞 (MOC)，提供 High-Level 各目錄職責定義、永久 Wiki 核心卡片的認知索引與全景審查報告書。

---

## 🏛️ 一、High-Level 頂層目錄職責與定位 (Directory Mandates)

| 目錄路徑 | 核心職責與功能定位 | 生命週期與治理規則 |
| :--- | :--- | :--- |
| **`01_raw/`** | **臨時素材收集池 (Intake Pool)**<br/>存放未加工的 API Traces、逆向日誌、臨時截圖與靈感碎片。 | **Digest & Delete**：一旦經 `wiki-distiller` 100% 提煉進 `02_wiki/`，原始檔案立即安全清理刪除，保持素材池極致乾淨。 |
| **`02_wiki/`** | **永久核心知識資產庫 (Permanent Asset Hub)**<br/>世界級、排版精美、結構自洽、具備 4 維度圖解導讀與極簡演繹實例的永久資產。 | **永不刪除 / 持續迭代**：嚴禁存放未完成的草稿。依大腦認知演進（`01_` $\to$ `02_` $\to$ `03_` $\to$ `04_`）嚴密編號。 |
| **`ithome_draft/`** | **文章草稿工作區 (Writing Workspace)**<br/>以 Wiki 為武器庫，專門用於撰寫 iThome 鐵人賽 30 天連載草稿。 | **短期專案週期**：與底層 Wiki 完全解耦，專注於文章受眾節奏、開場 Hook 與章節編排。 |
| **`ithome_ready/`** | **定稿發布庫 (Production Release)**<br/>完成最終潤稿、排版校對，隨時可直接複製 Po 到發文後台。 | **發布就緒**：代表可對外公開發表的正式文章。 |
| **`reviews/`** | **審查辯論與終審報告室 (Audit Room)**<br/>存放 18 位頂尖審查員的深層思維鏈挑惕、Main Agent 駁回/採納辯論與大檢察官簽核。 | **歷史審計存檔**：永久留存審查軌跡，正文 0 人名，所有審查員人名與辯論完整留存於此。 |

---

## 🧠 二、永久知識資產庫導覽 (`02_wiki/`)
👉 **開啟 Wiki 總導覽：[[02_wiki/index|02_wiki 知識庫首頁 (含完整模組結構樹)]]**

### 1. [[02_wiki/01_theory/index|⚡ 01_theory: 推論物理與數學模型 (認知起點)]]
* [[01_Transformer_Prefill_vs_Decode]]：推論兩階段 GEMM 算力密集 vs. GEMV 顯存帶寬密集深度剖析（附 3-Token 極簡演繹）。
* [[02_KV_Cache_Mechanics]]：自回歸 KV Cache 顯存大小數學推導、GQA 演進與 128k OOM 實例計算 ($40GB)。
* [[03_Prompt_Caching_Lifecycle]]：前綴快取生命週期時序轉換、TTL 顯存淘汰與同族變體快取共享（附 2 輪快取突變演繹）。
* [[04_Context_Compaction_and_Summarization]]：雙水位線非同步壓縮管線與「摘要的摘要」$O(1)$ 常數空間收斂數學模型（附實機壓測演繹）。

### 2. [[02_wiki/02_architecture/index|🏛️ 02_architecture: 通用架構與演算法模式 (系統落地)]]
* [[01_Context_5_Dimensions]]：Agent Context 載荷 5 維度模型、對話輪次膨脹趨勢與壓縮戰略（附 3 輪 5 維度數值變遷演繹）。
* [[02_Token_Calculation_and_LCP]]：TikToken (BPE) 分詞與 LCP 最長公共前綴快取演算法 Go 實作（附 Token ID 陣列逐位比對演繹）。
* [[03_Agent_Storage_and_State_Machine]]：工業級 Agent 雙層 SQLite 7 表結構、Protobuf 官方遙測與 100KB 滾動切片雙軌日誌。
* [[04_Service_Plan_Agent_Observer]]：`agent-observer` Go 觀測服務 Clean Architecture 系統架構設計書。
* [[05_Model_Payload_and_API_Traces]]：Context 4 大板塊（System, Tools, Trajectory, Active）組裝順序與底層 API 通訊 JSON Schema。
* [[06_Dual_Track_Telemetry_and_Window_Accounting]]：雙軌遙測引擎（Track 1 官方帳單 vs Track 2 本地解剖）與倒推滑動窗口會計演算法。
* [[07_TUI_Engine_and_Terminal_Layout_Mechanics]]：全螢幕 TUI 引擎架構、Lipgloss 盒模型內外算術、中文字元（CJK）2 倍列寬與軟換行虛擬緩衝區。

### 3. [[02_wiki/03_planning/index|🏆 03_planning: 系列藍圖與規劃規格 (產品全景)]]
* [[01_Master_Plan]]：系列總體企劃書、核心價值主張與四大模組進程圖。
* [[02_30_Days_Breakdown]]：30 天每日詳細大綱、程式碼交付物與 Wiki 武器庫映射。
* [[03_Tech_Stack_Tradeoffs]]：Go vs. Python 跨維度客觀選型矩陣與權衡分析。
* [[04_Phased_Implementation_Roadmap]]：Phase 1 至 Phase 5 循序漸進實作路線圖（Phase 3 雙軌遙測與 TUI 已 100% 驗收）。

### 4. [[02_wiki/04_meta/index|🤖 04_meta: AI 協同工程與知識庫方法論 (元架構與體系)]]
* [[01_Multi_Agent_Adversarial_Review_Pattern]]：19 位世界前 1% 審查矩陣、雙輪深層對抗審查、Main Agent 批判性過濾與自我進化閉環。
* [[02_Obsidian_Vault_Topology]]：全域檔案結構樹、High-Level 目錄職責對照表、Wiki 終點論、三權分立拓撲、.obsidian 嚴格 Git 白名單與 AGENTS.md 協同憲法。

### 5. [[02_wiki/05_troubleshooting/index|🛠️ 05_troubleshooting: 實戰故障排查、QA 問答與 Runbook 手冊 (實戰武器庫)]]
* [[01_Context_Inflation_and_Intermediate_Compounding|01. 89 萬字膨脹與基線污染]]：中間步驟非遞增基線修復與 256k 物理窗口硬約束。
* [[02_Startup_Warmup_Double_Ingestion_and_Cache_Lag|02. 開機預熱雙重分析與歷史遙測誤用]]：單一攝入責任鏈與開機 700+ 世代紀錄預載入。
* [[03_TUI_ANSI_Escape_Truncation_and_Overscroll_Lag|03. ANSI 字元隱形佔位與滾動卡頓]]：ANSI 感知狀態機、29 格懸掛縮排與 `getDocsMaxScroll` 邊界約束。

---

## 📜 三、審查委員會全景報告書 (`docs/reviews/`)
* **👉 [[reviews/2026-08-27_02-15-00_multi_agent_adversarial_review_audit_report|🏛️ 2026-08-27 02:15:00 19 角色雙輪深層對抗審查、QA/Runbook 模組落地與全量 Raw 提煉審查報告書]]**
* **👉 [[reviews/2026-08-26_23-13-21_wiki_distillation_comprehensive_audit_report|🏛️ 2026-08-26 23:13:21 全量 16 篇 Wiki 卡片深度提煉、雙軌遙測、TUI 盒模型與雙水位線壓縮審查報告書]]**
* **👉 [[reviews/2026-08-26_15-11-37_concrete_walkthrough_audit_report|🏛️ 2026-08-26 15:11:37 具體演繹實例升級雙輪審查全景報告書]]**
* **👉 [[reviews/2026-08-26_02-28-53_meta_module_audit_report|🏛️ 2026-08-26 02:28:53 04_meta 模組雙輪審查全景報告書]]**
* **👉 [[reviews/2026-08-26_02-15-00_two_round_audit_report|🏛️ 2026-08-26 02:15:00 全量 12 篇卡片雙輪審查、意見採納與駁回裁決全景報告書]]**
