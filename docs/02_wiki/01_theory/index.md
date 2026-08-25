# ⚡ 01_theory: 推論物理與數學模型 (Inference Physics & Models)

> 本模組為整個 LLM 記憶中樞的 **底層物理與數學基石**。
> 按照人類認知學習的最優路徑編號，帶領讀者從 Transformer 推論兩階段的硬體特徵出發，推導 KV Cache 的顯存暴增機制，進而掌握前綴快取的生命週期。

---

## 📑 認知學習演進順序 (Learning Pathway)

1. **[[01_Transformer_Prefill_vs_Decode]]**：
   * *認知起點*：解構輸入預填充 (Prefill - 算力受限) 與自回歸解碼 (Decode - 帶寬受限) 的物理對立。
2. **[[02_KV_Cache_Mechanics]]**：
   * *因果遞進*：因為自回歸逐字生成慢，才需要 KV Cache 救場，進而引出顯存呈 $O(N)$ 線性暴增的數學公式與 GQA 架構演進。
3. **[[03_Prompt_Caching_Lifecycle]]**：
   * *技術突破*：為了解救顯存與首字延遲，深入剖析前綴快取如何重用顯存中的現成矩陣與快取固化時序。

---

## 🧭 目錄導航
* 🔙 回到上一層：[[02_wiki/index|Wiki 知識庫總導覽]]
* 🔜 下一模組：[[02_wiki/02_architecture/index|02_architecture: 通用架構與演算法模式]]
