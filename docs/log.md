# ⏱️ LLM Wiki Chrono Log

## [2026-08-27] bugfix | 修復 Docs 頁面因 ANSI 字元污染提前折行之缺陷並實作全寬懸掛縮排 (Docs Full-Width Hanging Indent Layout v0.7.6)：
1. **抓出固定位置提前換行的根本原因**：先前代碼在字串經過 Lipgloss 上色後直接調用 `wrapVisualLines`，隱形的 ANSI 轉義字元（如 `\x1b[38;2;...m`，每個約佔 20 字元）被誤計入視覺欄寬（虛擬累計達 50+ 欄），導致系統誤以為已達到行尾而在螢幕中央（約 50~60 欄處）強行提早折行；
2. **重構純文字先折行與 29 格懸掛縮排 (Hanging Indent)**：實作 `formatDocItem` 與 `wrapPlainText`，在純文字階段以精確終端欄寬（`descMaxWidth = contentWidth - 29`）進行無 ANSI 污染折行，使說明文字完美撐滿延伸至螢幕最右側邊界（第 `contentWidth` 列），且折行後的第 2+ 行自動縮排 29 格，呈現整齊權威的字典排版！
## [2026-08-27] bugfix | 修復 Docs 頁面滾動到底部時 k 鍵延遲與過度滾動 Bug (Docs Scroll Clamping & Zero Overscroll v0.7.5)：
1. **抓出延遲根本原因**：在 `ViewDocs` 中，按 `j` 或 `G` 滾動時未限制上限（`G` 甚至直接賦值 `9999`），導致 `m.docsScroll` 數值遠大於畫面總行數；當使用者按 `k` 往回滾動時，需連按數百下將溢出值扣回可視範圍畫面才會移動，造成卡頓延遲之假象；
2. **完整徹底修復**：實作 `m.getDocsMaxScroll()` 動態計算總行數與視窗可用行數差額，將 `j`、`k`、`Ctrl+d/u` 與 `G` 嚴格約束在 `[0, maxScroll]` 區間內，保證按 `j` 到底後按下第一下 `k` 即刻 100% 毫秒級即時往上滾動！
## [2026-08-27] feature | Docs 頁面多語言 Markdown 嵌入架構與快取標籤辭典 (Docs i18n Embedded Markdown & Cache Badge Glossary v0.7.4)：
1. **補齊快取狀態辭典**：將 `[CACHE HIT]`、`[PARTIAL HIT]`、`[CACHE WRITE]`、`[TTL EXPIRED]` 與 `[CACHE MISS]` 5 大標籤之物理原理、計費折扣與 TTFT 延遲特性完整收錄；
2. **多語言架構解耦 (i18n)**：將文檔按語言分流為 `internal/ui/docs/docs_zh.md` (繁體中文) 與 `docs_en.md` (英文)，透過 Go `//go:embed docs/*.md` 靜態嵌入編譯進單一執行檔；
3. **一鍵切換中英**：在 `[3] Docs` 視圖支援按 **`L`** / **`Tab`** 即時切換繁體中文與英文辭典，標題、過濾器、內容與 Footer 均隨語言即時響應
## [2026-08-27] docs | 建立 Cache Hit vs Partial Hit 物理原理與狀態機分階剖析篇 (Prefix Cache Hit vs Partial Hit Mechanics v0.7.3)：在 `docs/01_raw/2026-08-27_01-52-00_cache_hit_vs_partial_hit_mechanics.md` 深度解密 Google SQLite Protobuf 底層欄位真實本質（僅存儲 `Total` 與 `Cached` 原始數值，無文字狀態欄位），剖析 Observer 領域模型如何透過 $\frac{\text{Cached}}{\text{Total}}$ 計算命中率並劃分 `WRITE`、`HIT` (>=80%)、`PARTIAL` (<80%)、`EXPIRED` 與 `MISS` 五大語意狀態，並詳解大檔案讀取/Tool Output 湧入稀釋命中率之物理場景
## [2026-08-27] docs | 建立開機預熱管線雙重分析漏洞排查篇並標準化 Raw 檔名至「秒」 (Startup Warmup Pipeline & Second-Precision Raw Naming v0.7.2)：
1. 在 `docs/01_raw/2026-08-27_01-48-00_startup_warmup_pipeline_and_double_ingestion_bug.md` 深度記錄開機預熱（Warmup）階段因 `main.go` 與 `watcher.go` 雙重調用 `AnalyzeStep` 導致基線混亂、開機瞬間暫存未就緒與全局最新遙測誤用之四重連鎖根因排查；
2. 將 `docs/01_raw/` 下全體 8 篇技術文件檔名全面標準化升級為包含精確至「秒」的時序命名（`YYYY-MM-DD_HH-MM-SS_<topic>.md`），與 `docs/reviews/` 審查報告命名規範達成完美統一
## [2026-08-27] bugfix | 徹底修復歷史步驟遙測載入與中間步驟非遞增基線 (Historical SQLite Telemetry Linking & Non-Compounding Intermediate Baseline v0.7.1)：
1. **抓出固定 256,000 的根本原因**：
   - 在 `NewWatcher` 初始化時未呼叫 `sqliteReader.PollLatest()`，導致全量 2,300+ 歷史步驟在回放時全部走 Fallback 模式；
   - 在 Fallback 模式中，每遇到一個中間步驟（如 `RUN_COMMAND`）就執行 `state.PrevTotalTokens = totalTokens` 覆寫基線，導致數十個本地步驟連續累加，最終觸頂並死鎖在 `256,000`；
   - 在 `watcher.parseLine` 中，未匹配 SQLite 的步驟誤 Fallback 到「全會話最後一筆 Generation 遙測」，導致早期步驟（Step 100）也顯示最新的 25 萬字；
2. **完整徹底重構**：
   - 在 `NewWatcher` 即刻呼叫 `PollLatest()`，即時精準掛載 690+ 筆歷史 SQLite 世代遙測；
   - 確立「中間步驟不覆寫 `state.PrevTotalTokens` 基線」鐵律，中間步驟僅作為當前 Turn 的局部增量，絕不連鎖累加污染歷史基線；
   - 移除誤用全局最新遙測的 Fallback 分支，完美呈現歷史上下文從 9k $\to$ 11k $\to$ 142k $\to$ 174k $\to$ 192k $\to$ 223k $\to$ 254k 自然且平滑的真實物理增長曲線！
## [2026-08-27] bugfix | 實作 Fallback 物理窗口上限約束與防膨脹截斷 (Physical Context Limit Clamping v0.7.0)：在 `internal/core/analyzer.go` 導入硬性物理窗口上限約束（`maxContextLimit = 256,000`），徹底杜絕連續多個中間過渡步驟因 `state.PrevTotalTokens` 未受限連續累加而虛擬膨脹至 89 萬 Tokens 之 Bug，保證 Fallback 與 Official 遙測皆嚴格鎖定在 256k 物理窗口之內
