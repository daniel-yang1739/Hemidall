---
title: 消失的 31 個步驟與跳號之謎：Google 內部管線過濾機制與全量 SQLite 補齊
type: troubleshooting
created: 2026-08-28
updated: 2026-08-28
status: completed
tags: [troubleshooting, sqlite, jsonl, filtering, permissions, blocked, subagent]
aliases: [Missing_Steps_Gap_Incident, SQLite_Gap_Recovery]
---

# 消失的 31 個步驟與跳號之謎：Google 內部管線過濾機制與全量 SQLite 補齊

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**  
> * **異常現象**：左側列表總數顯示 `STEPS (5889)`，但最新序號卻是 `Step #5919`，中間短少了 30 多個步驟序號（出現跳號現象）。
> * **根本根因**：Google 底層資料庫 SQLite `steps` 表擁有 5,923 筆連續記錄（0~5922，完全無跳號），但 Google 在輸出對話軌跡日誌 `transcript_full.jsonl` 時，**主動過濾了 31 個內部環境設定、背景路徑探測與權限被阻擋（`status = 7`, BLOCKED）的步驟**，以防污染 LLM 的 Context 視窗。
> * **架構修復**：實裝 `MergeMissingSQLiteSteps`，在會話載入時自動從 SQLite `steps` 表補齊遺失記錄，並依元數據賦予 **`⚙️ INTERNAL`** 與 **`🛡️ BLOCKED`** 專屬徽章，使列表完整度達到 100%（5,923 / 5,923）。
> * **雙向鏈接**：對應架構概念參見 [[03_Agent_Storage_and_State_Machine]] 與 [[11_Multi_Agent_Hierarchy_and_Subagent_Token_Economics]]。

---

## 一、現象與問題定義 (Symptom & Problem Definition)

### 1. 觸發情境與異常現象
在去重修復後，使用者發現步驟列表中出現了「消失的序號」：例如從 `#0009` 直接跳至 `#0011`（遺失 `#0010`）、從 `#0059` 跳至 `#0061`（遺失 `#0060`）、甚至在 `#5780` 後直接跳到 `#5782`（遺失 `#5781`）。

### 2. 數據差異對比
* **左側步驟清單顯示**：`STEPS (5889)`
* **最新步驟物理序號**：`Step #5919`
* **差異**：短少整整 30 筆步驟！

---

## 二、根因排查與代碼溯源 (Root Cause Discovery & Deduction)

### 1. 資料庫全量比對法醫取證
我們撰寫 Python 腳本對比 SQLite 與 JSONL 的所有序號：

```python
# 比對兩大儲存層的集合差集
jsonl_indices = set(...) # 長度 5,891
sqlite_indices = set(...) # 長度 5,923 (0 ~ 5922, 100% 連續)
missing = sqlite_indices - jsonl_indices # 恰好 31 個遺失號碼
```

### 2. 遺失的 31 個步驟二進制 Payload 解密
解開 SQLite 裡的這 31 筆二進制資料：
* `Step #10, #60`：Agent 啟動時寫入內部環境配置檔 `write_to_file`；
* `Step #77`：內部目錄探測 `list_directory`；
* `Step #291, #2949`：子代理執行的背景 `git status` 與 `view_file`；
* `Step #4006, #5781`：**`status = 7` (BLOCKED)** 的操作（例如嘗試讀取受保護的 `settings.json` 被安全中介層阻斷）。

### 3. Google 為什麼要過濾這些步驟？
1. **保護 LLM Context**：若將內部讀寫配置檔的雜音塞入歷史，模型會產生幻覺（誤以為是使用者要它讀的）；
2. **終端介面降噪**：人類使用者只關心業務代碼，不關心背景管線。

---

## 三、架構修復方案與實作 (Solution & Architectural Fix)

### 1. 觀測器哲學：全量還原，標明角色
觀測系統的職責是忠實記錄全貌，不能幫使用者隱瞞。
在 [`discovery.go`](file:///Users/daniel_y_yang/Documents/self/ithome2026/agent-observer/internal/adapters/antigravity/discovery.go#L290-L380) 實裝 `MergeMissingSQLiteSteps`：

```go
// 🟢 實裝 SQLite 遺失步驟全量合併引擎 (discovery.go)
func MergeMissingSQLiteSteps(events []core.UnifiedAgentEvent, dbPath string, sessionID string) []core.UnifiedAgentEvent {
    // 1. 讀取 SQLite steps 表中的所有交易
    rows, _ := db.Query("SELECT idx, step_type, status, metadata FROM steps ORDER BY idx ASC")
    
    // 2. 比對出 JSONL 漏掉的步驟並建立 UnifiedAgentEvent
    for rows.Next() {
        if !existingIndices[idx] {
            var stType core.StepType
            var statusLabel = "DONE"
            if status == 7 {
                statusLabel = "BLOCKED" // 標記權限阻斷
            }
            // 辨識 Subagent vs Internal
            agentRole := "INTERNAL"
            if isSubagent(metadata) {
                agentRole = "SUBAGENT"
            }
            
            merged = append(merged, core.UnifiedAgentEvent{
                StepIndex: idx,
                Status:    statusLabel,
                AgentRole: agentRole,
                // ...
            })
        }
    }
    // 3. 依 StepIndex 排序，達成 100% 序列連續！
    sort.Slice(merged, func(i, j int) bool { return merged[i].StepIndex < merged[j].StepIndex })
    return merged
}
```

---

## 四、總結、抗體防禦與 Runbook SOP (Key Takeaways & Diagnostic SOP)

### 1. 通用設計準則 (Design Invariants)
* **交易完整性 (Transaction Integrity)**：底層儲存層的 Primary Key 序號是系統唯一的物理真實，任何過濾或投影必須在 View 層做語意標註，而非在 Data Ingestion 層粗暴丟棄。

### 2. 1 分鐘快速診斷 Runbook SOP
1. **檢查 SQLite 總筆數**：
   ```bash
   sqlite3 ~/.gemini/antigravity-cli/conversations/<session_id>.db "SELECT COUNT(*), MAX(idx) FROM steps;"
   ```
2. **驗證 Observer**：確認 Observer 載入後的總步驟數與 `MAX(idx) + 1` 100% 吻合，無任何中斷跳號。
