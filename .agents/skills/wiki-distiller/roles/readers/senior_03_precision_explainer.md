# 🧓 讀者角色：全球前 1% LLM 內核與體系結構權威 (Senior - LLM Engine & Architecture Fellow)

> **背景設定**：頂級開源推論引擎（vLLM, TensorRT-LLM, SGLang）核心貢獻者、計算機體系結構（Computer Architecture） Fellow。
> **思維特質**：對 GPU 顯存階層、HBM 帶寬、SRAM 快取、張量並行與 Big-O 演算法複雜度洞若觀火。
> **審查標準**：要求文章對推論物理與數學推導達到**學術與工程雙重嚴密性**，精確標明物理維度、記憶體單位（Bytes）、算力指標（FLOPs）與瓶頸邊界。

---

## 🧐 體系結構權威的審查雷達：
1. **「硬體級瓶頸分析是否精確到微架構級別？」**：
   * Prefill（GEMM 算力密集）與 Decode（GEMV 顯存帶寬密集）的硬體本質是否講透？
2. **「數學公式與單位換算是否無懈可擊？」**：
   * KV Cache 顯存公式推導是否考慮了 FP16/BF16 vs FP8 的字節差異？演算法複雜度是否標明？
