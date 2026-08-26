# 🔬 ~/.gemini 本地 SQLite 儲存架構與 gen_metadata Protobuf 二進制結構全逆向解析

> **建立時間**：2026-08-26 11:30:00
> **更新時間**：2026-08-26 14:00:00  
> **來源目錄**：`~/.gemini/antigravity-cli/conversations/<session-id>.db`  
> **研究主題**：Antigravity CLI 本地資料庫儲存模式、Google 官方 API 回傳之 Protobuf 二進制結構、欄位語義與萃取技術。

---

## 📌 一、發現背景與核心突破

在過去的設計中，我們認為本地日誌 `transcript_full.jsonl` 只包含人類與 Agent 的對話行為文字（沒有官方計費與 Token 指標）。然而在本次深度探索中，我們逆向了 `~/.gemini/antigravity-cli/conversations/<session-id>.db`，發現了一個極具價值的寶藏：

**Google 雲端 API 每次回傳的完整 Protobuf Generation Metadata，被 100% 原汁原味地儲存在 SQLite 資料庫的 `gen_metadata` 資料表中！**

這意味著我們不需要架設任何中間人 HTTP Proxy 攔截網路，只需透過本地 SQLite 讀取，就能以 **100% 零侵入、零延遲** 的方式，直接拿到 Google 官方真實的 Token 總數、快取命中數與後端模型代號！

---

## 🏛️ 二、~/.gemini/antigravity-cli 檔案拓撲與 SQLite 7 大核心表

### 1. 目錄拓撲
```text
~/.gemini/antigravity-cli/
├── brain/
│   └── <session-id>/
│       └── .system_generated/logs/
│           ├── transcript.jsonl          # 輕量版對話軌跡
│           └── transcript_full.jsonl     # 全量未截斷對話行為日誌
└── conversations/
    ├── <session-id>.db                   # SQLite 會話主資料庫
    ├── <session-id>.db-shm               # SQLite 共享記憶體
    └── <session-id>.db-wal               # SQLite 預寫日誌 (Write-Ahead Logging)
```

### 2. SQLite `<session-id>.db` 資料表架構
經 `sqlite3 .schema` 解析，核心包含 7 張表：

| 資料表名稱 (Table) | 核心欄位 | 儲存內容與工程職責 |
| :--- | :--- | :--- |
| **`gen_metadata`** | `idx` (INTEGER), `data` (BLOB), `size` (INTEGER) | **【最核心】** 儲存每次模型生成回傳的原始 Protobuf 二進制二維資料。 |
| **`steps`** | `idx`, `step_type`, `status`, `metadata`, `step_payload` | 儲存每一步驟的執行狀態、權限、工具呼叫與渲染資訊。 |
| **`trajectory_meta`** | `trajectory_id`, `cascade_id`, `trajectory_type`, `source` | 會話軌跡的全局 Metadata 與關聯 ID。 |
| **`executor_metadata`**| `idx`, `data` (BLOB) | 執行器（如 subagents、執行環境）的狀態資料。 |
| **`parent_references`**| `idx`, `data` (BLOB) | 父子對話、Subagent 派生關聯參照。 |
| **`trajectory_metadata_blob`** | `id`, `data` (BLOB) | 全局軌跡配置二進制資料。 |
| **`battle_mode_infos`** | `idx`, `data` (BLOB) | 模型對抗/評估模式相關資訊。 |

---

## 🧬 三、gen_metadata 表的 Protobuf Wire Format 逐層欄位解析

`gen_metadata.data` 欄位儲存的是 Google 標準 Protocol Buffers (proto3) 序列化位元組流。以下為我們遞迴解碼出的完整欄位對照樹：

```text
gen_metadata.data (Root Protobuf Message)
├── Field 1 (Sub-Message: Telemetry & Model Metadata)
│   ├── Field 3 (Varint): 內部呼叫序號 (如 1298)
│   ├── Field 4 (Sub-Message: Session & Latency Info)
│   │   ├── Field 1 (Varint): 1298
│   │   ├── Field 2 (Varint): 耗時 / 延遲 (Latency in ms, 如 5733ms)
│   │   ├── Field 3 (Varint): 思考鏈 Token 數 (Thinking Tokens, 如 951)
│   │   ├── Field 5 (Varint): 提示詞 Tokens (Prompt Tokens 估計)
│   │   ├── Field 7 (String): 後端 Bot 實例 ID (如 bot-beb30e67-...)
│   │   └── Field 8 (Sub-Message): sessionID
│   │
│   ├── Field 9 (Sub-Message: Token Counting & Cache Telemetry)
│   │   └── Field 10 (Sub-Message: 核心上下文與快取計量)
│   │       ├── Field 1 (Varint): ⭐ 【Total Prompt Tokens】(官方總輸入 Token 數，如 175,683)
│   │       ├── Field 4 (Varint): ⭐ 【Context Window Limit】(窗口上限，固定為 256,000)
│   │       └── Field 3 (Sub-Message: ⭐ 【Context Caching Details】)
│   │           └── Field 1 (Sub-Message)
│   │               ├── Field 2 (Varint): 快取類型 (Bucket Type, 固定為 7)
│   │               ├── Field 4 (Varint): ⭐ 【Cached Content Tokens】(物理顯存真實命中數，如 154,888)
│   │               └── Field 5 (Sub-Message: 快取 TTL / 索引資訊)
│   │
│   ├── Field 16 (Repeated Sub-Messages): 各系統提示詞區塊 (<identity>, <user_rules>, <skills> 等)
│   └── Field 19 (String): ⭐ 【Model Name】(官方後端真實模型代號，如 "gemini-3.7-flash-high")
│
└── Field 20 (Repeated Key-Value Sub-Messages: 請求軌跡映射)
    ├── "request_id": UUID (如 "5eaed723-5d73-4605-a9b3-e0f99431df2f-554")
    ├── "last_step_index": 關聯的對話步驟 (如 "1236")
    ├── "used_claude": "false"
    ├── "used_non_gemini_model": "false"
    └── "model_enum": "MODEL_PLACEHOLDER_M298"
```

---

## 🛠️ 四、純 Go 零依賴 Protobuf 解碼演算法

因為 Antigravity 沒有公開 `.proto` 描述檔，我們直接實作了 **基於 Wire Format 的純 Go 動態解析器**：

* **Varint 讀取器**：解析 Tag/WireType 與 64-bit 整數（Shift 7 位元）；
* **長度界定遞迴**：當 `WireType == 2` 時，自動嘗試遞迴解析為子 Message；若無有效 Tag 則作為 UTF-8 字串或 Raw Bytes 處理；
* **正規表示式輔助**：針對固定字串標籤（如 `last_step_index\x12\x04(\d+)` 與 `(gemini-[a-zA-Z0-9\.\-]+)`）進行極速正則匹配。

```go
// 核心解碼呼叫範例
meta, err := ParseGeminiGenMetadata(idx, rawBlob)
// meta.TotalTokens  -> 175683
// meta.CachedTokens -> 154888
// meta.HitRate      -> 88.16%
// meta.ModelName    -> "gemini-3.7-flash-high"
// meta.ContextLimit -> 256000
```
