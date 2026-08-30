# 🏛️ 系統架構與代碼實證反饋記憶庫 (Architecture & Code Feedbacks)

> 本檔案由 **秘書長 (Chief Secretary)** 維護。
> 記錄所有針對「5 維度上下文、LCP 演算法、儲存狀態機、雙軌遙測、TUI 盒模型、Go 源碼 AST 對齊」的歷史審查建議，明確標註 **🟢 採納 (ACCEPTED)** 與 **🔴 駁回 (REJECTED)**。

---

## 🟢 採納標準 (ACCEPTED Guidelines)

### 📌 條目 02-A：Wiki 結構體與欄位必須直接 Trace 專案源碼，嚴禁自創名詞
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-26 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/01_Context_5_Dimensions.md`, `02_architecture/02_Token_Calculation_and_LCP.md`
* **【採納理由】**：保障技術文檔與真實 Codebase 的 100% 同步，杜絕文檔技術債。
* **【強制執行標準】**：
  * 必須打開 `agent-observer/internal/core/types.go`，嚴格對齊 `TokenBreakdown` 的 5 大欄位（`SystemTokens`, `ToolsDefTokens`, `ToolResultTokens`, `HistoryTokens`, `ActiveTurnTokens`）。

---

### 📌 條目 02-B：架構決策必須提供全方位客觀 Pros & Cons 矩陣
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-26 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/02_Token_Calculation_and_LCP.md`, `03_planning/03_Tech_Stack_Tradeoffs.md`
* **【採納理由】**：客觀分析方案優缺點與物理代價，使資深架構師被深度說服。
* **【強制執行標準】**：
  * 必須提供 Watcher 模式 vs Proxy 模式、Go vs Python 的多維度對比表格（侵入性、記憶體開銷、高並發模型、生態）。

---

### 📌 條目 02-C：TUI 終端機佈局必須採用 runewidth 視覺寬度硬切與嚴格行數預算
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-26 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics.md`
* **【採納理由】**：徹底根除中文字元 (CJK 寬度為 2) 與 Emojis 引發的隱形自動換行與終端機溢出滾動。
* **【強制執行標準】**：
  * 嚴禁用 `len([]rune)` 進行單行截斷，必須使用 `go-runewidth` 依物理欄寬進行硬切；
  * 垂直行數必須嚴格遵循 $1 + 1 + (H-4) + 1 + 1 \equiv H$ 守恆公式，左右雙欄強制鎖定等高。

---

### 📌 條目 02-D：雙軌遙測必須以官方 Protobuf 帳單為基準並透過倒推滑動窗口校準 5 維度
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-26 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting.md`
* **【採納理由】**：解決本地 Append-Only 永久日誌（數十萬字）與雲端滑動窗口（十餘萬字）之間的截斷失真。
* **【強制執行標準】**：
  * Track 1 直解 SQLite Protobuf 取得官方 Total/Cached 真理；
  * Track 2 從最新步驟往前倒推扣除預算，保證 5 維度總和與官方帳單 100% 數學閉環。

---

