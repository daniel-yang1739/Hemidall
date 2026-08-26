# 歷史步驟防抖動鎖定機制 (History Inspection Anti-Jitter Lock) 狀態機設計

- **建立時間**: 2026-08-27 01:32:00
- **更新時間**: 2026-08-27 01:32:00
- **模組歸屬**: `02_architecture`
- **狀態**: `RAW_INTAKE` (待 wiki-distiller 二次提煉)

---

## 🎯 痛點發現：即時遙測流動下的畫面跳動與滾動失焦

在即時觀測（Live Telemetry）系統中，Agent 的工具執行、模型推理與用戶交互是持續非同步產生的。當使用者切換到 **History Explorer（步驟歷史瀏覽器）** 仔細研讀過往某個關鍵步驟（例如 Step #12 的巨大 JSON 回傳值），或是正在右欄 Inspector 滾動閱讀時，會遇到嚴重的**視圖跳動黑天鵝（UI Jittering Trap）**：
1. **步驟逆向漂移**：因為 Step 列表採用「最新在最上（Newest First）」排序，每當一筆新事件進來，陣列長度增加，原本選中的 Step 5 會被硬生生推到下一行，導致左欄游標錯位或突然跳回最新步驟；
2. **閱讀緩衝區中斷**：當使用者在右欄進行 Visual Selection（視覺標記選取）準備按下 `y` 拷貝時，新事件的抵達會重置 Detail 視圖，導致使用者閱讀被強行打斷。

---

## 🧮 數學原理與狀態機凍結演算法

為了實現「**回看歷史時絕對凍結、追隨最新時即時推進**」的流暢體驗，我們設計了 **History Inspection Anti-Jitter Lock（歷史檢視防抖鎖定機制）**。

### 1. 索引映射數學關係
在 `Newest First` 佈局中，UI 游標索引 `selectedIdx`（0 表示最頂部最新）與底層歷史陣列 `m.history` 的物理索引 `realIdx` 關係為：
$$\text{realIdx} = \text{len}(m.history) - 1 - \text{selectedIdx}$$

### 2. 動態推進鎖定推導
當新事件 `AgentEventMsg` 抵達時：
* **情況 A（即時追隨 LIVE 模式）**：
  若 `selectedIdx == 0` 且焦點在左側列表（`focusPane == FocusList`），表示使用者希望即時觀測最新動態，`selectedIdx` 保持為 0，視圖自動推進呈現最新到達的步驟。
* **情況 B（歷史檢視凍結模式）**：
  若使用者正在回看歷史（`selectedIdx > 0`）或正在右欄研讀細節（`focusPane == FocusDetail`），系統觸發防抖鎖定：
  $$\text{selectedIdx}_{\text{new}} = \text{selectedIdx}_{\text{old}} + 1$$
  
此時驗證新事件抵達後的實際指向物理索引：
$$\begin{aligned}
\text{realIdx}_{\text{new}} &= \text{len}_{\text{new}} - 1 - \text{selectedIdx}_{\text{new}} \\
&= (\text{len}_{\text{old}} + 1) - 1 - (\text{selectedIdx}_{\text{old}} + 1) \\
&= \text{len}_{\text{old}} - 1 - \text{selectedIdx}_{\text{old}} \\
&= \text{realIdx}_{\text{old}}
\end{aligned}$$

**結論**：在數學上嚴格保證，使用者當前正在檢視的歷史步驟物理對象，其記憶體位址與 Step Index 保持 100% 絕對不變！

---

## 💻 核心代碼實作 (`internal/ui/model.go`)

```go
case AgentEventMsg:
    m.eventCount++
    m.latestEvent = core.UnifiedAgentEvent(msg)

    // 檢查是否正在進行歷史檢視 (使用者停留在過往步驟或焦點在右欄 Inspector)
    isInspectingPastStep := (m.activeView == ViewHistory && (m.selectedIdx > 0 || m.focusPane == FocusDetail))
    
    // 將新事件 Append 至歷史快照庫
    m.history = append(m.history, m.latestEvent)

    if isInspectingPastStep {
        // 【核心防抖鎖定】：同步遞增選取索引，鎖定當前檢視的歷史步驟不跳動
        m.selectedIdx++
        if m.selectedIdx >= len(m.history) {
            m.selectedIdx = len(m.history) - 1
        }
    } else {
        // LIVE 模式：追隨最新事件
        m.selectedIdx = 0
        m.historyOffset = 0
        m.detailScroll = 0
    }
```

---

## 🧪 單元測試驗證 (`TestHistoryInspectionAntiJitterLock`)

```go
func TestHistoryInspectionAntiJitterLock(t *testing.T) {
    m := NewModel("test-session", false)
    m.activeView = ViewHistory
    m.focusPane = FocusList

    // 預置 5 筆歷史事件
    for i := 0; i < 5; i++ {
        m.history = append(m.history, core.UnifiedAgentEvent{
            StepIndex: i,
            Summary:   fmt.Sprintf("Step %d", i),
            Timestamp: time.Now(),
        })
    }
    m.selectedIdx = 2 // 使用者選取 Step 2

    selectedEv, _ := m.getSelectedEvent()
    if selectedEv.StepIndex != 2 {
        t.Fatalf("Expected selected step to be 2, got %d", selectedEv.StepIndex)
    }

    // 模擬非同步湧入新事件 Step 5
    newEvent := core.UnifiedAgentEvent{StepIndex: 5, Summary: "New incoming Step 5"}
    updated, _ := m.Update(AgentEventMsg(newEvent))
    m = updated.(Model)

    // 驗證：鎖定機制生效，檢視目標嚴格維持在 Step 2
    selectedEvAfter, _ := m.getSelectedEvent()
    if selectedEvAfter.StepIndex != 2 {
        t.Fatalf("Expected inspected step to remain locked at 2, got %d", selectedEvAfter.StepIndex)
    }
}
```
