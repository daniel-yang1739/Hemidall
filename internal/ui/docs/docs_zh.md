## Heimdall 目前到底知道什麼

Heimdall 讀的是 Antigravity 留在本機的檔案。它沒有攔到真正送去雲端的 HTTP request，所以不能把畫面上的資料說成官方帳單、實際價格，或保證的 KV cache 行為。

* **Transcript**：`transcript_full.jsonl` 是事件時間線。裡面有使用者輸入、模型回覆、tool call、本機 tool 輸出，以及 checkpoint 類型的內容。
* **Generation metadata**：`conversations/<session>.db` 的 `gen_metadata` table 有 protobuf blob。Heimdall 會解出目前已觀察到的 `last_step_index`、context total、context limit，以及部分 cache 欄位。
* **Persisted context snapshot**：Heimdall 由大於安全門檻、從最新 `idx` 往回找到，且能解出 wire path `1.1` 的 `gen_metadata` blob。它可能包含 system prompt、重複 context record 與 tool 定義；這是一份「當時被保存的 context 狀態」，不是官方 request body。

## 一筆 generation metadata 對應哪個 step

`last_step_index` 的意思是這次 generation 開始前，最後吃到哪一個 transcript step。真正產生的 model event 是下一個 step。

`input boundary step N` → `generated transcript step N + 1`

這是目前 Antigravity 資料中觀察到的固定關係，不是拿前後 step 猜一個最像的。如果其中一邊找不到，Heimdall 會顯示 unavailable。

## Token 數字分成兩條完全不同的資料線

### Persisted context observation

如果某個 model event 成功對應到 generation metadata，Heimdall 可以顯示：

* **Observed context tokens**：那筆保存資料裡解出的 total。
* **Observed context limit**：只有 protobuf 裡真的有這個欄位時才顯示。
* **Cache fields**：只有 total 和 cache 兩個欄位都有被解出、而且彼此合理時才計算。

這些是根據本機 wire format 做出的觀察，不是官方 API response、帳單、價格或快取保證。Heimdall 不會再根據前一輪、閒置時間或 model 名稱去補一個 cache hit rate。

### Local transcript estimate

`cl100k_base` 只是在數某一筆 transcript text 大約有多少 token。Heimdall 用它顯示「這一個 event 的文字量」和「整個 session 目前累積讀過多少 transcript text」。

後者**不是**下一個 request 的 context。舊資料可能已被 compact、被排除，或在 snapshot 裡用不同形式保存，所以累積值很大是正常的，不能拿來當 context window。

## 五維表是在估什麼

Track 2 的五維表是**目前選中 playback step 的可見 context evidence**。history 更新時，Heimdall 會先替每一個 event 建立本機 transcript estimate，所以在 Dashboard 移動游標不會 query SQLite，也不會重新掃完整段 transcript。若選到的是 cloud generation event，estimate 使用該 generation 發生前的 evidence；其他 event 則使用到該 event 為止的 evidence。它不是 Heimdall 重組出的 provider request：

* system instruction；
* tool definitions；
* staged tool buffers；
* history evidence；以及
* transcript 裡觀察到的 latest inbound prompt。

每一格都是本機 `cl100k_base` 的估算。如果選中的 step 剛好等於最新 persisted snapshot 對應的 generated step，Track 2 才會用 snapshot 的 system、tools、history 值；該 step 的 buffers 和 inbound 仍使用 transcript 觀察。若沒有這種精確對應，五個維度仍會顯示，但都會清楚標成 `transcript`。這五格讓人看得懂「可見資料來自哪一條資料線」，但它不是 Heimdall 重組出的 provider request，也不能拿數字直接和 Track 1 的 observed context total 比較。

## Dashboard 怎麼看

* **Track 1 — Persisted Cloud Usage Observation**：只看目前選中的 model event 有沒有對應到保存的 generation metadata。
* **Track 2 — Playback Context Evidence**：會跟著選中的 event 改變。它使用該 step 已快取的本機 transcript estimate；只有 snapshot 剛好對應到這一個 generated step 時，才使用 snapshot 的維度。對應規則是有 `last_step_index` 時的 `last_step_index + 1`。其他 step 一律清楚標為 transcript estimate。移動 playback 時，Track 1 和 Track 2 都會變，但不會觸發 SQLite I/O。
* **Session aggregates**：把每一輪觀察到的 context total 加總，適合看趨勢；它不是 API 帳單，也不是去重後的 token 數。

## Context 頁面怎麼看

Context 頁不是只選一個來源：它先用 transcript 建立「這場對話發生過什麼」的底稿；若有 snapshot，再覆寫其中確實由 snapshot 保存的 system、rules、skills、tools 和 persisted record。runtime metadata 仍由 Heimdall 本機程序取得。Raw mode 的 JSON 是 Heimdall 產生、每欄帶資料來源標籤的 evidence view，不是 Antigravity 原始 request JSON。

Context 頁會把資料來源分開：

* **System and rules**：有 snapshot 時，來自 snapshot 解出的文字。
* **Tools**：有 snapshot 時是保存的重複 tool entry；沒有時才是 transcript 裡看過的 tool name 和 argument key。
* **Compacted checkpoint**：保存資料裡解出的 `<CONTEXT_SUMMARY>`，有才會顯示。
* **Active history**：snapshot 裡保存的 context record，依保存順序列出。選某一筆時只看那一筆；raw mode 不會偷偷把其他 record 一起塞進來。
* **Latest inbound / staged buffers**：來自 transcript 的觀察，不代表已經證明它們就是 outbound HTTP payload。
* **MCP**：目前只顯示 snapshot system prompt 裡可辨識的 MCP 相關文字；Context 頁不會讀取 `mcp_config.json`、`settings.json` 或 project config，也無法把 tool 歸屬給特定 MCP server。

`field 2` 的數量是重複 wire field 出現的次數；Heimdall 會安全地顯示部分可讀文字與 wire observation，但沒有官方 protobuf schema，因此不能替每筆 record 判定官方 role 或完整語意。

## Cache 狀態

只有成功對應、而且 total/cache 欄位都合理的資料，才可能顯示 `HIT`、`PARTIAL` 或 `MISS`。`UNKNOWN` 代表沒有足夠資料，或欄位彼此不能比較。

Heimdall 不會從本機證據推論 startup cache write、TTL expired、invoice、storage fee 或 output charge。

## Cache-adjusted input projection

* **Effective Input**：公式是 `uncached + cached × cache-input price ratio`。只有同時有可比較 total/cache 欄位的 rows，且 model ID 有 exact match 的官方 pricing profile 時才計算。
* **Gemini 3.7 Flash profile**：Google Gemini Developer API 的 paid standard 表目前是 standard input `$0.75/M`、cached input `$0.075/M`，所以 cache-input price ratio 是 `0.10×`。
* **範圍**：這是每個 model 各自的 standard-input-price equivalent，不是重組出的 request、跨 model 總計、Antigravity invoice、storage fee、output charge，也不能證明 Antigravity 使用 Google API Standard tier。

## 目前支援的資料來源

目前產品只會發現和監看 **Antigravity** session。Session switcher 列的是本機 `.gemini/antigravity-cli` 裡的 conversation；其他 agent 產品不會被顯示成已支援的來源。
