## 上下文 5 大維度解剖 (Track 2: Local 5-Dimension Anatomy)
* **1. System Instruction (系統提示詞)** : 基礎系統規範、專案憲法（AGENTS.md）與操作限制。
  - **物理邊界** : 位於 GPU 顯存 KV Cache 的最前端頂部（Prefix Offset 0），在對話生命週期中永久常駐。
  - **容量佔比** : 約佔總視窗的 2% ~ 8%（約 5k ~ 20k Tokens），視載入的自訂 Skill 與 Agent Role 數量而定。
  - **快取效益** : 享有 100% 顯存復用率，是全會話快取命中率與延遲降低的基本盤。

* **2. MCP Tools Schema (工具定義規格)** : 工具函式呼叫的結構化 JSON Schema 規格庫。
  - **規格內容** : 包含所有可用 Tool（如 view_file, run_command, replace_file_content 等）的參數型別、欄位描述與呼叫限制。
  - **物理機制** : 模型依靠這些 Schema 生成結構化的 Tool Call 請求；每次發起推論時都會作為系統提示詞的延伸完整傳入。
  - **容量佔比** : 約佔總視窗的 3% ~ 10%（約 8k ~ 25k Tokens）。

* **3. Tool Results / Diff (工具執行結果與差異)** : 本地工具執行後向模型回報的實體數據反饋。
  - **內容來源** : 終端機 stdout/stderr 輸出、檔案讀取內容、目錄結構清單、代碼 Diff 變更塊。
  - **增長特徵** : 隨使用者任務進行急劇膨脹，為整機上下文消耗最大宗來源（通常佔據 60% ~ 85% 總容量）。
  - **快取特性** : 本地產生的字數在產生當步為 0 GPU Tokens，於下一輪雲端推論時一次性包裝送出並寫入 KV 快取。

* **4. Conversation Hist (過往對話歷史)** : 當前活躍滑動窗口內保留的過往輪次歷史對話。
  - **內容組成** : 過去幾輪的 User Prompt 提問、Model 的回覆結論及歷史工具調用摘要。
  - **滑動機制** : 透過 Reverse Sliding Window 倒推算法維持在 256k 上限內；超出預算的遠古輪次會被逐步淘汰或壓縮。
  - **容量佔比** : 通常維持在 10% ~ 25% 總容量。

* **5. Active Turn / CoT (當前活躍輪次與思維鏈)** : 當前正在進行中的推論輪次暫存負載。
  - **內容組成** : 使用者最新輸入的 Prompt + 雲端模型思維鏈推理 (Chain of Thought, CoT) + 當前發起的工具調用。
  - **生命週期** : 在當前 Turn 結束並獲得所有工具結果後，自動沉澱並轉化為 Conversation Hist。

## Track 1 面板標題狀態與觸發情境 (Track 1 Panel Title Variations & Scenarios)
* **☁️ TRACK 1: OFFICIAL CLOUD TELEMETRY** : 主控規劃代理發起雲端推論，且 SQLite 成功寫入官方真實帳單。
  - **觸發情境** : 主控代理 (Main Planner) 執行推論思考，且世代元資料庫已解析到 Google Protobuf 官方 Token 計費。
  - **核心指標** : Total Context Window (總視窗長度)、Step Delta (新輸入冷字)、Prefix Cache Savings (快取命中率與 75% 費用減免)、Turn Cost (單輪美金/台幣帳單)。
  - **物理意義** : 唯一真正產生雲端 API 費用與 GPU 顯存吞吐的步驟，代表模型消化先前所有暫存資料並完成決策。

* **👥 TRACK 1: SUBAGENT CLOUD TELEMETRY** : 並行子代理 (Subagent Worker) 發起獨立雲端推論。
  - **觸發情境** : 主控 Agent 透過 `invoke_subagent` 衍生之獨立並行 Worker 發起雲端推論。
  - **核心指標** : 子代理獨立的 Context Window、獨立的前綴快取命中率與 API 計費，與主會話視窗完全隔離。
  - **架構意義** : 子代理擁有獨立的 System Prompt 與工作區，其 Token 消耗與主 Agent 分流計費。

* **👤 TRACK 1: USER INTERACTION (CLIENT PROMPT)** : 人類工程師在終端機輸入 Prompt 自然語言指令。
  - **觸發情境** : 使用者在終端機下達新任務目標，或在多選 Clarification 互動視窗中點選回應。
  - **核心指標** : Total Context 標記為 Nil (尚未發起推論)、Step Delta (本次使用者打字字數)、狀態為 `Staged locally ⏳`。
  - **物理意義** : 使用者意圖剛進入本機暫存佇列，尚未被傳送至雲端 GPU，不具備 Context 視窗載荷。

