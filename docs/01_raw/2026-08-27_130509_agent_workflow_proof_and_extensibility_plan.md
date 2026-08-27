# 🔬 AI Agent 工作流通用性證明與領域模型擴展性論證 (Universal Workflow Proof & Extensibility)

> **版本**：v4.0 (包含物理拓撲完備性證明、標準協議對照表、領域模型設計哲學與未來 Agent 擴展機制)  
> **關聯文件**：`docs/01_raw/2026-08-27_130509_agent_workflow_proof_and_extensibility_plan.md`

---

## 🏛️ 一、 如何證明「所有 AI Agent 都遵循這個工作流且只有這些狀態」？

這不是憑空猜想，而是由 **「分散式系統物理邊界」** 與 **「現代 LLM 通訊協議」** 兩大鐵律所決定的數學與工程必然性。

---

### 1. 物理拓撲完備性證明 (Physical & Distributed Proof)

任何基於 LLM 的 AI Agent，在物理世界上**只有 2 個計算實體與 2 個外部主體**，不存在第 5 種物理存在：

```text
               ┌──────────────────────────────────────────────┐
               │           👤 人類使用者 (User Actor)          │
               └──────────────────────┬───────────────────────┘
                                      │ 1. 使用者意圖 (USER)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ 💻 本地執行環境 (Local Host Machine - Client)                                │
│                                                                             │
│   ⚙️ Agent 框架與開發者設定 (SYSTEM) ➔ 組合 Context 與工具 Schema           │
│                                                                             │
│   💻 本機實體執行 (LOCAL) ➔ Shell, Diff, File I/O, LSP, Subprocess          │
└─────────────────────────────────────┬───────────────────────────────────────┘
                                      │ 2. HTTP / gRPC 網路請求 (Payload)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ ☁️ 雲端 LLM 推理伺服器 (Remote GPU Cluster - Server)                        │
│                                                                             │
│   ☁️ 雲端生成與決策 (CLOUD) ➔ KV Cache 檢索、Next-Token 生成、Tool 決策    │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 嚴格完備性推論 (Exhaustive Proof)：
1. **使用者發話**：來自外部人類 $\implies$ 物理本質必然是 **`USER`**；
2. **框架開局**：來自本機設定檔、系統提示詞、工具定義 $\implies$ 物理本質必然是 **`SYSTEM`**；
3. **雲端推論**：來自遠端 GPU 集群的 HTTP/gRPC 生成結果 $\implies$ 物理本質必然是 **`CLOUD`**；
4. **本機執行**：來自本機作業系統的 Process、檔案讀寫、終端指令 $\implies$ 物理本質必然是 **`LOCAL`**。

$$\text{All Agent Events} \subseteq \{ \text{USER}, \text{SYSTEM}, \text{CLOUD}, \text{LOCAL} \}$$

**物理上不可能存在第 5 種狀態**，因為計算要么發生在雲端 GPU，要么發生在本機 CPU，要么來自人，要么來自系統開局。

---

### 2. 業界三大主流 Agent 協議 1:1 精確對照表 (Protocol Proof)

所有主流 AI Agent（Google Antigravity、Anthropic Claude Code、OpenCode、OpenAI Agents）底層全部基於標準的 **Chat Completion Message Specification**：

| 通用 Scope (`agent-observer`) | Google Antigravity | Anthropic Claude Code | OpenAI / OpenCode | 物理發生地點 | 是否消耗 GPU 計費？ |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`USER`** | `USER_INPUT` | `type: "user"` | `role: "user"` | 本機 ➔ 發往雲端 | 是（作為輸入 Token 計費） |
| **`SYSTEM`** | `CONVERSATION_HISTORY` / `CHECKPOINT` | `type: "system"` | `role: "system"` | 本機設定檔 | 是（前綴快取 / KV Cache） |
| **`CLOUD`** | `PLANNER_RESPONSE` (ToolCall / Text) | `type: "assistant"` (`tool_use`) | `role: "assistant"` (`tool_calls`) | 雲端 GPU 集群 | **是（產出 Output Tokens + 決定工具）** |
| **`LOCAL`** | `RUN_COMMAND` / `VIEW_FILE` / `CODE_ACTION` | `type: "user"` (`tool_result`) | `role: "tool"` (`tool_output`) | 本地 Mac/Linux Process | **否（本地執行 0 元，下一輪才送出）** |

---

## 🧩 二、 「這些 Event 是我們自己維護的，其他 Agent 要多狀態也能擴展嗎？」

### 💡 答案是：**完全正確！這正是 `agent-observer` 作為「觀測中樞」的核心職責！**

在軟體架構中，這稱為 **「領域驅動設計 (DDD)」** 與 **「開放封閉原則 (Open-Closed Principle)」**：

1. **我們自己維護統一的領域模型 (`UnifiedAgentEvent`)**：
   * `agent-observer` 不受任何單一商業公司的私有日誌格式綁架；
   * 我們定義了最乾淨、最具表達力的核心領域模型。
2. **兩層結構設計：頂層穩定 (4 Scopes) + 底層無限擴展 (Sub-Types & Metadata)**：
   * **頂層 Scope（4 種，永久穩定）**：`USER`, `SYSTEM`, `CLOUD`, `LOCAL`（基於物理與協議邊界，永遠不會過時）；
   * **底層 StepType（開放擴展）**：各個 Agent 可以有自己特有的細部動作！

```go
// 1. 頂層物理邊界 (4 大骨架，絕對穩定)
type StepScope string
const (
    ScopeUserInteraction ScopeScope = "USER"   // 👤 使用者意圖
    ScopeCloudInference  ScopeScope = "CLOUD"  // ☁️ 雲端 LLM 算力
    ScopeLocalExecution  ScopeScope = "LOCAL"  // 💻 本機離線執行
    ScopeSystemBootstrap ScopeScope = "SYSTEM" // ⚙️ 系統約束注入
)

