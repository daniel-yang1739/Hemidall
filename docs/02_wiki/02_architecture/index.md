# 🏛️ 02_architecture: 通用架構與演算法模式 (Architectures & Algorithms)

> 本模組建立在 `01_theory/` 的底層物理基石之上，深入剖析自主 AI Agent 的 **上下文結構、快取推導演算法、工業級儲存模式、雙軌遙測會計、載荷組裝協議與全螢幕 TUI 引擎設計**。

---

## 📑 認知學習演進順序 (Learning Pathway)

1. **[[01_Context_5_Dimensions]]**：
   * *結構解剖*：將龐大的 Context 解構為 5 個具備不同生命週期與膨脹速率的維度，鎖定膨脹元兇（Tool Results）。
2. **[[02_Token_Calculation_and_LCP]]**：
   * *演算法實作*：利用原生 BPE 分詞器與最長公共前綴（LCP）演算法，在本地零侵入式推導快取命中率。
3. **[[03_Agent_Storage_and_State_Machine]]**：
   * *工業級儲存*：解密商業級 Agent 如何運用雙層 SQLite 7 表結構、Protobuf 二進制與 100KB 滾動切片雙軌日誌實現高並發與零鎖競爭。
4. **[[04_Service_Plan_Agent_Observer]]**：
   * *系統落地*：遵循 Clean Architecture 設計原則，將上述理論落地為極致輕量（< 15MB RSS）的 `agent-observer` 觀測服務。
5. **[[05_Model_Payload_and_API_Traces]]**：
   * *通訊協議與載荷*：解剖 Context 4 大板塊（System, Tools, Trajectory, Active）的物理拼裝順序與底層 API 通訊 JSON Schema。
6. **[[06_Dual_Track_Telemetry_and_Window_Accounting]]**：
   * *雙軌遙測與會計*：實作 Track 1 官方物理帳單與 Track 2 本地 5 維度解剖，透過倒推滑動窗口（Reverse Sliding Window）消除本地與雲端截斷失真。
7. **[[07_TUI_Engine_and_Terminal_Layout_Mechanics]]**：
   * *終端機佈局物理*：以 Bubbletea 與 Lipgloss 打造全螢幕互動介面，攻克盒模型幾何陷阱、中文字元（CJK）2 倍列寬與軟換行虛擬緩衝區。
8. **[[08_Interactive_Session_Switching_and_Anti_Jitter]]**：
   * *會話快切與防抖動*：全域會話快切中樞（`Ctrl+p`）、動態目錄發現與歷史步驟防抖動鎖定機制（Anti-Jitter Lock）。
9. **[[09_History_Explorer_and_Causality_Graph]]**：
   * *歷史步進瀏覽器與因果圖譜*：雙軌正交過濾引擎（`[T:Type]` 與 `[C:Cache]`）、增量搜尋、方案 B 緊湊括號封裝與雙向因果跳轉（`p`/`c`/`C`）。

---

## 🧭 目錄導航
* 🔙 回到上一層：[[02_wiki/index|Wiki 知識庫總導覽]]
* 🔙 上一模組：[[02_wiki/01_theory/index|01_theory: 推論物理與數學模型]]
* 🔜 下一模組：[[02_wiki/03_planning/index|03_planning: 系列藍圖與規劃規格]]
* 🔜 實戰手冊：[[02_wiki/05_troubleshooting/index|05_troubleshooting: 實戰故障排查與 Runbook]]