* **💻 TRACK 1: LOCAL EXECUTION STEP (OFFLINE)** : Mac 本機執行工具產生的離線步驟。
  - **觸發情境** : 本機執行工具（如 `run_command` 終端指令、`view_file` 讀檔、`replace_file_content` 編輯）。
  - **核心指標** : Total Context 為 Nil (本機子程序運作，0 GPU Tokens)、Step Delta (本地輸出的 BPE 字數暫存緩衝區)、狀態標記為 `Staged for Next Cloud Turn`。
  - **物理意義** : 這是離線的本機計算，不消耗雲端配額，產生的大量資料會在下一輪雲端推論時打包結算。

* **🛡️ TRACK 1: PERMISSION BOUNDARY (BLOCKED)** : 安全守衛攔截敏感檔案或高危險操作。
  - **觸發情境** : Agent 嘗試存取受保護系統檔、使用者拒絕高風險 Shell 指令，或在確認框按下 Cancel。
  - **核心指標** : Total Context 為 Nil、0 GPU Tokens 消耗、`Blocked by System Permission Guard 🛡️`。
  - **防禦機制** : 透過本地攔截機制確保敏感指令完全不會送往雲端，保障資安且零 Token 浪費。

* **🦿 TRACK 1: INTERNAL HARNESS BACKGROUND TASK** : 宿主系統 (Antigravity) 自動產生的背景協調事件。
  - **觸發情境** : 背景非同步任務執行完畢通知 `<SYSTEM_MESSAGE>`、定時排程 `schedule` 喚醒信號、Linter / IDE 自動診斷回饋。
  - **核心指標** : Total Context 為 Nil、本機系統自動發起之事件，用於反應式喚醒 (Reactive Wakeup) 模型。
  - **協調機制** : 確保背景非同步運作的輸出能以可持久化文字形式記錄，並在下個步驟精確餵給 LLM 大腦。

* **⚙️ TRACK 1: SYSTEM COMPACTION (CHECKPOINT)** : 上下文達到物理上限觸發之 Sidecar GC 壓縮檢查點。
  - **觸發情境** : 對話歷史達到 256k 物理上限時，Antigravity 自動啟動背景壓縮，將 200k+ 歷史濃縮為 10k 摘要。
  - **核心指標** : Summary Size (壓縮後摘要大小)、Step Delta (摘要長度)、`Re-anchors Active Window Base 🔄`。
  - **記憶再生** : 將龐大的歷史背景摘要化，重置會話前綴基底，讓對話得以無上限持續進行。

## 多代理協同與角色分工 (Multi-Agent Architecture & Roles)
* **👑 MAIN PLANNER (主控規劃代理)** : 負責直接與使用者對話、制定頂層執行計劃並調度工具。
  - **獨立視窗** : 維持主會話獨立的 Context 滑動窗口，享有主會話前綴快取的持續命中。
  - **決策邊界** : 負責頂層任務拆解、權限請求與向使用者回報最終成果。

* **👥 SUBAGENT WORKER (並行子代理)** : 由主控 Agent 透過 `invoke_subagent` 動態派發之背景工作程序。
  - **隔離空間** : 擁有獨立的 Context 視窗、獨立的 System Prompt 與獨立的 API 配額計費。
  - **成本效益** : 本地執行的 Tool 步驟為 0 GPU Tokens，執行成果匯總後回傳主 Agent。

* **⚙️ INTERNAL HARNESS (本機宿主載具)** : 包裹 LLM、提供終端環境與背景進程管理的 Antigravity 宿主系統。
  - **協調職責** : 管理檔案讀寫權限、排程計時器、非同步 Task 監聽與事件日誌記錄。

* **🛡️ BLOCKED / DENIED (安全守衛)** : 權限攔截防線。
  - **防護範圍** : 阻止未授權的敏感檔案存取、危險 Shell 執行，確保 0 GPU Token 消耗。

## 步驟類型與生命週期 (Step Types & Agent Lifecycle)
* **👤 USER_INPUT** : 人類工程師在終端機輸入之自然語言指令。
  - **生命週期** : 標誌新 Turn 的起點。本機暫存狀態（Total Context 為 Nil），於下一輪雲端決策回傳時核算真實帳單。
* **🤖 MODEL_RESPONSE** : 雲端 LLM 自然語言思考、思維鏈推理 (CoT) 與結構化回覆。
  - **生命週期** : 產生真實 GPU 運算與帳單，享有前綴快取加速與 75% 費用折扣。
