## 系統架構與資料庫全景 (System Architecture & Database)

* **Antigravity 儲存架構** : 對話工作階段（Session）持久化於 `~/.gemini/antigravity-cli/conversations/<UUID>.db`，由 7 張核心資料表與 `exa.cortex_pb` 官方 Protobuf 結構組成。
  * **`trajectory_meta`** : 工作階段的身分識別根節點（包含 Trajectory ID、Cascade ID、來源與軌跡類型）。
  * **`trajectory_metadata_blob`** : 本機工作區目錄路徑、Git Remote、分支與環境指紋（`CortexTrajectoryMetadata`）。
  * **`executor_metadata`** : 檔案變更安全氣囊，記錄 Tool 執行前後的本地檔案 Diff 快照，支援使用者中斷（`Ctrl+C`）時原子性回滾。
  * **`gen_metadata`** : 雲端推論收據（Token 帳單、TTFT 延遲、Vertex Trace ID，以及最新活動上下文快照）。
  * **`steps`** : 使用者時間線核心表，包含 6 大 BLOB（`metadata`、`step_payload`、`render_info`、`permissions`、`task_details`、`error_details`）。
  * **`parent_references`** : 子 Agent（Subagent）派生拓撲關聯（記錄父對話 ID、Step 索引與 Workspace 隔離模式）。
  * **`battle_mode_infos`** : 多模型 A/B 對決評測分叉與勝出對話紀錄。

## System Prompt 歷史解析策略 (Solution 1: Snapshot Baseline)

* **滾動瘦身與歷史快照機制** : Antigravity 為防止 SQLite 資料庫膨脹（避免單一對話超過 400MB），採用了滾動壓縮機制。
  * **歷史筆數（`idx < MAX`）** : 壓縮為約 1.1 KB 的精簡收據，僅保留 Token 計費、TTFT 延遲、總耗時與 Step 邊界；完整的 `system_prompt` 與 `tools` 被物理覆寫。
  * **最新一筆（`idx = MAX`）** : 保存為 400KB~850KB 的完整活動快照（包含 2.5 萬字 System Prompt、17 個 Tools Schema 與完整歷史對話）。
* **解法 1：全域基準反向投影 (Snapshot Baseline Reconstruction)** :
  * **語義常數性** : 在同一個工作區對話中，System Prompt（專案規則 `AGENTS.md`、內建 Instruction、Tool Schema）在 99.9% 情況下為不可變常數。
  * **Heimdall 處理原則** : Heimdall 直接由最新一筆 `MAX(idx)` 快照提取權威 System Prompt 作為該 Session 的全域基準（Global Baseline），反向賦予給歷史各輪次。
  * **誠實可觀測性標示** : 在 UI 上明確標示「歷史完整 Context 已由客戶端壓縮；當前呈現繼承自最新快照之系統提示詞基準」，確保觀測資料的技術嚴謹性。

## 雙層 Token 記錄機制 (Dual-Layer Token Architecture)

* **雙層資料流設計** : 資料庫中同時存在兩條互相呼應的 Token 資料線，分別滿足「審計」與「UI 極速渲染」。
  * **`gen_metadata`（雲端推論收據層）** : 每次雲端 API 生成產生 1 筆權威收據，記錄未快取 Token（`F4.2`）、快取命中 Token（`F4.5`）、思考 Token（`F4.3`）與 Vertex Trace ID（`req_vrtx_*`）。
  * **`steps.metadata`（UI 即時氣泡層）** : 當 Step 為 `PLANNER_RESPONSE`（模型回覆）時，系統將 Token 消耗直接複製進該 Step 的 `metadata.model_usage`。
  * **UI 零延遲收益** : 前端在渲染對話時間線與氣泡徽章時，無需跨表 JOIN 即可瞬間顯示「⚡ 17.5k cached / 409 thinking tokens」。

## Prompt Caching 快取生命週期 (Prompt Caching Lifecycle)

* **前綴快取原理與定價模型** :
  * **前綴嚴格匹配** : 基於 Transformer 自注意力因果遮罩（Causal Mask），快取由開頭連續匹配，中間任何字元異動將導致下游快取全面失效。
  * **計費經濟學** : 寫入快取加收 25% 溢價（1.25x），讀取快取享有 90% 折扣（0.1x）；前綴複用 2 次以上即產生巨額成本節省。
* **第 0 輪不快取 User Prompt 之三大決策** :
  * **全域前綴共用** : `System Rules + Tools`（17,535 Tokens）對全專案所有 Session 與子 Agent 100% 相同，錨點切齊在 Tools 結尾可實現全域即時快取命中。
  * **中斷回滾保護** : 第一輪使用者請求具備可中斷性（`Ctrl+C`）與動態性；若對未提交狀態打錨點，中斷時將引發 Prefix Hash 衝突與 GPU 快取崩潰。
  * **多輪滾動吞併** : 每一輪執行並確認提交後，歷史對話逐輪晉升為快取前綴，快取命中率隨對話深入穩步攀升至 99% 以上。

## 延遲指標與效能分析 (Latency & Performance Telemetry)

* **TTFT 與總生成耗時剖析** :
  * **`time_to_first_token` (TTFT, `F11`)** : 首字延遲，標準 `google.protobuf.Duration` `{1: 秒, 2: 納秒}`，實測穩定於 1.2s ~ 2.5s。
  * **`streaming_duration` (總耗時, `F12`)** : 模型串流生成總時長，與輸出長度及思考 Token 呈高度線性正相關（生成速率約 50~70 tokens/sec）。
  * **上游 Trace ID 溯源 (`F4.11`)** : 每次生成皆附帶 Google Cloud Vertex AI 請求序號（`req_vrtx_*`），可直接用於雲端審計與對帳。

## Dashboard 與 Context 頁面操作指南 (UI Views Guide)

* **Dashboard 儀表板檢視** :
  * **Track 1（雲端推論收據）** : 呈現目前選中模型事件關聯之 `gen_metadata` 權威 Token 計費、TTFT 與真實延遲。
  * **Track 2（時間線上下文證據）** : 依據時間線還原當時可見的 System、Tools、History 與最新 Inbound Prompt。
  * **有效輸入折算 (Effective Input Projection)** : 依各模型之快取折扣倍率計算加權等效輸入，客觀評估 Session 成本節省效益。
* **Context 頁面檢視** :
  * **五大分區展示** : System & Rules（系統規約）、Tools（工具 Schema）、Active History（歷史對話）、Compacted Checkpoint（壓縮摘要）、Latest Inbound（當前提示詞）。
  * **Raw Evidence Mode** : 提供帶有各欄位資料來源標籤的結構化 JSON 證據視圖，完整還原底層 Protobuf 欄位。
