# 🛡️ 反饋記憶庫 05：實戰排查、QA 問答與 Runbook 知識化標準 (Troubleshooting, QA & Runbook Guidelines)

> 本文件記錄 Wiki 知識庫中關於「Bug 排查」、「問答 QA」與「Runbook 操作指引」的永久採納規範 🟢 與駁回警示 🔴。

---

## 🟢 [ACCEPTED] 核心採納標準 (Mandatory QA & Runbook Standards)

### 1. 【四段式實戰排查結構 (4-Stage Troubleshooting Framework)】
* **[規則]**：所有記錄 Bug、異常排查或 QA 的卡片，必須具備完整的四段式結構：
  1. **現象與問題定義**（觸發情境、終端錯誤畫面、復現路徑）；
  2. **根因排查與代碼溯源**（排除的假說、具體出錯的代碼行/函式/狀態機變數）；
  3. **架構修復方案與實作**（核心代碼修正、不變量 Invariant 防護）；
  4. **總結、抗體防禦與 Runbook SOP**（如何 1 分鐘內定位、長效防禦原則）。

### 2. 【雙向拓撲鏈接 (Bidirectional Topological Anchoring)】
* **[規則]**：排查卡片與 QA 卡片絕不能是獨立孤島。
  * 核心概念卡片（如 `01_storage/`、`02_metrics/`、`04_ui/`）必須在文末或相關章節鏈接對應的排查案例 `[[...]]`；
  * 排查卡片開頭必須以 `[[...]]` 鏈接其對應的架構概念。

### 3. 【可執行的 Runbook SOP (Actionable Diagnostic SOP)】
* **[規則]**：在總結處必須附帶具體、可立即執行的診斷指令或 Log 特徵檢查清單，使未來的工程師能快速驗證與排查。

### 4. 【多狀態過濾器顯式互斥排除守衛 (Strict Exclusion Guards)】
* **[規則]**：在枚舉型多狀態過濾中，嚴禁使用寬鬆的 Fallback 兜底條件，所有非目標狀態（如 `EXPIRED` 洩漏至 `MISS`）必須在前置 Guard 階段顯式攔截排除。

### 5. 【本地意圖輸入與雲端 GPU 計費時序解耦 (Intent Ingestion vs Inference Settlement)】
* **[規則]**：人類在終端機打字輸入意圖（`USER_INPUT`）時雲端尚未推論，嚴禁為輸入步驟合成虛假的 `[MISS]` 標籤與全額未命中帳單，必須定義為 `Staged Intent` 待下一輪雲端步驟結算。

### 6. 【本地打字與中間步驟嚴禁覆蓋 TTL 計時器 (LastCloudTurnTime 隔離鐵律)】
* **[規則]**：GPU KV-Cache 的過期判定必須死守雲端推論間隔。本地事件（User Input, Tool Output）嚴禁更新 `LastCloudTurnTime`，杜絕時序遮蔽效應。

### 7. 【串流事件接收端必須以實體主鍵 (SessionID, StepIndex) 原地覆蓋去重 (In-Place Deduplication)】
* **[規則]**：面對 `RUNNING` $\to$ `DONE` 的連續串流通知，UI 接收端必須按實體序號查找並原地覆蓋更新，嚴禁無腦 append 導致計數器膨脹與記憶體洩漏。

### 8. 【底層過濾的內部與阻擋步驟必須在 View 層全量補齊並明確標記角色】
* **[規則]**：觀測器必須 100% 忠實還原底層 SQLite 的所有步驟，補齊遺失序號並賦予 `INTERNAL` 與 `BLOCKED` 專屬徽章，絕不替使用者做未告知的過濾。

---

## 🔴 [REJECTED] 駁回警示與邊界 (Rejected Patterns)

### 1. ❌ 【流水帳日記式記錄 (Chatty Debugging Diary)】
* **[反面教材]**：「今天測試發現數字不對，我們看了看代碼，加了一行 if 就修好了。」
* **[駁回原因]**：無問題定義、無根因推導、無架構啟發，屬於低信噪比垃圾資訊。

### 2. ❌ 【孤島式排查記錄 (Isolated Troubleshooting Islands)】
* **[反面教材]**：排查卡片獨立存放在邊緣目錄，與任何架構概念卡片毫無 `[[wiki-links]]` 關聯。
* **[駁回原因]**：破壞知識圖譜互聯性，讀者研讀概念時無法獲取實戰經驗。
