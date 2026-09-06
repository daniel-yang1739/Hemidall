# 🧠 全域拓撲與認知編號反饋記憶庫 (Vault Topology & Numbering Feedbacks)

> 本檔案由 **秘書長 (Chief Secretary)** 維護。
> 記錄所有針對「大腦認知演進編號順序、目錄正交性 (MECE)、Raw 素材消化即刪除、Vault 拓撲檔案樹與目錄職責呈現」的歷史審查建議，明確標註 **🟢 採納 (ACCEPTED)** 與 **🔴 駁回 (REJECTED)**。

---

## 🟢 採納標準 (ACCEPTED Guidelines)

### 📌 條目 04-A：編號必須代表讀者大腦的認知演進路徑，嚴禁隨意編號
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-26 | **累犯次數**：1 次
* **適用檔案**：全庫所有目錄與檔案命名
* **【採納理由】**：確保讀者心智模型層層遞進，讀完 `01` 自然解鎖 `02`。
* **【強制執行標準】**：
  * 編號代表依賴關係：`01_theory`（物理基石）$\to$ `02_architecture`（架構落地）$\to$ `03_planning`（全域企劃）$\to$ `04_meta`（方法論體系）。

---

### 📌 條目 04-B：Raw 素材一旦 100% 提煉完成，必須立即清理刪除
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-26 | **累犯次數**：1 次
* **適用檔案**：`docs/01_raw/`
* **【採納理由】**：貫徹 **Digest & Delete** 原則，避免過渡檔案殘留造成版本混亂。

---

### 📌 條目 04-C：Obsidian Vault 拓撲卡片必須完整呈現全域檔案結構樹與 High-Level 目錄職責解說
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-26 | **累犯次數**：1 次
* **適用檔案**：`04_meta/02_Obsidian_Vault_Topology.md`, `docs/index.md`, `02_wiki/index.md`
* **【採納理由】**：提升讀者初次進入 Vault 時的「全景可視度 (Global Visibility)」，明確定義各資料夾的職責邊界與生命週期。
* **【強制執行標準】**：
  * 必須在 `02_Obsidian_Vault_Topology.md` 與主索引中包含完整的 ASCII 檔案結構樹（附帶 Emoji 與簡要標註）；
  * 必須包含 High-Level 目錄職責表格（定義核心功能、生命週期治理規則與內容範例）；
  * 標題與檔名統一標準化為 `02_Obsidian_Vault_Topology.md`（無需冗餘的 `Git Control` 字眼）。

---

## 🔴 駁回警示 (REJECTED Guidelines - 嚴禁重複提出或實裝)

### 🚫 條目 04-R1：嚴禁把 iThome 30 天的文章草稿拆成 30 篇 Markdown 塞入 02_wiki/
* **決策狀態**：🔴 **REJECTED (已駁回 - 嚴禁採納)**
* **提議角色**：某流程規劃員
* **首次駁回**：2026-08-26
* **【駁回理由】**：
  * 違反「Wiki 是長效終點，iThome 是短期專案」的架構憲法；
  * `02_wiki/03_planning/02_30_Days_Breakdown.md` 作為大綱索引足矣，真正的 30 篇文章草稿必須放在獨立的 `docs/ithome_draft/`，保持 Wiki 資產的精煉。

---

### 📌 條目 04-D：具體 Agent 實體細節必須獨立封裝至 00_agents/ 專屬子模組
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-08-28 | **累犯次數**：1 次
* **適用檔案**：`00_agents/`, `00_agents/antigravity/`, `00_agents/opencode/`
* **【採納理由】**：實現通用抽象模式（`02_architecture/`）與特定廠商/開源 Agent 實體細節的關注點分離（Separation of Concerns），避免架構庫膨脹。
* **【強制執行標準】**：
  * 每個 Agent 獨立目錄（如 `antigravity/`, `opencode/`），標準化拆解為 5 篇核心卡片（01 檔案拓撲 $\to$ 02 資料庫字典 $\to$ 03 遙測狀態機 $\to$ 04 擴充/大腦 $\to$ 05 鑑識 SQL 與 Runbook）。

---

### 📌 條目 04-E：微批次地毯式遞增提煉鐵律 (Incremental Micro-Batch Carpet Distillation - 嚴禁 Big-Bang 批量更新)
* **決策狀態**：🟢 **ACCEPTED (已採納為標準規範)**
* **首次記錄**：2026-09-06 | **累犯次數**：1 次
* **適用檔案**：`docs/01_raw/`, `docs/02_wiki/`, 全體 Agent 蒸餾流程
* **【採納理由】**：一次性吞入大量 Raw 素材並企圖批量輸出，必然導致 LLM 注意力稀釋、工程取證數據遺漏、Protobuf 欄位與二進位細節被過度抽象化甚至幻覺。必須採取「單篇或 1~3 篇高相關微批次」遞增推進，確保每一卡片均達世界級精度。
* **【強制執行標準】**：
  * **嚴禁 Big-Bang 一次性批量更新**：嚴格限制單次提煉作業的消化窗口為 1~3 篇 Raw 素材；
  * **單批次閉環驗收**：每批次必須走完「深入精讀 $\to$ 源碼 Trace $\to$ 編譯產出/增補 Wiki 卡片 $\to$ 驗證審查 $\to$ Digest & Delete 刪除 Raw $\to$ MOC 索引同步」；
  * 前一批次 100% 沉澱並完成驗收清理後，才允許提取下一批次 Raw 檔案，循環前進直至素材池清空。
