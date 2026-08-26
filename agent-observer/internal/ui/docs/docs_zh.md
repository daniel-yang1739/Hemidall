## 上下文 5 大維度解剖 (Track 2: Local 5-Dimension Anatomy)
* **1. System Instruction** : 基礎系統提示詞、開發者規範、專案憲法與安全限制，引導 Agent 核心行為準則。
* **2. MCP Tools Schema** : 工具函式呼叫的 JSON Schema 定義，包含所有可用工具、參數型別、欄位描述與必要條件。
* **3. Tool Results / Diff** : 本地工具執行後回傳的實體內容（終端機 stdout/stderr、檔案內容讀取、目錄清單、代碼差異）。
* **4. Conversation Hist** : 當前活躍滑動窗口內保留的過往使用者與 Assistant 歷史對話及過往工具呼叫摘要。
* **5. Active Turn / CoT** : 當前輪次使用者最新 Prompt + 模型思維鏈推理 (Chain of Thought) + 活躍工具呼叫 Payload。

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

## 上下文力學與顯存物理機制 (Context Mechanics & Physics)
* **Context Compaction** : 雙水位線非同步壓縮，於觸及 95% High Watermark (~245k) 時觸發遞迴摘要，並重置至 48% Low Watermark (~120k)。
* **Reverse Sliding Window** : 倒推滑動窗口演算法，由最新步驟往前倒推填滿 Google 官方活躍預算，精確排除已被淘汰的遠古步驟。
* **Longest Common Prefix** : 最長公共前綴演算法，逐字元比對時序步驟，精確定位 GPU KV Cache 可復用的前綴邊界。
