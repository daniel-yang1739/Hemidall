# ⏱️ LLM Wiki Chrono Log

## [2026-08-28] distill | 官方遙測真理、Multi-Agent 經濟學、四大時序排查與全局計價矩陣 Wiki 深度提煉 (Wiki Distillation & Forensics Audit)：
1. **全新架構卡片落地 (`02_architecture/10` & `11`)**：
   * `10_Dashboard_Aggregate_Metrics_and_Multi_Model_Pricing.md`：跨輪次總吞吐量、Effective Tokens 等效字數折扣矩陣（Flash 75%, Sonnet 90%）、即時貨幣換算（`$` 鍵循環切換）與 Google AI Pro 5,000 RPD 配額消耗模型；
   * `11_Multi_Agent_Hierarchy_and_Subagent_Token_Economics.md`：Subagent 獨立 Context 生命週期、四階段計費拆解、本地 Tool 0 GPU Token 原則與 `status = 7` (BLOCKED) 沙盒防禦；
2. **四大 SRE 四段式實戰故障覆盤卡片落地 (`05_troubleshooting/07..10`)**：
   * `07_Idle_TTL_Masking_by_Local_User_Input_Timestamps.md`：10 分鐘閒置快取未過期之謎、本地打字 `USER_INPUT` 時序遮蔽與 `LastCloudTurnTime` 專屬時鐘隔離修復；
   * `08_Stream_Update_Duplication_and_Step_Counter_Inflation.md`：事件數 10,336 與步驟序號 5,919 脫節之謎、串流 `RUNNING` $\to$ `DONE` 重複累加與 In-Place 原地覆蓋修復；
   * `09_Harness_Internal_Plumbing_Filtering_and_SQLite_Gap_Recovery.md`：消失的 31 個步驟與跳號之謎、Google 內部管線過濾機制剖析與 `MergeMissingSQLiteSteps` 全量補齊；
   * `10_Destructive_History_Filter_vs_Non_Destructive_Jump_Navigation.md`：歷史搜尋 Context 丟失之謎、破壞式過濾修復與 Vim Jump-to-Step 導航器平滑錨定重構；
3. **全庫雙向鏈接與 MOC 矩陣同步**：
   * 同步升級 `docs/index.md`、`02_wiki/index.md`、`02_architecture/index.md`、`05_troubleshooting/index.md`、`01_theory/03` 與 `02_architecture/06`；
4. **19 角色雙輪審查與全景報告書**：
   * 由 19 位頂尖專家與讀者完成雙輪對抗審查，產出 `docs/reviews/2026-08-28_01-25-00_wiki_distillation_and_forensics_audit_report.md`；
5. **秘書長反饋庫沉澱與 Digest & Delete 執行**：
   * 更新 `feedbacks/` 5 大記憶庫，並安全清理 `docs/01_raw/` 8 份已消化原始素材。
1. **全新架構卡片落地 (`02_architecture/09_History_Explorer_and_Causality_Graph.md`)**：
   * 提煉雙軌正交過濾引擎（`[T:Type]` 與 `[C:Cache]` 嚴格隔離）；
   * 提煉增量步驟搜尋（`/` 與 `n`/`N` 跳轉）、方案 B 緊湊連續括號（`┌[` / `│[` / `└[`）與雙向因果導航（`p` 跳父步驟，`c`/`C` 跳已消費子步驟）；
   * 包含標準 4 維度圖解導讀與 Minimal Input $\to$ Step 1..n Trace $\to$ Final Output 實例；
2. **核心架構卡片升級 (`02_architecture/03` & `02_architecture/07`)**：
   * `03_Agent_Storage`：升級 Universal 4 態 FSM、`StepLinkageTracker` 親緣追蹤器與跨平台可移植性（Google Antigravity, Claude Code, OpenCode, Codex）；
   * `07_TUI_Engine`：升級 3-Panel 雙模式響應式盒模型（全寬 $\ge 100$ 欄 vs 半寬 $< 100$ 欄）、方案 B 括號與動態行數打包演算法（Dynamic Line Packing）；
3. **三大 SRE 四段式實戰故障覆盤卡片落地 (`05_troubleshooting/04..06`)**：
   * `04_Filter_Isolation_and_Cache_Expired_Boundary_Leak.md`：EXPIRED 步驟洩漏至 MISS 篩選漏洞、顯式互斥排除守衛與 Runbook SOP；
   * `05_USER_Input_Inbound_Intent_vs_GPU_Cache_Settlement.md`：USER 輸入步驟誤標合成 MISS 標籤與時序結算錯位、意圖暫存語意與雲端帳單解耦；
   * `06_Single_Line_Card_Static_Packing_Blank_Gap.md`：單行卡片靜態除二計算導致清單底部 14 行留白、動態累加行數打包修復；
