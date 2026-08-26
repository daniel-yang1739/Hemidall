# 🗄️ Google Gemini SQLite 本地資料庫終極實戰查詢手冊 (Gemini SQLite Query Handbook)

> **建立時間**：2026-08-26 11:30:00
> **更新時間**：2026-08-26 14:00:00  
> **資料庫路徑**：`~/.gemini/antigravity-cli/conversations/<session-id>.db`  
> **適用目標**：從零到精通隨心所欲查詢、分析、逆向與監控 Google Gemini Agent 本地儲存庫。

---

## 🧭 一、核心架構與連線方式

Antigravity CLI 將每一個會話的所有狀態、歷史訊息、工具執行結果與 Google API 官方遙測資料，全部持久化在 SQLite 資料庫中。

### ⚠️ 最高安全鐵律：永遠使用「非阻塞唯讀 (Read-Only WAL)」模式
因為 Antigravity CLI 會在背景即時寫入資料庫，如果你直接用寫入模式打開，可能會導致資料庫被鎖定 (`database is locked`) 甚至造成 Agent 當機！

---

### 💻 3 大常用連線查詢方式

#### 方式 1：終端機原生 `sqlite3` CLI (互動式查詢與表格美化)
在終端機中執行（自動開啟唯讀模式與表格美化）：

```bash
# 取得當前 session_id 的 db 路徑
DB_PATH="$HOME/.gemini/antigravity-cli/conversations/$(ls -t $HOME/.gemini/antigravity-cli/conversations/*.db | head -n 1 | xargs -n 1 basename)"

# 以唯讀模式進入 sqlite3
sqlite3 -readonly "$HOME/.gemini/antigravity-cli/conversations/$DB_PATH"
```

進入 sqlite3 後，輸入以下指令開啟「美化表格輸出」：
```sql
.mode table
.headers on
```

---

#### 方式 2：Python 3 腳本查詢（適合資料萃取與自動化分析）
```python
import sqlite3

db_path = "/Users/daniel_y_yang/.gemini/antigravity-cli/conversations/aa726359-08e2-4687-a15c-073a2f4a705b.db"

# 以 URI 模式開啟唯讀連線
conn = sqlite3.connect(f"file:{db_path}?mode=ro", uri=True)
cursor = conn.cursor()

cursor.execute("SELECT idx, size FROM gen_metadata ORDER BY idx DESC LIMIT 5")
for row in cursor.fetchall():
    print(row)

conn.close()
```

---

#### 方式 3：Go 語言零 CGO 唯讀連線
```go
package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

func main() {
	dsn := "file:/path/to/session.db?mode=ro&_journal=WAL"
	db, err := sql.Open("sqlite", dsn)
	if err != nil { panic(err) }
	defer db.Close()

	var count int
	db.QueryRow("SELECT count(*) FROM steps").Scan(&count)
	fmt.Printf("Total Steps: %d\n", count)
}
```

---

## 🏛️ 二、SQLite 7 大資料表結構與欄位辭典

透過 `.tables` 指令可看到資料庫中共有 7 張表：

```text
├── 👑 gen_metadata             # 模型生成遙測數據 (Google API 官方回傳的 Protobuf BLOB)
├── 📜 steps                    # 每一個執行步驟 (使用者輸入、模型思考、工具呼叫與輸出)
├── 🌐 trajectory_meta          # 會話全局元資料 (Session ID, Cascade ID)
├── ⚙️ executor_metadata        # 執行器與 Subagent 環境狀態
├── 🔗 parent_references        # 父子會話與關聯鏈結
├── 📦 trajectory_metadata_blob # 全局配置二進制資料
└── ⚔️ battle_mode_infos        # 模型對抗/評估模式資訊
```

---

### 1. 👑 `gen_metadata` 表（模型生成與 Token 遙測表）
每次模型進行一次 API 推理，就會在此表寫入一筆記錄：

