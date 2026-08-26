# ⏱️ LLM Wiki Chrono Log

## [2026-08-27] bugfix | 實作 Fallback 物理窗口上限約束與防膨脹截斷 (Physical Context Limit Clamping v0.7.0)：在 `internal/core/analyzer.go` 導入硬性物理窗口上限約束（`maxContextLimit = 256,000`），徹底杜絕連續多個中間過渡步驟因 `state.PrevTotalTokens` 未受限連續累加而虛擬膨脹至 89 萬 Tokens 之 Bug，保證 Fallback 與 Official 遙測皆嚴格鎖定在 256k 物理窗口之內
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
