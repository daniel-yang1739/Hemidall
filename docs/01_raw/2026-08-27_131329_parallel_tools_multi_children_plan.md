# 🌲 一對多父子階層結構分析：平行工具調用場景 (1-to-Many Parent-Child Tree Specification)

> **版本**：v7.0 (涵蓋平行多工具呼叫 Parallel Tool Calls、樹狀視覺連接符號與多子節點鍵盤導航規則)  
> **關聯文件**：`docs/01_raw/2026-08-27_131329_parallel_tools_multi_children_plan.md`

---

## 🎯 一、 會有一對多的情境嗎？(Are there 1-to-Many Scenarios?)

### 💡 答案是：**會的！最典型且極其常見的場景就是「平行多工具呼叫 (Parallel Tool Calling)」！**

在現代 LLM Agent（包括 Google Antigravity、Anthropic Claude Code、OpenAI Assistants）中，模型可以在 **單一輪次決策** 裡同時下達多個工具指令：

---

### 🌟 實戰案例：同時讀取 3 個檔案 (Multi-File View)

1. **第 1 層 (父層 ☁️ 雲端決策)**：
   * `Step 100 [TOOL_CALL]`：Gemini 決定同時查看 `model.go`、`views.go` 與 `types.go`；
2. **第 2 層 (子層 💻 本地執行，共有 3 個實體步驟)**：
   * `Step 101 [VIEW_FILE]`：本地讀取 `model.go` (Child 1)
   * `Step 102 [VIEW_FILE]`：本地讀取 `views.go` (Child 2)
   * `Step 103 [VIEW_FILE]`：本地讀取 `types.go` (Child 3)
3. **成果打包 (下一輪 ☁️ 雲端接收)**：
   * `Step 104 [PLANNER_RESPONSE]`：Agent 將這 3 個檔案的內容一次性打包送入 Google API，模型讀完 3 個檔案後開始寫 code！

---

## 🖥️ 二、 左欄樹狀清單渲染標準 (Tree Connector Symbols)

當一個 `TOOL_CALL` 衍生出多個本地子步驟時，採用標準樹狀分支符號（`├──` 與 `└──`）：

```text
╭────────────────────────────────────────╮
│ STEPS (3585) < [T:All] [C:All]         │
│   ...                                  │
│   [104|MODEL] 11:11:25 ☁️ [HIT 98%]     │ ➔ 雲端接收 3 個檔案輸出並決策
│     I will now modify model.go...      │
│   [100|TOOL ] 11:11:20 ☁️ [HIT 95%]     │ ➔ 雲端同時下達 3 個 view_file 指令
│     view_file x 3 (model, views, types)│
│   ├── [101|FILE ] 11:11:21 💻 (Local)  │ ➔ 子步驟 1 (讀取 model.go)
│   │     package ui ...                 │
│   ├── [102|FILE ] 11:11:21 💻 (Local)  │ ➔ 子步驟 2 (讀取 views.go)
│   │     package ui ...                 │
│   └── [103|FILE ] 11:11:21 💻 (Local)  │ ➔ 子步驟 3 (讀取 types.go)
│         package core ...               │
│   [099|USER ] 11:11:15 👤              │ ➔ 使用者 Prompt
│     請幫我檢查這三個檔案...            │
╰────────────────────────────────────────╯
```

---

## ⌨️ 三、 一對多情境下的 `p` / `n` 導航規則

1. **從任意子步驟按 `p` (Parent)**：
   * 無論你停在 `Step 101`、`Step 102` 還是 `Step 103`，按 **`p`** 都會**精準跳回父層 `Step 100`**！
2. **從父層步驟按 `n` (Next / Child)**：
   * 在 `Step 100` 按 **`n`**，優先跳至**第一個子步驟 `Step 101`**；
   * 在子步驟按 **`j` / `k`** 即可在 `101 ➔ 102 ➔ 103` 之間上下選取；
3. **從最後一個子步驟按 `n`**：
   * 在 `Step 103` 按 **`n`**，跳至打包消耗它們的下一輪雲端步驟 `Step 104`！
