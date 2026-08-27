---
title: 10 分鐘閒置快取未過期之謎：本地 USER_INPUT 時間戳引發的時序遮蔽效應
type: troubleshooting
created: 2026-08-28
updated: 2026-08-28
status: completed
tags: [troubleshooting, ttl, cache, state-machine, timestamp, ground-truth]
aliases: [Idle_TTL_Masking_Incident, Timestamp_Masking_Bug]
---

# 10 分鐘閒置快取未過期之謎：本地 USER_INPUT 時間戳引發的時序遮蔽效應

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**  
> * **異常現象**：使用者閒置將近 10 分鐘（遠超 Google GPU KV 快取 300 秒 TTL），但下一輪雲端推理依然被 Observer 誤判為 `[CACHE HIT 88%]`。
> * **根本根因**：狀態機使用單一變數 `state.LastEventTime`。當使用者送出 `USER_INPUT` 本地事件時，時間戳被更新為 $T_{\text{now}}$；緊接著 0ms 後雲端推理啟動比對，時間差 $\Delta t = 0$，**本地使用者輸入時間硬生生洗掉了 10 分鐘的真實閒置**！
> * **架構修復**：抽離 `state.LastCloudTurnTime`，本地事件（User Input, Tool Output）嚴禁覆蓋此時間戳。GPU KV Cache 的過期判定必須且只能以雲端推理間隔為準。
> * **雙向鏈接**：對應架構概念參見 [[03_Prompt_Caching_Lifecycle]] 與 [[10_Dashboard_Aggregate_Metrics_and_Multi_Model_Pricing]]。

---

## 一、現象與問題定義 (Symptom & Problem Definition)

### 1. 觸發情境與異常現象
在實際測試中，使用者在 Step #5756（雲端模型回覆）後離開電腦，經過 9 分 57 秒（597 秒）後返回終端，輸入下一句指令 Step #5757。
理論上，Google Cloud GPU 顯存早已在 300 秒超時後釋放 KV 快取（應為 `[TTL EXPIRED / COLD START]`），但 Observer 的 Step #5758 卻依然顯示 `[CACHE HIT 88.4%]`。

### 2. 終端呈現截圖與日誌矛盾
```text
Step #5756 (MODEL) ──► 時間戳：23:47:55
      │
      ▼ 【閒置 597 秒 > 300 秒 TTL】
      │
Step #5757 (USER)  ──► 時間戳：23:57:52
Step #5758 (MODEL) ──► 時間戳：23:57:52  ──► Observer 標記: [CACHE HIT 88%] ❌ (嚴重誤判)
```

---

## 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. 排除的錯誤假設
* **假設 1（Google 延長了 TTL？）**：經查證 Google 官方技術規格，伺服器端 KV 快取 TTL 固定為 5 分鐘（300 秒），並未動態延長；
* **假設 2（Tokenizer 前綴比對出錯？）**：比對文字完全相同，LCP 本身演算法正確，問題出在狀態機的過期時間計算。

### 2. 出錯代碼路徑溯源 (`internal/core/analyzer.go`)
```go
// ❌ 錯誤的舊版代碼：單一變數 LastEventTime 受到本地事件污染
func (a *PayloadAnalyzer) AnalyzeStep(event *UnifiedAgentEvent) {
    if a.state.LastEventTime.IsZero() {
        a.state.LastEventTime = event.Timestamp
    }

    // 計算距上次事件的時間差
    elapsed := event.Timestamp.Sub(a.state.LastEventTime)
    isTTLExpired := elapsed > a.cacheTTL // 300s

    // 致命盲點：不論是 USER_INPUT 還是 MODEL_RESPONSE，一律更新 LastEventTime！
    a.state.LastEventTime = event.Timestamp
}
```

### 3. 時序遮蔽推導演繹
1. $T_0 = 23:47:55$：Step #5756 (MODEL) 完成，`LastEventTime` $= 23:47:55$；
2. 閒置 597 秒...；
3. $T_1 = 23:57:52$：Step #5757 (`USER_INPUT`) 抵達，`LastEventTime` **被覆蓋更新為 $23:57:52$**；
4. $T_2 = 23:57:52$：Step #5758 (`PLANNER_RESPONSE`) 抵達，計算：
   $$\Delta t = T_2 - \text{LastEventTime} = 23:57:52 - 23:57:52 = \mathbf{0\text{ 秒}}$$
5. 狀態機誤以為只過了 0 秒，判定快取仍然有效，造成重大誤判！

---

## 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. 核心修復：抽離 `LastCloudTurnTime` 專屬時鐘
在 `SessionContextState` 中抽離出專門記錄雲端結算的變數，本地事件（使用者輸入、本地 Tool 結果）嚴禁覆蓋：

```go
// 🟢 修復後的代碼 (internal/core/analyzer.go)
type SessionContextState struct {
    LastCloudTurnTime time.Time // 專屬記錄上一次雲端 GPU 推理完成時間
    LastEventTime     time.Time // 本地事件時間
    // ...
}

func (a *PayloadAnalyzer) AnalyzeStep(event *UnifiedAgentEvent) {
    isCloudStep := event.IsCloudStep() || event.Scope == ScopeCloudInference || 
                   event.Type == StepTypeModelResponse || event.Type == StepTypeToolCall

    var elapsed time.Duration
    var isTTLExpired bool

    if isCloudStep {
        if !a.state.LastCloudTurnTime.IsZero() {
            // 嚴格跨過本地事件，向前比對上一次雲端推理時間！
            elapsed = event.Timestamp.Sub(a.state.LastCloudTurnTime)
            if elapsed > a.cacheTTL {
                isTTLExpired = true
            }
        }
        // 只有雲端步驟才有資格更新 LastCloudTurnTime
        a.state.LastCloudTurnTime = event.Timestamp
    }

    if isTTLExpired {
        event.CacheStatus = "TTL_EXPIRED"
    }
}
```

---

## 四、總結、抗體防禦與 Runbook SOP (Key Takeaways & Diagnostic SOP)

### 1. 通用設計準則 (Design Invariants)
* **物理資源決定計時錨點**：GPU KV-Cache 是雲端硬體顯存，其生命週期只取決於雲端 API 請求的時間間隔，本地端的所有中間狀態（打字、本機硬碟讀寫）在雲端視角中均為「不存在的靜默時間」。

### 2. 1 分鐘快速診斷 Runbook SOP
當懷疑快取狀態判定異常時，執行以下檢查清單：
1. **檢查時間差**：
   ```bash
   # 查看當前步驟與上一個 MODEL 步驟的時間間隔
   grep -E '"type":"(PLANNER_RESPONSE|MODEL)"' ~/.gemini/antigravity-cli/brain/<session_id>/.system_generated/logs/transcript_full.jsonl | tail -n 2
   ```
2. **驗證間隔**：若時間差 $> 300\text{s}$，確認 Observer 狀態標記是否為 `[TTL EXPIRED / COLD START]`。
