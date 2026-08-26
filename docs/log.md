# ⏱️ LLM Wiki Chrono Log

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
