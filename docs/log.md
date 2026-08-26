# ⏱️ LLM Wiki Chrono Log

## [2026-08-27] bugfix | 修復 Fallback 中間步驟未繼承累積歷史導致 Track 2 僅有 152 Tokens 之計算缺陷 (Fallback Sliding Window History Allocation v0.6.9)：
1. **抓出根本原因**：當步驟屬於未觸發 LLM Generation 的中間事件時（`IsOfficialData == false`），系統走 Fallback 增量計算；先前在 Fallback 區塊中 `event.Tokens.HistoryTokens` 誤直接賦值當前步驟大小 `stepTokens`（152 Tokens），而未扣除 Baseline 計算累積歷史，導致 Track 2 五維度總和驟降至 5,335 Tokens，與 Track 1 的 18.4 萬字產生斷層；
2. **完整推導修復**：在 Fallback 模式計算 `HistoryTokens = totalTokens - (BaseSystem + BaseToolsDef + currentStep)`，並消除 Model 預設值的 `(Official API)` 後綴，使所有步驟的 Track 1 與 Track 2 五維度數據 100% 嚴格守恆！
## [2026-08-27] raw | 沉澱五大主題技術實戰與排查文檔至 docs/01_raw/ (Comprehensive Raw Ingestion v0.6.8)：
1. `2026-08-27_agent_session_storage_architecture_hexagonal_and_ttl.md`: Session 存儲、六角架構適配器模式、WAL 直讀 vs 分析型 Sink DB，以及本地無 TTL vs GPU 顯存 5 分鐘 TTL 冷啟動淘汰機制；
2. `2026-08-27_session_quick_switcher_and_dynamic_discovery.md`: 全域會話快切浮動面板、動態目錄發現、Vim-First 鍵盤巡覽與即時掛載切換管線；
3. `2026-08-27_history_inspection_anti_jitter_lock.md`: 歷史步驟防抖動鎖定機制 (Anti-Jitter Lock) 狀態機與物理索引不變性數學推導；
4. `2026-08-27_tui_lipgloss_box_model_and_ansi_escape_pitfalls.md`: 終端機 UI 排版四大黑天鵝排查實錄（ANSI 轉義序列污染、父容器背景色重置、分隔線過長折行與面板右邊界間隙）；
5. `2026-08-27_shortcuts_modal_and_searchable_docs_page_decoupling.md`: Shortcuts 浮動面板與 View 3 Docs 獨立頁面的職責分離、`/` 即時搜尋與定高盒模型；
6. `2026-08-27_terminal_minimalism_and_layout_polishing.md`: 極簡無 Emoji 終端美學、Header 尾端時間序列排版與 Recent Events 自適應貼合
## [2026-08-27] bugfix | 徹底修復邊框寬度未頂到最右側之 2~4 列間隙 Bug (Pixel-Perfect Full-Width Terminal Alignment v0.6.7)：
1. **抓出右側間隙根本原因**：Lipgloss `PanelStyle.Width(W)` 在設定 `.Border()` 時外框寬度為 `W + 2`，先前因誤扣 padding 設定為 `m.width - 4`，導致面板總寬度僅有 `m.width - 2`（History 甚至僅 `m.width - 4`），比全滿版 Header 窄了 2~4 個字元，造成 Header 尾端 Hash Code 比面板右邊界更突出的視覺落差；
2. **達成 100% 絕對頂到底 (Pixel-Perfect Full-Width Grid)**：將 Dashboard、History 左右雙欄、Docs 與 Header/Footer 全面重構為 `W = m.width - 2`（單欄）與 `listInnerWidth + detailInnerWidth + 4 = m.width`（雙欄），在 80、100、120、140 等全尺寸下所有框線最右邊緣 `╮`、`│`、`╯` 與 Header/Footer 100% 完美貼齊終端機最右側第 `m.width` 列！
## [2026-08-27] style | Recent Live Events 高度內容自適應貼合與 Header 尾部時間序列重構 (Content-Hugging Events Box & Header Clock Sequence v0.6.6)：
1. **Recent Live Events 外框高度自適應貼合 (Content-Hugging Box)**：移除 Dashboard Panel 3 強制填滿螢幕底部的過度空白，改為依實際事件數（最多 6 筆）動態貼合外框高度，排版緊密且自然；
2. **頂部 Header 尾端重構為 `hh:mm:ss | Events: xxxx | (hashhash)`**：將時間戳、事件數與會話短 Hash 依時序遞進排版（如 `01:22:13 | Events: 2200 | (aa726359)`），視覺節奏更具韻律感
## [2026-08-27] style | Recent Live Events 限制最多 6 筆與 Header 尾部格式優化 (Live Events Cap & Header Tail Polish v0.6.5)：
1. **Recent Live Events 限制最多 6 筆**：在 Dashboard 視圖中，將底部 `RECENT LIVE EVENTS` 列表硬性上限鎖定為最多呈現最新 6 筆事件（`maxEventLines <= 6`），避免在大螢幕下過度拉長佔據版面；
2. **頂部 Header 尾端美化為 `(hashhash) | Events: xxxx | hh:mm:ss`**：將當前會話 Hash、事件總數與當前時鐘以標準直槓分隔整齊排列（如 `(aa726359) | Events: 2200 | 01:19:15`），使兩端視覺達到完美平衡
## [2026-08-27] bugfix | 修復 Shortcuts 與 Switcher 浮動面板分隔線折行 Bug (Modal Divider Padding Width Fix v0.6.3)：抓出 Lipgloss `Padding(0, 1)` 使內部可用字元寬度為 `modalInnerWidth - 2`，先前分隔線 `strings.Repeat("─", modalInnerWidth)` 因超出 2 字元被終端折行至下一行產生雙重框線之瑕疵；精確重構為 `contentWidth = modalInnerWidth - 2` 進行分隔線與文字嚴格截斷，徹底消除多餘折行
## [2026-08-27] refactor | 將 Shortcuts 快捷鍵與 Architecture Docs 架構定義解耦分離 (Shortcuts Float Modal & Dedicated Docs Page v0.6.2)：
1. **Shortcuts 獨立為精簡浮動面板 (Floating Shortcuts Modal [?])**：按下 `?` 或 `F1` 彈出居中浮動快捷鍵作弊條，採定寬左右對齊，按 `Esc`、`?`、`q`、`Enter` 隨時收合返回當前畫面；
2. **Docs 獨立為全螢幕第 3 頁面 (View 3: `[3] Docs`)**：專門收錄 5 大 Context 維度解剖、Token 計費真理與滑動窗口物理機制；支援 `/` 即時關鍵字搜尋與 `j/k` 虛擬滾動，外框高度 100% 嚴格鎖定永不推擠變形；
3. **全螢幕 3 大視圖絕對定高防抖 (Pixel-Perfect Strict View Padding)**：在 `Model.View()` 實作全維度自動行數補齊保護，使 Dashboard、History 與 Docs 3 大頁面在 80x24、100x30、120x35、140x40 下永遠 100% 貼齊螢幕底端，底框線與 Footer 絕對無縫吸附
## [2026-08-26] distill-production-ready | 透過 wiki-distiller 完成全量素材深度提煉與二次編譯 (Wiki Distillation v0.5.0)：經 18 位世界前 1% 頂尖專家與讀者展開雙輪對抗審查，大檢察官終審簽核通過。新增 3 篇全新世界級知識卡片：`01_theory/04_Context_Compaction_and_Summarization.md` (雙水位線壓縮與遞迴摘要)、`02_architecture/06_Dual_Track_Telemetry_and_Window_Accounting.md` (雙軌遙測與倒推滑動窗口)、`02_architecture/07_TUI_Engine_and_Terminal_Layout_Mechanics.md` (全螢幕 TUI 引擎與終端盒模型)；深度修訂 `01_theory/03`、`02_architecture/03`、`03_planning/04`；產出全景審查報告書 `docs/reviews/2026-08-26_23-13-21_wiki_distillation_comprehensive_audit_report.md`；同步更新 `docs/index.md`、`02_wiki/index.md` 與 `feedbacks/` 反饋記憶庫；貫徹「消化即刪除 (Digest & Delete)」鐵律安全清理全量 9 篇 Raw 素材，保持素材池極致純淨 (全庫共 16 篇世界級 Wiki 卡片)
