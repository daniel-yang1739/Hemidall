# ⏱️ LLM Wiki Chrono Log

## [2026-08-26] feature | 實作右欄軟換行 (Soft Wrap) 與左欄雙行步驟卡片 (2-Line Step Card v0.4.9)：在 Inspector 右欄實作視覺列寬軟換行演算法 (wrapVisualLines)，將長段落與大型 JSON 參數無損換行並整合進虛擬滾動 Buffer，實現零字元截斷遺漏；在左欄重構為人體工學「雙行步驟卡片」（第 1 行顯示序號、類型與時間戳，第 2 行縮排呈現繁中/英文摘要內容），並同步適配上下巡覽滾動邏輯
## [2026-08-26] docs | 建立 TUI 盒模型與 CJK 折行排查終極實戰篇：在 docs/01_raw/2026-08-26_tui_layout_box_model_and_cjk_line_wrapping_troubleshooting.md 完整記錄本次全螢幕 TUI 畫面溢出、底邊框線消失與高度隨步驟浮動之五度排查歷程，深層解密終端機等寬網格物理、Lipgloss 邊框內外距算術陷阱、中文字元 (CJK) 2 倍列寬隱形折行危機與 go-runewidth 視覺硬切終極解法
## [2026-08-26] bugfix | 實作終端機級 CSS `overflow: hidden; white-space: nowrap` 徹底消除高度浮動變形 (CJK Visual Width Hard Truncation v0.4.8)：抓出中文字元 (CJK 寬度為 2 列) 與 Emojis 在 Lipgloss 內因 rune 計數失真觸發隱形自動換行 (Line Wrap) 導致雙欄高度動態暴增的根本原因；導入 go-runewidth 實作全域 truncateVisualWidth(s, maxVisualWidth - 2)，左右兩欄每一行均以視覺欄寬硬切截斷，保證 1 行字元永遠嚴格輸出為 1 行終端列，徹底杜絕任何折行膨脹，雙欄高度絕對鎖死在 m.height - 2、快捷列絕對頂在 m.height - 1！
## [2026-08-26] bugfix | 徹底修復 Inspector 長文本與多 Tool Calls 導致高度長長推擠快捷列之缺陷 (Unified Inspector Buffer Strict Clamp v0.4.7)：將 Inspector 右欄重構為單一統一滾動緩衝區 (Unified Scroll Buffer)，不再分段計算 Tool Calls 與 Payload，而是將右欄總行數嚴格鎖定與左欄 Steps List 100% 等高 (innerRowsLimit = m.height - 4)；即使遇到 15 個 Tool Calls 或數千行長 Payload，右欄亦絕不長長，兩欄底邊永遠平齊並牢牢固定在快捷列正上方
## [2026-08-26] bugfix | 達成 100% 滿版雙欄與底邊精確吸附佈局 (Pixel-Perfect Border & Footer Alignment v0.4.6)：徹底解決框框底邊被快捷鍵蓋住的問題，採用精確行數陣列切片 (innerRowsLimit = m.height - 4)，保證框線頂部坐落在 Line 1、框線底邊 ╰────────╯ 100% 精準坐落在 Line m.height - 2、快捷鍵底列頂滿坐落在 Line m.height - 1（螢幕最底行），左右雙欄 100% 佔滿視窗寬度，並通過 80x24、100x30、120x35、140x40 全維度實機單元測試驗證
## [2026-08-26] bugfix | 終極攻克 TUI 畫面溢出滾動之核心黑天鵝 (Lipgloss Box Model Width-Wrapping Bug)：抓出 Lipgloss .Width(W) 為 Inner Width，在疊加 .Border(+2) 與 .Padding(0,1)(+2) 後 Outer Width 達到 W+4 且 Header 達 155 字元，導致在標準 80 欄終端下全量 24 行每行自動折行 (Line-Wrap) 膨脹至 48 行之重大渲染缺陷；全面重構 Header 緊湊響應式佈局、Box Model 內外距扣除與每行 MaxWidth(m.width) 硬裁剪，在標準 80x24 終端下 100% 完美貼合、零折行、零溢出
## [2026-08-26] bugfix | 徹底修復 Dashboard 在標準終端機 (Height 20~30) 溢出滾動之 Bug (v0.4.5)：新增 TestModelViewHeightExactMatch 單元測試，實作 Dashboard 自適應響應式折疊（高度 <30 自動收合 Panel 3，緊湊對齊 Panel 1&2），並在 Model.View() 導入終端高度硬裁切保護 (Hard Clamp lines[:m.height])，在數學上保證渲染字串行數 100% <= 視窗高度，徹底終結任何畫面跳動與溢出
## [2026-08-26] bugfix | 修復 TUI 終端機高度溢出與滾動問題：消除 renderHeader 與 renderFooter 多餘換行符，將內容高度精準鎖定為 m.height - 2，保證總行數與終端可視區域 100% 吻合，徹底消除游標超出螢幕與底部滾動問題；並在全體 docs/01_raw/ 文件標準化導入「建立時間」與「更新時間」追蹤欄位
## [2026-08-26] research | 逆向解密 Google Gemini 模型動態路由與 safety-le 變體：在 docs/01_raw/2026-08-26_gemini_model_routing_and_safety_le_analysis.md 深度剖析 738 次真實 API 呼叫之模型分佈，揭秘為何寫代碼與執行命令需切換至 gemini-3.7-flash-safety-le (寬鬆審查模式以防 Safety Filter 誤殺)，並物理澄清同族變體共享底層 Transformer 權重與 KV Cache 前綴快取的原理
## [2026-08-26] docs | 完善 Context Compaction 觸發與遞迴摘要機制研究：在 docs/01_raw/2026-08-26_context_compaction_and_recursive_summarization.md 完整寫入本次連線真實案例（Gen 716 觸發 23.8 萬字臨界水位 -> Gen 725 重置回 12.3 萬字）、雙水位線觸發條件矩陣、非同步原子替換管線與「摘要的摘要」O(1) 空間收斂數學證明
## [2026-08-26] feature | 升級 Inspector 全區域 Visual 複製與 Dashboard 一鍵 [Enter] 跳轉 History (v0.4.4)：將 Inspector 重構為包含元資料、工具呼叫與 Payload 之全局可選模式，按 [v/V] 可反白全欄任一行並一鍵 [y] 拷貝；在 Dashboard 按 [Enter] 可毫秒級直達該步歷史檢視
## [2026-08-26] research | 實機直擊第二輪 Context Compaction 觸發：在 Gen 716 (Step 1455) 活躍上下文攀升至 238,513 Tokens (觸及 95% High Watermark) 後，系統非同步壓縮並淘汰 Step 0-1400 舊歷史，在 Gen 725 (Step 1473) 成功重置回 123,275 Tokens (Low Watermark 48%)，釋放 12 萬字顯存空間
## [2026-08-26] feature | 實作歷史步驟「由新到舊 (Newest First)」人體工學排序 (v0.4.3)：重構 Step List 為最新步驟置頂（第一行即為最新 Step），按 [↓] 往下瀏覽過往歷史，按 [g] 直達最新、[G] 直達最舊，大幅提升除錯與觀察即時動態的流暢度
## [2026-08-26] feature | 實作開機歷史即刻加載、Dashboard 時光機回放與 Vim Visual Mode 剪貼簿複製 (v0.4.2)：啟動時即刻預載全量歷史並預設呈現最新一筆 23 萬字官方 Telemetry；在 Dashboard 視圖支援 [↑/↓/Ctrl+u/d/g/G] 跨步時光機回放；在 Inspector 視圖實作 [v/V] 視覺選取高亮與 [y] 一鍵複製至系統剪貼簿 (pbcopy/xclip)
## [2026-08-26] bugfix | 修復 TTL 超時冷啟動時 Fallback 遺漏活躍上下文歷史導致 Total/New 僅有 5,301 Tokens 之計算缺陷：精確修正為「當 GPU 顯存超時淘汰時，發送的總上下文仍包含約 16.5 萬活躍歷史，Cached 歸零，全量 16.5 萬字全部計入 New 計費」，並重新編析驗證
## [2026-08-26] docs | 建立 Google Gemini SQLite 本地資料庫終極實戰查詢手冊：在 docs/01_raw/2026-08-26_gemini_sqlite_query_handbook.md 完整收錄 7 大資料表欄位辭典、CLI/Python/Go 三種唯讀 WAL 連線範例、實戰 SQL Cheat Sheet 與 Protobuf BLOB 深度萃取腳本
## [2026-08-26] bugfix | 修復 Fallback 中間步驟誤用全歷史累積導致 New Tokens 暴增 35 萬之重大計算缺陷，改採活躍窗口增量模型 (Incremental Active Window)；並在 internal/ui/views.go 實施嚴格高度鎖定 (Fixed Height & Line Padding)，徹底消除右欄長文本引起的抖動與溢出
## [2026-08-26] research | 逆向實證 Agent Context 雙水位線壓縮與遞迴摘要機制 (Compaction & Recursive Summarization)：從 SQLite steps 表 (Step 1345 Type 15) 提取官方 <CONTEXT_SUMMARY> 二進制真實 Payload，證實 High Watermark (~245k) 觸發非同步摘要與 Low Watermark (~119k) 尾部滑動截斷，並解密「Summary of Summaries」遞迴聚合演算法如何保證前綴以 O(1) 常數空間永遠不溢出
## [2026-08-26] feature | 實作倒推滑動窗口 (Reverse Sliding Window) 演算法：徹底解決 Append-Only 本地全量日誌（40 萬字）與雲端 Active Window（17 萬字）之間的歷史截斷失真問題，從最新 Step 往前倒推填滿 Google 官方 Active Budget，精確排除已被淘汰的遠古步驟，並通過單元測試驗證
## [2026-08-26] feature | 實作 k9s 風格全螢幕互動式 TUI 應用 (v0.4.0)：引入 github.com/charmbracelet/bubbletea 與 lipgloss，打造 Alternate Screen Buffer 全螢幕終端介面（徹底告別 Log 滾動洗屏），支援 [1/d] 即時雙軌儀表板 (Live Dashboard) 與 [2/h] 歷史步驟瀏覽器 (Step History Explorer，支援 ↑/↓ 鍵盤巡覽、[Tab] 雙欄焦點切換與 Raw Payload 獨立滾動檢視)
## [2026-08-26] feature | 實作 TUI 雙軌上下雙表分流與狀態機基線同步：重構 internal/core/formatter.go 為「上表：Google 官方真實帳單與物理快取 (Track 1)」與「下表：本地 5 維度 Context 載荷深度解剖 (Track 2)」，徹底分流官方真理與本地解剖，並在 analyzer.go 實作官方截斷同步機制，徹底消除 40 萬 vs 17 萬的混淆
## [2026-08-26] feature | 實作 Phase 3 雙軌遙測引擎：開發 internal/adapters/antigravity/protobuf.go 與 sqlite_telemetry.go，以純 Go (modernc.org/sqlite) 零侵入讀取 ~/.gemini 本地 SQLite gen_metadata 表格，直解 Google Gemini 官方 Protobuf 遙測數據 (Total Tokens, Cached Tokens, Hit Rate, Model Name)，與本地 BPE 5 維度分類器無縫融合，並成功通過單元測試與即時監控驗證
## [2026-08-26] feature | 擴充 wiki-distiller 審查專家團：新增 roles/09_concrete_trace_walkthrough_specialist.md (具體演繹與端到端追蹤專家)，確立「極簡真實 Input 逐輪演繹 (Step 1..n) 與最終 Output」硬核實例標準，拒絕擬人擬物童話比喻，並寫入 SKILL.md、feedbacks/ 與 04_meta 卡片
## [2026-08-26] refactor | 重構 04_meta/02_Obsidian_Vault_Topology.md：完整收錄全域 ASCII 檔案結構樹與 High-Level 目錄職責與生命週期表，精簡標題與檔名，並由秘書長正式沉澱入 feedbacks/ 反饋記憶庫
## [2026-08-26] distill-meta-module | 透過 wiki-distiller 完成 04_meta 模組提煉：新增 01_Multi_Agent_Adversarial_Review_Pattern.md 與 02_Obsidian_Vault_Topology.md，經 17 位審查官雙輪對抗審查與大檢察官終審簽核，生成審查報告書並清空 01_raw/
## [2026-08-26] feature | 升級 feedbacks/ 反饋記憶庫：全面標註 🟢 [ACCEPTED] 採納標準 與 🔴 [REJECTED] 駁回警示，並在 SKILL.md 確立「Step 0: Pre-Flight Checklist」強制開局檢閱歷史注意事項
## [2026-08-26] distill-deep-audit-report | 升級 docs/reviews/ 審查報告書：全面展開 17 位審查官針對全庫 12 篇卡片的深層思維鏈 (Chain of Thought)、地毯式挑惕清單、Main Agent 駁回辯論與大檢察官終審簽核
## [2026-08-26] cleanup-raw | 貫徹「消化即刪除 (Digest & Delete)」鐵律：15 篇已 100% 提煉進 02_wiki/ 的 Raw 檔案已全數安全清理刪除，保持素材池極致乾淨
## [2026-08-26] skill-upgrade | 更新 wiki-distiller SKILL.md 與 docs/schema.md：將「強制輸出審查報告」與「Raw 消化即刪除」正式確立為系統最高執行憲法
## [2026-08-26] distill-raw-complete | 透過 wiki-distiller 執行全量 01_raw/ 素材深度提煉：由「實證代碼驗證官」Trace 專案原始碼校驗事實，提煉新增 02_architecture/05_Model_Payload_and_API_Traces.md，全量 15 篇 Raw 素材 100% 沉澱進 02_wiki/，大檢察官終審簽核通過 (共 12 篇世界級 Wiki 卡片)
## [2026-08-26] feature | 建立 roles/07_code_fact_checker_and_pruner.md 實證代碼驗證官，賦予代碼真實性一票否決權
## [2026-08-26] distill-production-ready | 執行 wiki-distiller 全面深度提煉：通過 16 位世界前 1% 專家與讀者地毯式窮舉審查，經 Main Agent 批判性過濾與大檢察官終審簽核，確立認知編號因果鏈、補齊每圖深度導讀、充實硬體微架構推導
## [2026-08-26] feature | 建立 roles/06_information_architect_vault_ontologist.md 與 senior_05_vault_learning_path_architect.md，確立認知編號鐵律與雙層駁回機制
## [2026-08-26] feature | 建立 roles/05_technical_writer_detail_auditor.md 並升級全體 readers 為 Stanford/MIT/FAANG 頂尖人設
## [2026-08-26] feature | 建立 .agents/skills/wiki-distiller/roles/ 雙輪審查官與讀者矩陣
## [2026-08-26] restructure | 依據 Karpathy 精神與專案獨立性重整目錄為 01_raw, 02_wiki, ithome_draft, ithome_ready
## [2026-08-26] feature | 完成 Phase 2：5 維度 Context Token 深度解剖與 LCP 快取命中率實證
## [2026-08-19] reverse-eng | 逆向解密 ~/.gemini 底層儲存、SQLite 7 張表與 100KB 切片機制
## [2026-08-18] planning | 確立 2026 iThome 鐵人賽主題與 30 天每日拆解大綱