4. **雙向拓撲互錨與 MOC 同步**：同步更新 `docs/index.md`、`02_wiki/index.md`、`02_architecture/index.md` 與 `05_troubleshooting/index.md`；
5. **19 角色雙輪審查與全景報告書**：由 19 位世界前 1% 專家與讀者完成雙輪對抗審查，產出 `docs/reviews/2026-08-27_15-26-00_history_explorer_and_troubleshooting_audit_report.md`；
6. **秘書長反饋庫沉澱與 Digest & Delete**：更新 `feedbacks/` 5 大記憶庫，並安全清理 `docs/01_raw/` 11 份已提煉素材。
1. **通用 4 大計算與計費範疇 (Universal 4 Scopes)**：在 `UnifiedAgentEvent` 引入物理完備的 4 大範疇（`USER` 使用者意圖、`CLOUD` 雲端 LLM 決策/計費、`LOCAL` 本機離線 0 元執行、`SYSTEM` 系統開局約束），消除所有 Token 計費與模型歸屬混淆；
2. **確定性單執行緒狀態機 (`StepLinkageTracker`)**：以 $O(N)$ 線性單次遍歷精確關聯 `ParentStepIdx`（觸發父層）、`PackagedInStepIdx`（打包計費雲端輪次）與 `ConsumedStepIndices`（消耗本機步驟清單），100% 支援單輪平行多工具呼叫 (Parallel Tool Calls)；
3. **Step Inspector 雙軌管線與自解釋渲染**：本地步驟清晰標註 `• Origin: Local Host Process`、`• Billing: Offline (0 tok) ➔ Packaged in Step #N (+X tok) [n] Jump`；雲端步驟標註 `• Model: Gemini 3.7 Flash (Official Telemetry)`、`• Input: Consumed Local Step #N [n] Jump` 與官方 5 維 Token 分佈；
4. **焦點在 Inspector 時支援 `p` / `n` 階層極速跳轉**：
   * 按 **`p` (Parent)**：直接向上跳轉至當前步驟的父層（如從 `RUN_COMMAND` 跳回觸發它的 `TOOL_CALL`，或從 `TOOL_CALL` 跳回原始 `USER_INPUT`）；
   * 按 **`n` (Next / Child / Consumed)**：直接向下跳轉至子層（如從 `TOOL_CALL` 跳至執行的 `RUN_COMMAND`，或從 `RUN_COMMAND` 跳至打包計費它的雲端輪次）；
   * 同步更新左欄選取框並動態校準 `historyOffset`，保證選取卡片永遠位處螢幕可視區；
