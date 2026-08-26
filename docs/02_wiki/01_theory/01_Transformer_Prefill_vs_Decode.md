---
title: Transformer 兩階段推論物理：Prefill 預填充 vs. Decode 自回歸解碼
type: concept
created: 2026-08-26
updated: 2026-08-26
status: completed
tags: [theory, transformer, inference, prefill, decode, kv-cache, memory-bandwidth, flop-bound, concrete-walkthrough]
aliases: [Prefill vs Decode, 推論兩階段物理, 自回歸推論本質, GEMM vs GEMV]
---

# 🧠 Transformer 兩階段推論物理：Prefill 預填充 vs. Decode 自回歸解碼

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 現代大語言模型（LLM）的推論過程並非均勻恆定的計算流，而是在物理上嚴格劃分為兩個計算特性、硬體瓶頸與顯存行為完全對立的階段：
> 1. **Prefill Phase (輸入預填充階段 / Prompt Processing)**：以高度矩陣並行（GEMM）一次性處理整個輸入 Prompt。運算受限於 **GPU 算力峰值 (Compute-Bound)**，是 [[03_Prompt_Caching_Lifecycle|前綴快取 (Prompt Caching)]] 唯一能夠生效的黃金區間。
> 2. **Decode Phase (自回歸解碼階段 / Token Generation)**：以單字串行循環（GEMV）逐一生成後續 Token。運算受限於 **GPU 顯存帶寬 (Memory-Bound)**，每生成一個新字都必須將數十 GB 的模型權重與歷史 [[02_KV_Cache_Mechanics|KV Cache]] 從顯存搬入晶片運算，無法被前綴快取加速。

---

## 🔍 一、技術背景：為什麼推論會被撕裂為兩個階段？

在標準 Transformer Decoder-Only 架構（如 GPT-4, LLaMA-3, Gemini）中，文字生成依賴 **自回歸機制（Autoregressive Property）**：
$$
P(w_1, w_2, \dots, w_T) = \prod_{t=1}^{T} P(w_t \mid w_1, w_2, \dots, w_{t-1})
$$

這意味著模型在預測第 $t$ 個字時，必須依賴前面 $t-1$ 個字作為上下文。這種數學特性直接導致了兩種完全不同的硬體執行模式：

1. **使用者發送請求的瞬間**：前面所有的 Prompt 文字（長度 $N$）已經完全確定。模型不需要等上一個字算完才能看下一個字，因此可以透過注意力遮罩（Causal Masking）**一次性將所有 $N$ 個文字送進 GPU Tensor Cores 進行矩陣並行相乘**，這就是 **Prefill**。
2. **模型開始輸出的瞬間**：未來的文字在宇宙中尚不存在。模型必須先算出第 $N+1$ 個 Token 的機率分佈並進行採樣，才能將該 Token 作為下一步的輸入去計算第 $N+2$ 個 Token。這種強制性的因果相依，將運算鎖死在 **嚴格串行循環** 中，這就是 **Decode**。

---

## 🏛️ 二、兩階段推論物理流程圖與精讀指引

```mermaid
flowchart LR
    subgraph Prefill ["1. Prefill 階段 (輸入預填充 / Prompt Processing)"]
        direction TB
        A["一次性載入整個 Context<br/>(長度 N，例如 10,000 Tokens)"] --> B["GEMM 矩陣相乘運算<br/>[Batch, N, Hidden] × [Hidden, Hidden]"]
        B --> C["⚡ 檢查並重用現成前綴 KV Cache<br/>(命中時大幅縮短首字延遲 TTFT)"]
        C --> D["批次寫入未命中部分之 Key/Value<br/>至 GPU HBM 顯存尾端"]
    end

    subgraph Decode ["2. Decode 階段 (自回歸逐字生成 / Generation)"]
        direction TB
        E["輸入前一步生成的單一 Token<br/>(長度 = 1 Token)"] --> F["GEMV 矩陣-向量相乘運算<br/>[Batch, 1, Hidden] × [Hidden, Hidden]"]
        F --> G["讀取歷史全部 KV Cache<br/>計算 Attention 機率分佈並採樣"]
        G --> H["🔥 消耗真實算力生成新字<br/>將新生成的 Key/Value 追加至顯存尾端"]
    end

    Prefill -->|產出第 1 個 Token| Decode
    H -.->|作為下一步輸入 (循環 N 次)| E
```