### 📌 條目 02-E：歷史步驟清單必須使用「因下果上 ＋ 方案 B 緊湊連續括號」與「100% 統一 Muted 細線顏色」
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-27 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics.md`, `02_architecture/09_History_Explorer_and_Causality_Graph.md`
* **【採納理由】**：直觀表達「輸入因 $\to$ 推論果」的因果閉環，消除無效縮排浪費，並徹底杜絕白灰線條混雜。
* **【強制執行標準】**：
  * 括號頂部 `┌[` 代表雲端果，垂直中幹 `│[` 代表本地因，底部 `└[` 代表使用者輸入；
  * 連接器符號與文字樣式徹底解耦，整條連接線 100% 統一使用 `ColorMuted` 灰色細線。

---

### 📌 條目 02-F：可變高度列表必須採用動態累加行數打包演算法 (Dynamic Line Packing)
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-27 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics.md`, `05_troubleshooting/06_Single_Line_Card_Static_Packing_Blank_Gap.md`
* **【採納理由】**：根除篩選單行步驟時底部出現大片無效留白的嚴重空間浪費。
* **【強制執行標準】**：
  * 嚴禁使用 `availLines / 2` 靜態整數除法；
  * 必須即時遍歷判定每張卡片行高（1 行 vs 2 行）並動態累加至填滿可用高度。

---

### 📌 條目 02-G：多模型聚合度量必須採用 Effective Tokens 折扣矩陣與即時貨幣換算
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-28 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/10_Dashboard_Aggregate_Metrics_and_Multi_Model_Pricing.md`
* **【採納理由】**：不同模型快取折扣率不同（Flash 75% vs Sonnet 90%），唯有標準化為 Effective Tokens 才能真實反映混合調用成本。
* **【強制執行標準】**：
  * 必須透過 `GetModelDiscount(Model)` 取得各模型專屬折扣係數；
  * 支援 Tokens ➔ USD ➔ TWD 循環換算與 Google AI Pro 5,000 RPD 配額模型。

---

### 📌 條目 02-H：Subagent 階層必須嚴格區分本地 Tool 執行（0 GPU Token）與雲端推理計費
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-28 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/11_Multi_Agent_Hierarchy_and_Subagent_Token_Economics.md`
* **【採納理由】**：Subagent 本地執行的 Tool 是在 Mac CPU 跑，不花 GPU 算力；只有雲端輪次才需計費。
* **【強制執行標準】**：
  * 本地 Tool 步驟標記為 0 GPU Token，並穿透標記發起模型與後續打包結算的 Cloud Turn 序號。

---

### 📌 條目 02-K：Host Artifact 必須採用 Evidence Taxonomy，禁止將 Snapshot 或 Transcript 直接命名為 HTTP Request
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-30 | **累犯次數**：1 次
* **適用檔案**：所有 Agent adapter、Context UI、Host storage wiki 與 API trace 文件。
* **【問題本質】**：本機儲存可能保存持久化 context snapshot 或 execution transcript，但沒有保存 HTTP body；以「official wire request」呈現會讓使用者對證據能力做錯誤判讀。
* **【強制執行標準】**：每個 context 欄位必須標示 `persisted snapshot`、`observed telemetry`、`transcript-derived fallback`、`filesystem discovery` 或 `inferred`；身份是 system prompt 的 subsection，MCP owner 未有 schema 時必須標示 unknown。

## 🔴 駁回警示 (REJECTED Guidelines - 嚴禁重複提出或實裝)

### 🚫 條目 02-R1：嚴禁為了追求理論通用性而將 Pure Go 實作重構為 Go/Python 雙語言混合
* **決策狀態**：🔴 **REJECTED (已駁回 - 嚴禁採納)**
* **提議角色**：某 Python 生態支持者
* **首次駁回**：2026-08-26
* **【駁回理由】**：
  * 本專案的根本核心約束是「單一靜態二進制檔分發、零 Python 虛擬環境依賴、常駐記憶體 < 15MB RSS」；
  * 引入 Python 會破壞極致輕量與使用者體驗的初衷。

---

### 🚫 條目 02-R2：嚴禁在終端機 TUI 核心卡片中引入無關的 Web CSS Grid/Flex 概念
* **決策狀態**：🔴 **REJECTED (已駁回 - 嚴禁採納)**
* **提議角色**：某 Web 開發者讀者
* **首次駁回**：2026-08-26
* **【駁回理由】**：
  * 終端機是離散等寬字元網格，沒有 Web 瀏覽器的次像素渲染與流式排版；
  * 過度展開 Web CSS 概念會造成讀者對終端機底層物理的理解混淆。

---

### 📌 條目 02-I：跨 Agent 本機存儲鑑識必須提供完整 XDG / 雙層 SQLite 字典與防鎖庫唯讀連線範式
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-28 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/12_Google_Antigravity_Host_Storage_and_Forensics_Schema.md`, `02_architecture/13_OpenCode_Host_Storage_and_Forensics_Schema.md`
* **【採納理由】**：確保任何第三方或觀測工具在對 Agent 本機數據進行鑑識分析時，不與 Agent 本體產生 SQLite 鎖庫衝突，且欄位定義 100% 完整。
* **【強制執行標準】**：
  * 必須包含全量資料表欄位字典、Protobuf / JSON 載荷層級解析、實用 SQL 查詢腳本，並強制指定 `file:...db?mode=ro&_journal=WAL` 唯讀連線。

---

### 📌 條目 02-J：動態會話觀測必須採用單一專注 Watcher 模式與無狀態透鏡視圖投影
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-28 | **累犯次數**：1 次
* **適用檔案**：`02_architecture/12_Dynamic_Watcher_Hub_and_Live_Streaming_Engine.md`, `02_architecture/13_Stateless_Lens_and_Deterministic_Projection.md`
* **【採納理由】**：消除多會話並發背景輪詢造成的 CPU/FD 浪費，並確保視圖與磁碟母體保持 100% 確定性一致。
* **【強制執行標準】**：
  * 後台永遠維持 1 個活動 Watcher Goroutine，切換時透過 Context 立即銷毀並重新開箱新實例；
  * TUI 必須實作 `event.SessionID` 標籤守衛，防止全域 Channel 殘留事件引發跨會話污染；
  * 第一幀必須採用同步注水（Sync Hydration）載入全量歷史，後續變更透過 250ms Live Tail 原地覆蓋。