5. **左欄清單樹狀分支符號渲染**：第二層本地執行步驟自動以 `└── [012|CMD  ] 10:18:40 (Local)` 縮排呈現，第一層齊左凸顯雲端關鍵計費節點，視覺心智模型極致清晰；
6. **全單元測試 100% 通過**：新增 `TestStepLinkageTracker_ParallelToolsAndTurns` 與 `TestHistoryParentChildNavigationPN`，保證跨 Agent 架構移植性與 TUI 導航穩定性
1. **Step Type 類別過濾 (`t` / `T`)**：按 `t` 鍵快速循環切換 `[T:All]` $\to$ `[T:Tool]` $\to$ `[T:Model]` $\to$ `[T:User]` $\to$ `[T:Code]` $\to$ `[T:Generic]` $\to$ `[T:All]`，瞬間萃取目標步驟；
2. **Cache 狀態過濾 (`c` / `C`)**：按 `c` 鍵快速循環切換 `[C:All]` $\to$ `[C:Hit]` $\to$ `[C:Partial]` $\to$ `[C:Miss]` $\to$ `[C:Broken]` $\to$ `[C:All]`，精確鎖定快取命中或斷裂點；
3. **'/' 數字步驟編號極速搜尋與跳轉**：按 `/` 喚出 `Filter: [#14█]` 搜尋列，輸入任意數字（如 `14`），清單毫秒級篩選並跳轉至 `#014`，按 `Esc` 瞬間清空復原；
4. **左欄快取狀態微徽章與標題徽章**：步驟清單卡片右側自動依色彩標註 `[HIT 78%]` / `[PART 40%]` / `[MISS]` / `[BROKEN]`，標題列精確統計當前篩選命中數與總數（如 `STEPS (12/540) [T:Tool] [C:Hit]`）；
5. **智能視窗邊界連動**：篩選與搜尋時自動重新校準可視卡片容量，徹底杜絕文字溢出或底部遮擋，全螢幕零高度抖動
## [2026-08-27] feat | 會話快切中樞升級 (Quick Switcher v2.0)：雙欄即時預覽、對話意圖與進度雙軌識別、純視覺標籤頁與零 Emoji 俐落排版：
1. **純視覺多 Agent 標籤頁 (Real Visual Tabs)**：移除非必要的 "Tabs:" 文字與 "All" 標籤，直接以原生標籤按鈕 `[Antigravity (N)]` / `Claude Code (0)` / `OpenCode (0)` 呈現，支援按 **`[` / `]`** 鍵快速循環切換；
2. **左欄極速掃描與末端目錄提取**：自動將長路徑精簡為末端 2~3 層目錄（如 `self/ithome2026 (#aa726359)`），每張卡片僅保留目錄、Hash、步驟數與相對時間，大幅降低掃描干擾；
3. **右欄深度透視 (Focused Live Inspector)**：移除非必要 Overview 區塊，垂直空間 100% 聚焦呈現 **`[INITIAL GOAL / FIRST PROMPT]`**（創立主題）與 **`[LATEST PROGRESS / LAST ACTION]`**（最新進度，完美辨識 Fork 分叉會話）；
4. **零 Emoji 工業級純文字排版**：全面根絕 Emoji 雜訊，統一採用 ASCII 盒模型、專業文字徽章與 Lipgloss 精準終端渲染；
5. **多欄位模糊過濾**：支援同時依照專案目錄名、Short Hash、起始提問或最新進度關鍵字即時搜尋
## [2026-08-27] plan | 升級 2026 鐵人賽總企劃、30 天大綱與路線圖至 v2.0 (Master Plan & 30 Days Breakdown v2.0)：
1. **確立核心精神與首尾呼應情感錨點**：
   * Day 01 開篇 Hook：《至少直到最後一刻，我與 AI 共舞著 —— 寫在黑盒時代前夕的工程自白》；
   * Day 30 終章殘響：《一個軟體工程師的時代殘響：當黑盒化為透明，至少直到最後一刻，我與 AI 共舞著》；
   * 詮釋身處 AI 典範轉移時代的軟體工程師從熱愛、抗拒、接受、擁抱到超越的心路歷程與工程驕傲；
2. **重構 30 天四大模組架構 (v2.0)**：
   * **模組一 (Day 01~07)**：AI 記憶與推論底層物理（新增 Day 02/03 Self-Attention QKV 幾何學與 GQA 基礎課，平緩學習坡度）；
   * **模組二 (Day 08~15)**：打造大腦聽診器 Agent-Observer（雙軌遙測、六角架構、全螢幕雙軌 TUI、會話快切與防抖動鎖）；
   * **模組三 (Day 16~21)**：擴充萬能 Agent 生態系（解剖與實作 Claude Code 5分鐘 TTL 斷點適配器、OpenCode/Ollama 通用適配器與三大 Agent 橫向實彈評測）；
   * **模組四 (Day 22~30)**：極致壓縮引擎、長程任務壓測與工程師終章（語義剪枝、Diff 差分壓縮、80% Benchmark、SRE 故障手冊與開源發布）；
3. **同步更新規劃模組**：同步升級 `03_planning/01_Master_Plan.md`、`03_planning/02_30_Days_Breakdown.md` 與 `03_planning/04_Phased_Implementation_Roadmap.md`
## [2026-08-27] polish | 全域 Tab/Shift+Tab 循環切頁、Docs 't' 語系切換與 History 'h/l' Vim 左右分欄導航 (Vim-First Cyclic Tabs & Conflict-Free Shortcuts v0.8.2)：
1. **全域循環切頁 (Cyclic View Switching)**：支援按 **`Tab`** 順時針循環切換分頁 (`[1] Dashboard` $\to$ `[2] History` $\to$ `[3] Docs` $\to$ `[1] Dashboard`)，按 **`Shift+Tab`** / **`Backtab`** 逆時針切換，提供流暢的現代 TUI 瀏覽體驗；
2. **解決 Vim 左右鍵衝突**：
   * **語系切換改為 `t` (Translate / Toggle)**：在 Docs 頁面改用 `t / T` 切換繁體中文與英文辭典，徹底釋放 `l` 鍵；
   * **History 雙欄導航回歸純粹 Vim**：在 History View 中，按 **`l`** (或 `Enter` / `Right`) 進入右欄 Inspector，按 **`h`** (或 `Esc` / `Left`) 返回左欄 Steps List；
   * **Docs 視圖直切改為 `3 / i` (Info/Insight)**：移除全域 `h` 快捷鍵，根除在歷史檢驗時按 `h` 誤跳頁之衝突；
