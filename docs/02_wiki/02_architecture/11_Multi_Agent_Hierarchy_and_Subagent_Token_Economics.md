---
title: Multi-Agent 協同階層、Subagent 獨立 Context 生命週期與權限阻斷機制
type: architecture
created: 2026-08-28
updated: 2026-08-28
status: completed
tags: [multi-agent, subagents, context, tokens, security, permissions, blocked]
aliases: [Multi_Agent_Hierarchy, Subagent_Token_Economics, Permission_Blocked_Mechanics]
---

# Multi-Agent 協同階層、Subagent 獨立 Context 生命週期與權限阻斷機制

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**  
> * **獨立 Context 視窗**：Subagent（子代理）不是單純的函式呼叫，而是擁有**專屬 System Prompt、獨立工具集與獨立 Context 視窗**的並行進程，其 Token 消耗與快取生命週期與主會話解耦。
> * **本地 Tool 0 GPU Token 原則**：Subagent 於本地執行的工具操作（如 `git status`、`view_file`）在 Mac 本機 CPU/硬碟運作（0 GPU Token），其 Payload 成果在回傳後的下一個雲端輪次打包結算。
> * **`status = 7` (BLOCKED) 沙盒防禦**：當 Agent 嘗試讀取受保護檔案（如 `settings.json`）、使用者拒絕高危指令或任務被中止時，Harness 在本地邊界直接阻斷（0 GPU Token 消耗）。
> * **雙向鏈接**：本架構依賴 [[01_Context_5_Dimensions]] 與 [[03_Agent_Storage_and_State_Machine]]，實戰排查參見 [[09_Harness_Internal_Plumbing_Filtering_and_SQLite_Gap_Recovery]]。

---

## 一、概念緣起：從單代理走向多代理階層 (Main vs. Subagents)

在複雜軟體工程中，單一 Agent 的上下文極易被超大檔案或多檔案搜尋撐爆。現代 Agent 框架（如 Antigravity / Gemini CLI）引進了 **Multi-Agent 階層架構**：主控 Agent（Main Planner）將子任務（如代碼搜尋、模組測試）派發給專門的子代理（Subagent Worker）平行執行。

然而，若觀測器將所有動作混為一談，會導致上下文飽和度誤判與計費失真。因此，Agent Observer 建立了專屬的 **`AgentRole`（角色分類）** 與 **`ScopeSubagent`（子代理範疇）**。

---

## 二、Subagent 的獨立 Context 視窗與生命週期

```mermaid
sequenceDiagram
    autonumber
    participant U as 👤 Human Client
    participant M as 👑 Main Planner (Main Context)
    participant H as 🛡️ Local Harness / Guardrail
    participant S as 👥 Subagent Worker (Sub Context)
    participant C as ☁️ Gemini Cloud API

    U->>M: 1. 使用者提問與指派複雜任務
    M->>C: 2. Main Cloud Turn #1: 決定派發子任務 (invoke_subagent)
    C-->>M: 3. 回傳 invoke_subagent 指令
    M->>S: 4. Fork 獨立子進程 (開闢全新 Context 視窗)
    Note over S: 🌱 載入專屬 System Prompt<br/>與獨立 Tools 定義 (Cold Start)
    S->>C: 5. Subagent Cloud Turn #1: 規劃搜尋指令
    C-->>S: 6. 回傳 run_command: git status
    S->>H: 7. 發起本地指令執行
    H-->>S: 8. 本機執行完畢 (0 GPU Token)
    S->>C: 9. Subagent Cloud Turn #2: 彙整成果並回報
    C-->>S: 10. 回傳 send_message 結果
    S-->>M: 11. 回傳子任務結論
    M->>C: 12. Main Cloud Turn #2: 整合成果回覆使用者
    C-->>M: 13. 回覆最終解答
    M-->>U: 14. 呈現完整結果

    style M fill:#1a1b26,stroke:#7aa2f7,color:#c0caf5
    style S fill:#1f2335,stroke:#bb9af7,color:#c0caf5
    style H fill:#3b1219,stroke:#f7768e,color:#ff9e3b
    style C fill:#1a2b23,stroke:#9ece6a,color:#c0caf5
```

### 🔍 圖表 4 維度深度精讀指南 (Diagram Walkthrough)
1. **【核心視野】**：展示主控 Agent 與子代理在獨立 Context 視窗與本機工具調用間的完整交互時序。
2. **【看圖路徑 (Step-by-Step)】**：
   * 步驟 1~4：主會話消耗 Prompt Token 發起 `invoke_subagent`；
   * 步驟 5~10：子代理在獨立視窗進行多次 Cloud Turn 與本地 Tool 執行；
   * 步驟 11~14：子代理成果回傳主會話，由主會話進行最終推論。
3. **【色彩與符號物理意義】**：
   * 藍色（Main）：主會話 Context 邊界；
   * 紫色（Subagent）：完全隔離的子 Context 視窗；
   * 紅色（Harness）：本機安全與執行中介層；
   * 綠色（Cloud）：雲端 GPU 計費結算點。
4. **【底層隱藏工程細節】**：子代理的步驟 8（本地 Tool）完全不耗費 GPU Token，其 Token 於步驟 9（Subagent Cloud Turn #2）送往雲端時才被計費。

---

## 三、Subagent 的 Token 經濟學與四階段計費拆解

