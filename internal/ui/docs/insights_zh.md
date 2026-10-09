## Insights 使用指南

* **先從這裡開始** : Insights 會依數值排列值得檢查的項目，協助你找到觀察線索；排名高不代表內容浪費。
  * **開啟與操作** : 按 5 開啟 Insights。按 h/l 或 Left/Right 切換分類，按 j/k 或 Up/Down 選擇項目，按 Enter 到 History 查看對應步驟。
  * **尋找本指南** : 按 4 開啟 Docs，再按 / 並輸入 Insights。按 Enter 完成搜尋，按 j/k 捲動內容。

* **先選擇想回答的問題** : Task 從一個 User Step 開始，到下一個 User Step 前結束；中間可以包含多次 Model、Tool 與 Compaction。
  * **Task usage** : 哪些使用者任務讓模型處理最多 Token？每個可觀測 Model Call 都會累加 cached input、uncached input、thinking 與 content output。重複 Context 會在每次呼叫重新計入，因為這是處理量，不是 unique context 大小。
  * **Task output** : 哪些使用者任務在所有可觀測 Model Call 中累積最多可見回答 Token？
  * **Task thinking** : 哪些使用者任務累積最多 Thinking Output Token？較多 Thinking 可能代表困難、探索、困惑或重試，不直接代表品質。
  * **Tool outputs** : 哪些本機工具結果文字最多？TOKENS 是 cl100k_base 本機估算，必要時使用位元組長度 fallback；不包含模型 Thinking 或可見回答。
  * **Compaction** : Checkpoint 不會開始新任務。Task 詳情會顯示次數，以及起始、峰值、結束 Context；不會把它們相減成容易誤解的淨成長。

* **讀懂長條圖** : RELATIVE 以目前分類裡最大的數值為滿格基準。滿格只表示該分類中最大，不代表整個 Session、Context 容量或預算已達 100%。
  * **舉例** : 如果兩筆數值為 5,000 與 2,500 Tokens，前者滿格、後者半格；這不代表前者浪費了 5,000 Tokens。
  * **切換分類** : 各分類有各自的比例尺。不同分類中相同長度的圖條不代表數量相同，新資料也可能改變比例尺。
  * **小數值與零** : 部分色塊讓很小但大於零的數值仍可見。精確數值請以旁邊的數字為準；零與資料不可用是不同狀態。

* **讀排名與證據** : USER STEP 是 Task 的起始提示；STEPS 是該 Task 實際保存的 History Step 數量。Tool output 本身算一個 Step。反白列的 SELECTED EVIDENCE 會顯示在寬版右側或窄版排名下方。
  * **來源** : 檔名及定位資訊指出 User Step、Tool Result 或 Generation 的來源。Task 詳情會顯示完整 Step 範圍、呼叫涵蓋率、Compaction 與 Token 拆解。
  * **Enter** : 開啟對應的 History 步驟，並清除篩選條件。若步驟沒有明確連結或已不存在，系統會提示，不會跳到不相關步驟。
  * **Top N / total** : Insights 最多列出前十筆。窄終端一次顯示的列數較少，可用 j/k 捲動；JSON 匯出保留完整排名。

* **下結論前先檢查資料涵蓋度** : Task 總量只累加可用觀測值，不會虛構缺少資料。重複 Model Call 的 Input 會重複計入，因為它代表實際處理工作量。
  * **Partial task** : 至少一個 Model Call 缺少必要 usage component；顯示數值是可用欄位形成的下限。
  * **Context start / peak / end** : 這些是 Task 內記錄到的 Context 狀態，不能把增長歸因給單一 Prompt 或 Tool Result；Compaction 也可能讓結束值小於起始值。
  * **Usage x/y turns** : Session 摘要仍顯示 generation-level input 覆蓋率，不保證所有 split output 欄位都有資料。
  * **Ready 與最後讀取時間** : 這些資訊描述來源監控狀態，不代表 Agent 正在思考。Read interrupted 會保留最近成功資料並持續重試。

* **建議的檢視流程** : 從 Task usage 開始，打開較大的任務並檢查 Step 數與 Model Call 數。接著比較 Task output 與 Task thinking，再查看同一 Step 範圍內的大型 Tool outputs。
  * **保存分析** : 在 Insights 輸入 :report 並按 Enter。程式會建立 JSON 檔案，保存目前 revision、完整排名與測量方法。
  * **隱私** : 報告不包含原始提示詞或工具輸出文字，但會包含 Session ID、Task Step 編號、模型／工具名稱與證據位置；分享前請先檢查。