3. **全面同步**：同步更新 Header Badge (`[t: 繁體中文]`)、搜尋提示、底部 Footer 狀態提示與 `?` Shortcuts 浮動面板
## [2026-08-27] distill | 19 角色雙輪對抗審查完成、落地 05_troubleshooting 模組並全量消化 10 份 Raw 素材 (Full Distillation & Troubleshooting Module v0.8.1)：
1. **建立 05_troubleshooting 實戰排查手冊**：產出 3 篇具備四段式結構與 Runbook SOP 的 Postmortem 卡片（89 萬字基線膨脹排查、開機雙重分析排查、ANSI 隱形佔位腰斬排查）與模組索引；
2. **新增架構卡片 08**：產出 `02_architecture/08_Interactive_Session_Switching_and_Anti_Jitter.md`，解密全域會話快切中樞與歷史步驟防抖動鎖定機制；
3. **全面深化核心卡片**：更新 `01_theory/03` (Partial Hit 稀釋機制)、`02_architecture/03` (六角架構與 WAL 直讀)、`02_architecture/06` (中間步驟非遞增基線) 與 `02_architecture/07` (ANSI 感知狀態機與嵌入式 Markdown i18n)；
4. **輸出全景審查報告書**：於 `docs/reviews/2026-08-27_02-15-00_multi_agent_adversarial_review_audit_report.md` 留存 19 角色雙輪審查全景紀錄；
5. **落實 Digest & Delete**：100% 安全清理刪除 `docs/01_raw/` 下已完全消化的 10 份原始素材；
6. **更新雙層 MOC 索引**：同步維護 `docs/index.md` 與 `docs/02_wiki/index.md`，全庫保持 0 孤島雙向鏈接
## [2026-08-27] skill | 升級 wiki-distiller 審查體系：擴充 QA/Runbook 專家與明確雙輪對抗審查閉環 (Two-Round Adversarial Review & Incident Runbook Role v0.8.0)：
1. **新增角色 10**：建立 `roles/10_qa_runbook_incident_troubleshooting_expert.md` (實戰排查、QA 問答與 Runbook 知識化專家)，審查所有 Bug 排查與 QA 卡片是否符合「現象定義 $\to$ 根因代碼溯源 $\to$ 架構修復 $\to$ 總結與 Runbook SOP」四段式標準，並強制與核心概念頁面建立雙向鏈接；
2. **新增反饋記憶庫 05**：在 `feedbacks/05_qa_troubleshooting_and_runbooks.md` 沉澱排查四段式結構、診斷 SOP 與防孤島抗體規範；
3. **明確雙輪對抗審查閉環 (2-Round Loop)**：在 `SKILL.md` 正式定型 Round 1 (19 位審查官初審) $\to$ Main Agent Triage $\to$ Fix 1 (首輪修訂) $\to$ Round 2 (19 位原班人馬 Delta 差量複驗) $\to$ 大檢察官終審簽核之完整閉環流水線
## [2026-08-27] config | Docs 頁面預設語言切換為英文 (Default English for Docs Page v0.7.9)：將 Docs 視圖之預設語系初始化為 English (`docsLang: "en"`)，預設載入 `docs_en.md`，並可隨時按下 `[l]` 鍵即時無縫切換為繁體中文辭典 (`docs_zh.md`)
## [2026-08-27] polish | 統一多語言切換提示為小寫 `[l]` 鍵與 Shortcuts 說明補齊 (Lowercase Language Toggle Hint & Shortcuts Docs v0.7.8)：
1. **提示字元統一為小寫 `l`**：將 Docs 標題列 `[l: 繁體中文]`、搜尋狀態提示 `[按 l 切換中英]` 及底部狀態列 `[l] Lang (繁中)` 全面統一為小寫 `l`，符合終端機鍵盤操作習慣；
2. **補齊快捷鍵浮動面板**：在 `?` Shortcuts Modal 中的 `Docs View Controls` 區塊新增 `l / Tab` 切換中英文辭典之操作提示
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