### 📖 圖表深度精讀指南 (Diagram Walkthrough)

1. **【核心視野】**：本圖揭示了一個請求從進門到結束的資料形態轉換。左側 Prefill 負責將龐大的初始輸入轉化為初始狀態；右側 Decode 則是一個封閉的自回歸反饋環（Feedback Loop）。
2. **【看圖路徑 (Step-by-Step)】**：
   * **步驟 1 (進入 Prefill)**：長度為 $N$ 的完整提示詞進入模型，觸發 GEMM 稠密矩陣運算。此時系統會比對前綴快取，若命中則跳過重複計算，直接將狀態寫入顯存。
   * **步驟 2 (跨越分界點)**：Prefill 結束時，模型產出第 1 顆 Token（此時耗時即為 **TTFT 首字延遲**），隨即交棒給 Decode。
   * **步驟 3 (Decode 自回歸循環)**：每次只餵入長度為 1 的向量，觸發 GEMV 運算，並透過虛線箭頭不斷回灌自身，直到吐出 `<EOS>` 結束符號。
3. **【色彩與物理意義】**：
   * 🟢 **綠色 (快取操作區)**：Prefill 階段具備前綴確定性，是顯存快取讀取與費率折扣的唯一作用區。
   * 🔴 **紅色 (新生算力區)**：Decode 階段每次都在產生未知文字，必須即時消耗 GPU 算力與帶寬。

---

## 🎯 三、極簡 3-Token 輸入 $\to$ 2-Token 生成之端到端演繹 (Concrete Trace)

為了徹底看清微架構底層資料流，我們帶入極簡真實輸入，精確演繹資料與顯存的變遷：

* **極簡輸入 (Input Prompt)**：`["What", "is", "Go"]`（長度 $N = 3$ Tokens）
* **目標生成 (Target Tokens)**：`["Go", "is"]`，隨後輸出 `<EOS>` 終止。

```text
════════════════════════════════════════════════════════════════════════════════
【階段 1：Prefill 預填充 (GEMM 矩陣乘法，單次批處理)】
  輸入張量 : Shape [1, 3, 4096] (一次性吞入 ["What", "is", "Go"])
  運算特性 : 算力密集 (Compute-Bound)，Tensor Cores 滿載並行相乘
  顯存寫入 : 同時為 Token 0, 1, 2 計算 Key/Value 矩陣並寫入 HBM
            KV Cache 增量 = 3 × 320 KB = 960 KB
  階段產出 : 產出第 1 個 Token -> "Go" (耗時即為 TTFT 首字延遲)
════════════════════════════════════════════════════════════════════════════════
【階段 2：Decode Step 1 (GEMV 矩陣-向量相乘，自回歸第 1 輪)】
  輸入張量 : Shape [1, 1, 4096] (僅傳入單一 Token "Go")
  顯存搬移 : 為了這 1 個字，從 HBM 讀取 140GB 權重 + 960 KB 歷史 KV (Tokens 0..2)
  運算特性 : 帶寬密集 (Memory-Bound)，算力核心 95% 時間處於等待資料搬移
  顯存寫入 : 為 Token 3 ("Go") 計算 Key/Value，追加至顯存尾端 (KV 總量 = 1,280 KB)
  階段產出 : 產出第 2 個 Token -> "is"
════════════════════════════════════════════════════════════════════════════════
【階段 3：Decode Step 2 (GEMV 矩陣-向量相乘，自回歸第 2 輪)】
  輸入張量 : Shape [1, 1, 4096] (僅傳入單一 Token "is")
  顯存搬移 : 再次從 HBM 完整搬移 140GB 權重 + 1,280 KB 歷史 KV (Tokens 0..3)
  階段產出 : 採樣命中 <EOS> 結束符號，宣告生成完畢！
════════════════════════════════════════════════════════════════════════════════
【最終 Output 結算】
  * 總生成內容 : "Go is"
  * Prefill 耗時 : 1 次 GEMM (高算術強度)
  * Decode 耗時  : 2 次 140GB 權重全量顯存搬移 (受限於 HBM 帶寬)
```