// 2. 底層細部動作 (可隨不同 Agent 隨意擴展！)
type StepType string
const (
    // 通用動作
    StepTypeUserInput      StepType = "USER_INPUT"
    StepTypeModelResponse  StepType = "MODEL_RESPONSE"
    StepTypeToolCall       StepType = "TOOL_CALL"
    
    // Antigravity 專屬或通用 Tool 動作
    StepTypeRunCommand     StepType = "RUN_COMMAND"
    StepTypeViewFile       StepType = "VIEW_FILE"
    StepTypeCodeAction     StepType = "CODE_ACTION"
    
    // 未來擴展：Claude Code 專屬
    StepTypeThinkingBlock  StepType = "CLAUDE_THINKING" // Claude 3.7 擴展思維塊
    StepTypeSubagentHandoff StepType = "SWARM_HANDOFF"  // Multi-Agent 轉移
    StepTypeLSPDiagnostic  StepType = "LSP_DIAGNOSTIC" // IDE 語法診斷
)
```

---

## 🔌 三、 具體案例：未來新增一個 Agent 是如何擴展的？

假設未來我們新增了 **`Claude Code`** 或 **`OpenCode`**：

```text
[Claude Code 原生日誌 (.claude/transcript.jsonl)]
   │
   ▼ 經過 internal/adapters/claudecode/watcher.go
{
    "type": "assistant",
    "content": [
        {"type": "thinking", "thinking": "Let me search..."},
        {"type": "tool_use", "name": "Bash", "input": {"command": "npm test"}}
    ]
}
   │
   ▼ 轉換為 agent-observer 標準領域事件 (Universal Event)
UnifiedAgentEvent{
    Scope:     ScopeCloudInference,       // ☁️ 頂層歸類為雲端推論
    Type:      StepTypeToolCall,          // 動作歸類為工具調用
    Summary:   "🛠️ Tool Call: Bash [npm test]",
    Thinking:  "Let me search...",
    OfficialModel: "claude-3-7-sonnet",
}
```

* **狀態機 (`StepLinkageTracker`)**：直接接收這個 `Universal Event`，自動判定 Parent，完全不用為 Claude 寫一套新狀態機！
* **TUI 畫面 (`views.go`)**：直接根據 `Scope` 渲染藍色雲端徽章與模型資訊，完全不用為 Claude 改動 UI！

---

## 🎯 四、 總結回覆

1. **為何能證明通用？**
   * 因為所有 AI Agent 都是運行在「本機 Client」與「雲端 GPU」之間的分散式系統，事件來源**物理上必然嚴格收斂在 `USER`、`SYSTEM`、`CLOUD`、`LOCAL` 4 種範疇**，且所有主流 Agent 都遵循相同的 Message Protocol。
2. **事件是我們自己維護的嗎？其他 Agent 要多狀態怎麼辦？**
   * **是的！** `UnifiedAgentEvent` 是我們的主權領域模型。頂層 4 大 Scope 提供穩定的物理與計費心智模型，而底層的 `StepType` 與 `Metadata` 則是 100% 開放擴展的，任何新 Agent 的特殊狀態（如思維鏈塊、IDE 診斷、Swarm 交接）都可以隨時無痛追加！