| 欄位名稱 | 型態 | 欄位說明與用途 |
| :--- | :--- | :--- |
| **`idx`** | `INTEGER` | Generation 序號（自增主鍵，如 0, 1, 2, ..., 687）。 |
| **`data`** | `BLOB` | **【最核心】** Google Gemini 回傳的完整 Protobuf 二進制二維流（含 Token 數、快取命中、模型名）。 |
| **`size`** | `INTEGER` | `data` 二進制位元組大小（通常在 1,000 ~ 1,200 bytes 之間）。 |

---

### 2. 📜 `steps` 表（所有步驟行為軌跡表）
記錄 Agent 執行的每一步驟：

| 欄位名稱 | 型態 | 欄位說明與用途 |
| :--- | :--- | :--- |
| **`idx`** | `INTEGER` | Step 步驟序號（0, 1, 2, ..., 1400）。 |
| **`step_type`** | `INTEGER` | 步驟類型代碼（`15`: 模型回答/規劃, `132`: 工具呼叫, `14`: 使用者發問, `5`: 任務管理）。 |
| **`status`** | `INTEGER` | 執行狀態（`3`: 完成 DONE, `2`: 執行中 RUNNING, `6`/`7`: 錯誤/取消）。 |
| **`step_payload`** | `BLOB` | 該步驟的詳細 Payload（包含工具名稱、呼叫參數 JSON、終端輸出文字、或 `<CONTEXT_SUMMARY>`）。 |
| **`metadata`** | `BLOB` | 附加元資料（如延遲毫秒數、呼叫來源等）。 |
| **`error_details`**| `BLOB` | 若執行失敗，記錄具體的錯誤堆疊與原因。 |

---

### 3. 🌐 `trajectory_meta` 表（會話全局元資料）
| 欄位名稱 | 型態 | 說明 |
| :--- | :--- | :--- |
| **`trajectory_id`** | `TEXT` | 該會話的唯一 Session UUID。 |
| **`cascade_id`** | `TEXT` | 級聯追蹤 ID。 |

---

## 📊 三、實戰 SQL 查詢錦囊 (Query CookBook & Cheat Sheet)

以下是日常分析與除錯最常用的 SQL 查詢指令，可直接複製貼入 `sqlite3` 或 Python 中執行：

---

### 🔍 查詢 1：取得資料庫總體概況（總步驟數、總 Generation 數）
```sql
SELECT 
    (SELECT count(*) FROM steps) AS total_steps,
    (SELECT count(*) FROM gen_metadata) AS total_generations,
    (SELECT max(idx) FROM steps) AS latest_step_idx,
    (SELECT max(idx) FROM gen_metadata) AS latest_gen_idx;
```

---

### 🔍 查詢 2：統計各種類型步驟的出現次數 (Step Type Breakdown)
```sql
SELECT 
    step_type,
    CASE step_type
        WHEN 15 THEN 'Model Response / Planner'
        WHEN 132 THEN 'Tool Call / Execution'
        WHEN 14 THEN 'User Input'
        WHEN 5 THEN 'Task / Subagent'
        ELSE 'Other (' || step_type || ')'
    END AS type_description,
    count(*) AS count,
    ROUND(count(*) * 100.0 / (SELECT count(*) FROM steps), 1) AS percentage
FROM steps
GROUP BY step_type
ORDER BY count DESC;
```

---

### 🔍 查詢 3：搜尋特定工具呼叫（例如搜尋所有 `write_to_file` 或 `run_command`）
```sql
SELECT 
    idx AS step_idx,
    status,
    length(step_payload) AS payload_size
FROM steps 
WHERE step_type = 132 
  AND step_payload LIKE '%run_command%'
ORDER BY idx DESC 
LIMIT 10;
```

---

### 🔍 查詢 4：尋找觸發歷史截斷與 `<CONTEXT_SUMMARY>` 的步驟
```sql
SELECT 
    idx AS step_idx, 
    step_type, 
    status, 
    length(step_payload) AS payload_bytes
FROM steps 
WHERE step_payload LIKE '%<CONTEXT_SUMMARY>%'
ORDER BY idx ASC;
```