---

## 📊 四、底層硬體與運算特徵全方位對比矩陣

| 比較維度 | ⚡ **Prefill 階段 (預填充)** | ⏳ **Decode 階段 (自回歸解碼)** |
| :--- | :--- | :--- |
| **輸入張量形狀** | $[B,\ N,\ D]$（批次大小 $B$、序列長度 $N$、隱藏維度 $D$） | $[B,\ 1,\ D]$（序列長度恆為 $1$） |
| **底層 BLAS 運算** | **GEMM (General Matrix-Matrix Multiply)** | **GEMV (General Matrix-Vector Multiply)** |
| **硬體運算瓶頸** | **Compute-Bound (算力受限)**：GPU Tensor Cores 滿載 | **Memory-Bound (帶寬受限)**：GPU 顯存讀寫帶寬 (HBM) 成為瓶頸 |
| **算術強度 (FLOPs/Byte)**| **極高**（高達數百 FLOPs/Byte，充分利用硬體浮點算力） | **極低**（接近 1~2 FLOPs/Byte，大部分時間在等待資料搬移） |
| **KV Cache 行為** | **【前綴讀取 + 批次寫入】**：比對前綴直接拿現成 KV，未命中處一次寫入。 | **【動態逐字追加 (Append)】**：每生成 1 顆 Token，將其 KV 向量追加至顯存尾端。 |
| **Prompt Caching** | 🟢 **100% 作用於此階段**（直接跳過前綴 Prefill 運算） | 🔴 **完全不適用**（生成新生文字無法預先快取） |
| **核心優化目標** | **TTFT (Time To First Token)**：首字延遲越短越好 | **TPS (Tokens Per Second)**：逐字吐字吞吐量越快越好 |

---

## 🔬 五、深入微架構：為什麼 Decode 階段受限於顯存帶寬？

* **運算強度的數學定義**：
  $$\text{Arithmetic Intensity} = \frac{\text{總計算量 (FLOPs)}}{\text{顯存搬移量 (Bytes)}}$$
* **Decode 階段的物理瓶頸**：
  * 在 Decode 階段，模型每產生 **1 個 Token**，神經網絡的每一層權重（例如 70B 模型的 140GB 參數）以及歷史上所有累積的 KV Cache 矩陣，**都必須從慢速的 GPU HBM 顯存完整搬入極小但極快的晶片內 SRAM 快取中**。
  * 搬了 140GB 的數據進去，只為了對「單一 Token」做一次矩陣相乘（僅幾百萬次運算）。
  * **結論**：GPU 強大的 Tensor Core 在 95% 的時間裡處於「飢餓等待顯存傳輸數據」的閒置狀態。因此，Decode 的生成速度完全由 GPU 的 **顯存帶寬 (Memory Bandwidth, 如 A100 的 2.0 TB/s)** 決定！

---

## 🔗 六、相關概念與延伸閱讀
* [[02_KV_Cache_Mechanics]]：深入了解 KV Cache 在顯存中的矩陣結構與顯存暴增公式。
* [[03_Prompt_Caching_Lifecycle]]：解析為什麼 Model Response 在當前回合是 Uncached，而下一輪會變成 Cached。
* [[02_architecture/01_Context_5_Dimensions|Context 5 維度模型]]：探討 Agent 如何在 Prefill 階段塞滿 128k 上下文。
