# 📥 01_raw: 原始素材收集池 (Raw Materials Pool)

> 本目錄存放未加工的原始素材、API Traces、終端日誌快照、架構討論與企劃檔案。

---

## 📜 核心管理規範 (Raw Management Constitution)

### 1. 檔名時間戳記規範（必須精確至秒）
* 所有放置於 `docs/01_raw/` 的企劃、日誌與素材檔案，檔名**必須嚴格包含「秒級」時間戳記**：
  $$\text{YYYY-MM-DD\_HHmmss\_<description>.md}$$
  * ✅ **正確範例**：`2026-08-27_125034_agent_step_mental_model_and_clarity_plan.md`
  * ❌ **錯誤範例**：`2026-08-27_agent_step_mental_model_and_clarity_plan.md`（未包含秒數，易發生同日覆蓋碰撞）
* **目的**：保證檔案在檔案系統中能按照真實時序嚴格排序，杜絕同日多次反覆討論時發生檔名碰撞或覆蓋。

### 2. 企劃版本演進與歷史保留原則 (Plan History Retention)
* 當因使用者反饋或架構深入而需更新企劃時，**嚴禁粗暴刪除過往討論紀錄**；
* 企劃文件必須完整保留：
  1. **原始問題定義與背景**
  2. **歷史決策與推論過程**
  3. **針對質疑點的補充解答與反覆迭代版本**（以 Version / Milestone 章節追加或以新秒級檔案記錄）。

### 3. Digest & Delete 提煉原則
* 當原始素材被 `wiki-distiller` 100% 提煉、核實並整合進 `docs/02_wiki/` 永久知識庫後，原始檔案始可被自動刪除清理，保持素材池純淨。

---

## 📑 當前素材池清單
* `2026-08-27_155000_dashboard_aggregate_metrics_and_multi_turn_trend_charts_plan_v1.md` (v1.0 初代條列企劃)
* `2026-08-27_162500_dashboard_aggregate_metrics_and_multi_turn_trend_charts_plan_v2.md` (v2.0 方案 A/B/C 提案與多模型折扣矩陣)
* `2026-08-27_223500_harness_middleware_context_compaction_and_injection_mechanics.md` (架構研究：中介層上下文壓縮、Sidecar 模型調用與自建 Proxy 介入注入機制)
* `2026-08-27_235000_cloud_telemetry_forensics_safety_classifier_and_usage_query_mechanics.md` (法醫取證：SQLite/Protobuf 本地儲存解碼、Safety Classifier 免費安檢機制、前綴快取物理推導與 /usage 遠端查詢架構)
* `2026-08-27_235900_dashboard_cost_converter_and_currency_toggle_plan.md` (企劃書：Dashboard 即時金額換算、多模型計價矩陣與快捷鍵切換 $ 實裝企劃)