---

### 🔍 查詢 5：搜尋特定關鍵字或錯誤日誌（模糊搜尋）
```sql
-- 搜尋含有 "error" 或 "FAIL" 的步驟
SELECT 
    idx AS step_idx,
    step_type,
    status
FROM steps 
WHERE error_details IS NOT NULL 
   OR step_payload LIKE '%error%' 
   OR step_payload LIKE '%FAIL%'
ORDER BY idx DESC 
LIMIT 10;
```

---

### 🔍 查詢 6：依 Generation 序號檢視 Protobuf Payload 大小變化
```sql
SELECT 
    idx AS gen_idx,
    size AS protobuf_bytes,
    ROUND(size / 1024.0, 2) AS size_kb
FROM gen_metadata 
ORDER BY idx DESC 
LIMIT 15;
```

---

## 🧬 四、進階：用 Python 深度解碼 `gen_metadata` 與 `steps`

在 SQLite 中，`data` 和 `step_payload` 是二進制 BLOB。我們可以用 Python 搭配正則表達式或 Protobuf 解碼器，直接將裡面的明文資訊萃取出來！

### 🐍 實用 Python 萬能查詢腳本 (`query_gemini.py`)

建立以下腳本即可快速查看即時數據：

```python
import sqlite3
import re
import os
import sys

db_dir = os.path.expanduser("~/.gemini/antigravity-cli/conversations")
# 自動抓取最新修改的 .db 檔案
dbs = sorted([os.path.join(db_dir, f) for f in os.listdir(db_dir) if f.endswith(".db")], key=os.path.getmtime, reverse=True)

if not dbs:
    print("❌ No SQLite database found!")
    sys.exit(1)

latest_db = dbs[0]
print(f"📂 Connected to SQLite: {os.path.basename(latest_db)}\n")

conn = sqlite3.connect(f"file:{latest_db}?mode=ro", uri=True)
c = conn.cursor()

# 1. 統計概況
c.execute("SELECT count(*) FROM steps")
total_steps = c.fetchone()[0]
c.execute("SELECT count(*) FROM gen_metadata")
total_gens = c.fetchone()[0]
print(f"📊 Total Steps: {total_steps} | Total Generations: {total_gens}\n")

# 2. 萃取最新 5 筆 Generation 的真實模型代號與關聯步驟
print("👑 Latest Generations Telemetry:")
c.execute("SELECT idx, size, data FROM gen_metadata ORDER BY idx DESC LIMIT 5")
for idx, size, data in c.fetchall():
    # 正則匹配模型代號
    model_match = re.search(b"(gemini-[a-zA-Z0-9\.\-]+)", data)
    model_name = model_match.group(1).decode("utf-8") if model_match else "unknown"
    
    # 正則匹配 last_step_index
    step_match = re.search(b"last_step_index\x12\x04(\d+)", data)
    last_step = step_match.group(1).decode("utf-8") if step_match else "N/A"
    
    print(f"  • Gen #{idx:03d} | Size: {size:4d} bytes | Model: {model_name:<22} | Step: {last_step}")

# 3. 檢查是否有 Context 截斷摘要
c.execute("SELECT idx, length(step_payload) FROM steps WHERE step_payload LIKE '%<CONTEXT_SUMMARY>%'")
summary_steps = c.fetchall()
if summary_steps:
    print(f"\n✂️ Context Summary Detected at Steps: {[s[0] for s in summary_steps]}")

conn.close()
```

---

## 🛠️ 五、小結與查詢心法

1. **查進度與步驟** $\to$ 查 **`steps`** 表（看 `idx`, `step_type`, `step_payload`）；
2. **查 Token 與官方計費** $\to$ 查 **`gen_metadata`** 表（看 `idx`, `size`, `data`）；
3. **查當前 Session ID** $\to$ 查 **`trajectory_meta`** 表；
4. **永遠記得加 `?mode=ro`**，享受安全無鎖定的極速資料庫探勘體驗！