* **🛠️ TOOL_CALL** : 模型向本地發出的工具呼叫請求（如 run_command, view_file）。
  - **生命週期** : 包含工具名稱與 JSON 參數，可單發或平行多發調用。
* **💻 TOOL_RESULT / OUTPUT** : 本地工具實體執行後回傳的文字數據。
  - **生命週期** : 為本機離線運作（0 GPU Token），打包作為下一次推論之輸入。
* **🦿 SYSTEM_MESSAGE** : 本機 Harness 產生的背景事件通知（如非同步子程序完成、定時喚醒信號）。
  - **生命週期** : 用於反應式喚醒 (Reactive Wakeup) 模型進入下一輪推論。
* **📜 SYSTEM_INIT** : 系統開局與環境初始化事件。
  - **生命週期** : 注入 Agent Identity、專案憲法規範與工具定義。
* **⚙️ CHECKPOINT** : 會話截斷與壓縮檢查點（Truncation Checkpoint）。
  - **生命週期** : 當上下文達到 256k 上限時由 Antigravity 自動觸發摘要壓縮，重置會話前綴。
* **⚠️ ERROR_MESSAGE** : 系統異常或執行錯誤事件（如網路逾時、程序崩潰）。

## 快取狀態標籤與計費語意 (Cache Status Badges & Semantics)
* **[CACHE HIT]** : 前綴快取命中率 >= 80.0%。
  - **物理狀態** : 絕大部分上下文直接命中 GPU 顯存 (HBM)，享有 75% 費用折扣與極低延遲。
* **[PARTIAL HIT]** : 前綴快取命中率介於 0.1% ~ 79.9%。
  - **物理狀態** : 既有前綴成功命中，但本輪湧入了超大檔案或大量 Tool 輸出，使總體命中率被稀釋。
* **[CACHE WRITE]** : 會話第 0 步首次開局（命中率 0.0%）。
  - **物理狀態** : 系統提示詞與工具定義首次寫入 GPU KV Cache 顯存。
* **[TTL EXPIRED]** : 閒置超時淘汰（超過 5 分鐘未操作）。
  - **物理狀態** : GPU 顯存釋放先前快取，本輪全量歷史被迫重新計算計費（冷啟動）。
* **[CACHE MISS]** : 快取未命中（0.0%）。
  - **物理狀態** : 因前綴文字被修改、模型動態切換或跨模型路由導致快取鏈破壞。

## 全局統計指標與計費演算法 (Aggregate Metrics & Pricing Algorithms)
* **Total Processed (處理總量)** : 整個會話所有雲端輪次累計處理的總 Token 數（Σ TotalTokens）。
  - **意義** : 衡量模型運算整體吞吐量。
* **Cache Hit Volume (快取總量)** : 整個會話累計成功命中 GPU KV 快取的字數（Σ CachedTokens）。
  - **意義** : 衡量會話整體節省的主要來源。
* **Uncached Inbound (冷字總量)** : 整個會話累計未快取新傳入字數（Σ NewTokens）。
  - **意義** : 需耗費 GPU 算力進行完整 Prefill 計算之冷字總量。
* **Effective Tokens (等效字數)** : 經多模型快取折扣加權後的等效字數。
  - **計算公式** : Effective = Σ [Cached × (1 - Discount) + New]，反映真實付費權重。
* **Cached Saved % (節省比例)** : 透過 GPU 前綴快取所節省的 Token 比例。
  - **計算公式** : Saved = Cached × Discount，代表快取帶來的成本減免效益。
* **Multi-Model Discount Matrix** : 各大模型官方快取折扣矩陣。
  - **折扣標準** : Gemini 3.7 Flash: 75% 折扣；Gemini 2.5 Pro: 75% 折扣；Claude 3.7 Sonnet: 90% 折扣。
* **Google AI Pro Quota (5000 RPD)** : Google AI Pro 每日請求配額消耗追蹤。
  - **計算公式** : 以 Turns / 5000 × 100% 計算當前配額消耗百分比。

## 上下文力學與顯存物理機制 (Context Mechanics & Physics)
* **Context Compaction (雙水位線壓縮)** : 記憶體垃圾回收機制。
  - **運作機制** : 於觸及 95% High Watermark (~245k) 時觸發遞迴摘要，並重置至 48% Low Watermark (~120k)。
* **Reverse Sliding Window (倒推滑動窗口)** : 活躍窗口精準定位。
  - **運作機制** : 由最新步驟往前倒推填滿 Google 官方活躍預算，精確排除已被淘汰的遠古步驟。
* **Longest Common Prefix (最長公共前綴)** : GPU KV Cache 邊界鎖定。
  - **運作機制** : 逐字元比對時序步驟，精確定位 GPU 顯存可復用的前綴邊界。