| 執行階段 | 動作內容 | 消耗 Token 類型 | 計費與 Quota 歸屬 | 是否使用快取？ |
| :--- | :--- | :---: | :---: | :---: |
| **① 主控派發** | Main Agent 呼叫 `invoke_subagent` 傳入任務指令 | Main Prompt Tokens | **主會話配額** | ✅ 依賴主會話 Prefix Cache |
| **② 子代理初始化** | Subagent 載入獨立的 System Prompt 與工具定義 | Subagent Inbound Context | **子代理獨立配額** | ❄️ 首次為 Cold Start |
| **③ 子代理推理** | Subagent 思考、決策並調用模型（如 `gemini-3.7-flash`） | Subagent GPU Output Tokens | **子代理獨立配額** | ✅ 連續多輪命中子快取 |
| **④ 本地工具執行** | Subagent 在本機執行 `run_command` 或 `view_file` | **0 GPU Tokens** (純本機 CPU) | **$0.00 (完全免費)** | ❌ 本地離線運作 |

---

## 四、權限邊界攔截與 `status = 7` (BLOCKED) 沙盒防禦機制

### 1. 觸發情境分類
在 SQLite `steps` 表中，標記為 `status = 7` 的步驟代表操作在送出執行前，被安全守護進程（Harness Guardrail）直接阻斷：

```text
┌───────────────────────────────┬─────────────────────────────────────────────────────────────┐
│ 攔截類型                      │ 具體觸發案例與原因                                          │
├───────────────────────────────┼─────────────────────────────────────────────────────────────┤
│ 1. 目錄保護邊界 (Protected)   │ Agent 嘗試讀取 ~/.gemini/antigravity-cli/settings.json      │
│ 2. 敏感破壞性操作 (User Veto) │ 執行高危指令 (rm -rf) 觸發彈窗，使用者點選 Reject          │
│ 3. 背景任務中途終止 (Aborted) │ 使用者或主控透過 manage_task(Action="kill") 取消子代理      │
└───────────────────────────────┴─────────────────────────────────────────────────────────────┘
```

### 2. 計費與 Token 特性
* **0 GPU Token 消耗**：攔截發生在本地 Harness 邊界層，未出門、未發送雲端 API 請求；
* **遙測標籤**：`🛡️ BLOCKED`，Track 1 明確宣告 `Action Status: Blocked by System Permission Guard (0 GPU Tokens Billed) 🛡️`。

---

## 五、本地 Tool 步驟的上下游穿透關聯

針對本地 Tool 步驟（如 Step #3074 `edit_file`），Observer 實裝了上下游穿透解析：
```text
TRACK 1: LOCAL EXECUTION STEP (OFFLINE OPERATION)
  • Origin / Role        : SUBAGENT (RUN_COMMAND | Step #3074 | Status: DONE | 08:06:00)
  • Model & Payload      : Gemini 3.7 Flash | 1,420 Tokens (Tool Result Data)
  • Billing Attribution  : Local Offline Subprocess (0 GPU Tokens) ➔ Billed in Cloud Turn #3076 ☁️
```

---

## 六、具體演繹實例 (Concrete Walkthrough)

### 1. Minimal Concrete Input
主控 Agent 接收重構任務後，發起子代理檢索代碼：
* Step #100: Main Planner (`gemini-3.7-flash`, Total 180k, Cached 160k) $\to$ `invoke_subagent`
* Step #101: Subagent Worker 初始化獨立 Context (Total 25k, Cached 0)
* Step #102: Subagent 執行 `run_command: git status`（本地 Tool）
* Step #103: Subagent 嘗試讀取 `settings.json`（遭沙盒攔截）
* Step #104: Subagent Worker (`gemini-3.7-flash`, Total 32k, Cached 25k) $\to$ 回報搜尋結果

### 2. Step-by-Step 狀態機與 Token 演繹

```text
Step #100 (Main Cloud Turn):
  • AgentRole: MAIN | Scope: CLOUD | Model: Gemini 3.7 Flash
  • Tokens: Total 180k, Cached 160k (88.9% HIT), New 20k
  • 行為: 產出 invoke_subagent 指令

Step #101 (Subagent Cloud Turn #1):
  • AgentRole: SUBAGENT | Scope: CLOUD | Model: Gemini 3.7 Flash
  • Tokens: Total 25k, Cached 0 (0% WRITE / Cold Start), New 25k
  • 行為: 載入獨立 Prompt，產出 run_command 指令

Step #102 (Subagent Local Tool):
  • AgentRole: SUBAGENT | Scope: LOCAL | Tool: run_command
  • Tokens: 0 GPU Tokens (Offline CPU) | Status: DONE
  • 穿透標記: Staged Output 1.2k Tokens ➔ Billed in Step #104

Step #103 (Permission Blocked):
  • AgentRole: INTERNAL | Scope: LOCAL | Tool: view_file (settings.json)
  • Tokens: 0 GPU Tokens | Status: BLOCKED (status=7)
  • 標記: 🛡️ BLOCKED (Permission Denied)

Step #104 (Subagent Cloud Turn #2):
  • AgentRole: SUBAGENT | Scope: CLOUD | Model: Gemini 3.7 Flash
  • Tokens: Total 32k, Cached 25k (78.1% PARTIAL), New 7k (含 Step #102 的 1.2k Tool Output)
  • 行為: 回報結論給 Main Planner
```

---

## 七、雙向鏈接與相關卡片

* **理論基石**：[[01_Transformer_Prefill_vs_Decode]]、[[03_Prompt_Caching_Lifecycle]]
* **架構實現**：[[01_Context_5_Dimensions]]、[[03_Agent_Storage_and_State_Machine]]、[[10_Dashboard_Aggregate_Metrics_and_Multi_Model_Pricing]]
* **實戰排查**：[[09_Harness_Internal_Plumbing_Filtering_and_SQLite_Gap_Recovery]]
