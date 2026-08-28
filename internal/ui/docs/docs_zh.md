## 上下文 5 大維度解剖 (Track 2: Local 5-Dimension Anatomy)
* **1. System Instruction** : 基礎系統提示詞、開發者規範、專案憲法與安全限制，引導 Agent 核心行為準則。
* **2. MCP Tools Schema** : 工具函式呼叫的 JSON Schema 定義，包含所有可用工具、參數型別、欄位描述與必要條件。
* **3. Tool Results / Diff** : 本地工具執行後回傳的實體內容（終端機 stdout/stderr、檔案內容讀取、目錄清單、代碼差異）。
* **4. Conversation Hist** : 當前活躍滑動窗口內保留的過往使用者與 Assistant 歷史對話及過往工具呼叫摘要。
* **5. Active Turn / CoT** : 當前輪次使用者最新 Prompt + 模型思維鏈推理 (Chain of Thought) + 活躍工具呼叫 Payload。

## 多代理協同與角色分工 (Multi-Agent Architecture & Roles)
* **👑 MAIN PLANNER** : 主控規劃代理，負責直接與使用者對話、制定頂層執行計劃，維持主會話獨立的 Context 視窗（享有主會話前綴快取）。
* **👥 SUBAGENT WORKER** : 並行子代理進程，由主控 Agent 透過 invoke_subagent 派發獨立任務。擁有獨立的 Context 視窗、獨立的 System Prompt 與獨立的 API 配額計費，本地執行的 Tool 步驟為 0 GPU Token。
* **⚙️ INTERNAL HARNESS** : 系統內部背景排程或初始化子任務（如環境探測、配置檔初始化），由 Antigravity 框架自動執行。
* **🛡️ BLOCKED / DENIED** : 安全權限攔截。當 Agent 嘗試讀取受保護檔案（如 settings.json）、使用者拒絕高危險指令或任務被中止時觸發，0 GPU Token 消耗。

## 核心度量與官方真實帳單 (Track 1: Billing Ground Truth)
* **Total Active Context** : 當前 HTTP 請求發送至雲端 LLM 的精確總 Token 數（例如 185k / 256k 物理窗口上限）。
* **Prefix Cache Hit** : 直接復用雲端 GPU 顯存 (HBM) 的前綴 Token 數（享有 70%~75% 費用折扣，極速響應）。
* **New Billable Tokens** : 本輪新輸入的未快取 Token（新 Prompt + 新工具結果），需消耗 GPU 算力進行完整 Prefill 計算。
* **Raw Log Accumulated** : 本地 Append-Only 磁碟日誌累積的未壓縮 Token 總量（如 170 萬字），大於雲端滑動窗口。
* **TTL Cold Start** : Google GPU 顯存閒置超時（約 5 分鐘）後淘汰 KV 快取；後續請求將引發全量冷啟動重新計費。

## 快取狀態標籤與計費語意 (Cache Status Badges & Semantics)
* **[CACHE HIT]** : 前綴快取命中率 >= 80.0%。高性價比黃金狀態，絕大部分上下文直接命中 GPU 顯存（享 75% 費用折扣、極低 TTFT 延遲）。
* **[PARTIAL HIT]** : 前綴快取命中率介於 0.1% ~ 79.9%。既有前綴成功命中，但本輪湧入了超大檔案或大量 Tool 輸出，使總體命中率被稀釋。
* **[CACHE WRITE]** : 會話第 0 步首次開局（命中率 0.0%）。系統提示詞與工具定義首次寫入 GPU KV Cache 顯存。
* **[TTL EXPIRED]** : 閒置超時淘汰（超過 5 分鐘未操作）。GPU 顯存釋放先前快取，本輪全量歷史被迫重新計算計費（冷啟動）。
* **[CACHE MISS]** : 快取未命中（0.0%）。因前綴文字被修改、模型動態切換或跨模型路由導致快取鏈破壞。

## 全局統計指標與計費演算法 (Aggregate Metrics & Pricing Algorithms)
* **Total Processed (處理總量)** : 整個會話所有雲端輪次累計處理的總 Token 數（Σ TotalTokens），衡量模型運算整體吞吐量。
* **Cache Hit Volume (快取總量)** : 整個會話累計成功命中 GPU KV 快取的字數（Σ CachedTokens），為整體節省的主要來源。
* **Uncached Inbound (冷字總量)** : 整個會話累計未快取新傳入字數（Σ NewTokens），需耗費 GPU 算力進行完整 Prefill 計算。
* **Effective Tokens (等效字數)** : 經多模型快取折扣加權後的等效字數：Effective = Σ [Cached × (1 - Discount) + New]，反映真實付費權重。
* **Cached Saved % (節省比例)** : 透過 GPU 前綴快取所節省的 Token 比例：Saved = Cached × Discount，代表快取帶來的成本減免效益。
* **Multi-Model Discount Matrix** : Gemini 3.7 Flash 享 75% 快取折扣；Gemini 2.5 Pro 享 75% 快取折扣；Claude 3.7 Sonnet 享 90% 快取折扣。
* **Google AI Pro Quota (5000 RPD)** : Google AI Pro 方案每日享有 5,000 次雲端推理請求上限，系統以 Turns / 5000 × 100% 計算當前配額消耗百分比。

## 上下文力學與顯存物理機制 (Context Mechanics & Physics)
* **Context Compaction** : 雙水位線非同步壓縮，於觸及 95% High Watermark (~245k) 時觸發遞迴摘要，並重置至 48% Low Watermark (~120k)。
* **Reverse Sliding Window** : 倒推滑動窗口演算法，由最新步驟往前倒推填滿 Google 官方活躍預算，精確排除已被淘汰的遠古步驟。
* **Longest Common Prefix** : 最長公共前綴演算法，逐字元比對時序步驟，精確定位 GPU KV Cache 可復用的前綴邊界。

## 步驟類型與生命週期 (Step Types & Agent Lifecycle)
* **👤 USER_INPUT** : 使用者自然語言指令，標誌新會話輪次（Turn）起點。因尚未進入模型推論，快取與計費真理在後續雲端決策回傳時核算。
* **🤖 MODEL_RESPONSE** : 雲端 LLM 自然語言思考與回覆，包含 CoT 思考鏈及結構化回應。享有前綴快取加速與 75% 費用折扣。
* **🛠️ TOOL_CALL** : 雲端模型向本地發出的工具調用指令（如 run_command, view_file, edit_file），可單發或平行多發。
* **💻 OUTPUT** : 本地工具實體執行反饋（stdout/stderr、檔案內容、代碼 Diff）。為本機離線運作（0 Token），打包至下一輪雲端計費。
* **⚙️ CHECKPOINT** : 會話截斷與壓縮檢查點（Truncation Checkpoint）。當上下文達到上限時由 Antigravity 自動觸發摘要壓縮，重置會話前綴。
* **⚠️ ERROR_MESSAGE** : 系統異常或執行錯誤事件（如網路中斷、指令逾時、程式崩潰）。
