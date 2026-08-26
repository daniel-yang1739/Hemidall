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
