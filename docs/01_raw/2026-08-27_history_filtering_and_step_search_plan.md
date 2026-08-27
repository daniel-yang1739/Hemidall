# History Explorer Multi-Dimension Filter & Step Search Implementation Plan

> **Goal**: Provide blazing-fast, intuitive, and clutter-free filtering in the History Explorer view (`ViewHistory`):
> 1. **Type Filter**: Filter by step type (`All`, `Tool Call`, `Model Response`, `User Input`, `Code Action`, `Generic`).
> 2. **Cache Status Filter**: Filter by 5 cache states (`All`, `HIT`, `PARTIAL`, `MISS`, `NEW`, `BROKEN`).
> 3. **Step Number Search (`/`)**: Instant numerical step search/jump (e.g. `/14` jumps/filters to `#014`).

---

## 🎨 Proposed UI/UX Designs

### Option 1 (Recommended): High-Efficiency Inline Badges + Fast `/` Search Bar

In the left Step List pane, display compact active filter badges on the header line, with an optional collapsible search input bar when `/` is pressed:

```text
╭────────────────────────────────────────╮╭──────────────────────────────────────────────╮
│ STEPS (12/540)  [T:Tool] [C:Hit]       ││ STEP INSPECTOR                               │
│ Filter: [#14█      ] (Esc to clear)    ││ • Step 014 (DONE) at 10:20:15 | Type: TOOL   │
│ ────────────────────────────────────── ││ • Tokens: 1540 | Cached: 1200 (77.9%) [HIT]  │
│   ...                                  ││ ───────────────────────────────────────────  │
│ > [014|TOOL ] 10:20:15  [HIT 77.9%]    ││ Tool: replace_file_content                   │
│     replace_file_content               ││                                              │
│   [012|TOOL ] 10:18:40  [HIT 82.1%]    ││                                              │
│     write_to_file                      ││                                              │
│   [008|TOOL ] 10:14:22  [HIT 91.0%]    ││                                              │
│     run_command                        ││                                              │
│   ...                                  ││                                              │
╰────────────────────────────────────────╯╰──────────────────────────────────────────────╯
  [t] Cycle Type  [c] Cycle Cache  [/] Search Step #  [Esc] Clear  [h/l] Pane
```

#### Key Advantages:
1. **Zero Clutter**: Takes only 1 line when searching, 0 extra lines when inactive.
2. **Instant Hotkeys**:
   * Press **`t`**: Cycles Type Filter (`[T:All]` $\to$ `[T:Tool]` $\to$ `[T:Model]` $\to$ `[T:User]` $\to$ `[T:Code]` $\to$ `[T:All]`).
   * Press **`c`**: Cycles Cache Status (`[C:All]` $\to$ `[C:Hit]` $\to$ `[C:Partial]` $\to$ `[C:Miss]` $\to$ `[C:Broken]` $\to$ `[C:All]`).
   * Press **`/`**: Activates search cursor; type numbers (`14`) to jump/filter to Step `#14`. Press `Esc` to clear.
3. **Step Badge with Cache Indicator**: Step list cards show cache badges (`[HIT 77.9%]`, `[MISS]`).

---

### Option 2: Dedicated Filter Modal Panel (`f` / `F2`)

Pressing `f` opens a pop-up filter config modal:

```text
╭────────────────── FILTER STEP HISTORY ──────────────────╮
│ Type Filter    : < All | Tool Call | Model | User >     │
│ Cache Status   : < All | HIT | PARTIAL | MISS | BROKEN >│
│ Step Number (#): [ 14_                                ] │
│                                                         │
│ Matching Steps : 12 / 540 (2.2%)                        │
│ ─────────────────────────────────────────────────────── │
│  [Tab/Shift+Tab] Move Field  [←/→] Select  [Enter] Apply│
╰─────────────────────────────────────────────────────────╯
```

#### Key Advantages:
* Centralizes all filter criteria in one form.
* Ideal for complex multi-condition queries.

---

### Option 3: Top Filter Sub-Tab Bar (Always Visible)

Under the `STEPS` header, display two horizontal badge rows:

```text
╭────────────────────────────────────────╮
│ STEPS (12/540)                         │
│ Type : [All]  Tool*  Model  User  Code │
│ Cache: [All]  Hit*  Partial  Miss      │
│ ────────────────────────────────────── │
│ > [014|TOOL ] 10:20:15                 │
│     replace_file_content               │
```

---

## 🏗️ Technical Architecture & Data Model

### 1. State Extensions in `internal/ui/model.go`
```go
type TypeFilter string
const (
    TypeFilterAll          TypeFilter = "ALL"
    TypeFilterToolCall     TypeFilter = "TOOL"
    TypeFilterModelResp    TypeFilter = "MODEL"
    TypeFilterUserInput    TypeFilter = "USER"
    TypeFilterCodeAction   TypeFilter = "CODE"
    TypeFilterGeneric      TypeFilter = "GENERIC"
)

type CacheFilter string
const (
    CacheFilterAll     CacheFilter = "ALL"
    CacheFilterHit     CacheFilter = "HIT"
    CacheFilterPartial CacheFilter = "PARTIAL"
    CacheFilterMiss    CacheFilter = "MISS"
    CacheFilterBroken  CacheFilter = "BROKEN"
)

// In Model struct:
type Model struct {
    ...
    historyTypeFilter  TypeFilter
    historyCacheFilter CacheFilter
    historyStepQuery   string
    isHistorySearching bool
    filteredHistory    []core.UnifiedAgentEvent
}
```

### 2. Fast Step Filter Pipeline
```go
func (m Model) getFilteredHistory() []core.UnifiedAgentEvent {
    var result []core.UnifiedAgentEvent
    for _, e := range m.history {
        // 1. Type Filter
        if m.historyTypeFilter != TypeFilterAll {
            if !matchType(e.Type, m.historyTypeFilter) {
                continue
            }
        }
        // 2. Cache Filter
        if m.historyCacheFilter != CacheFilterAll {
            if e.CacheStatus != string(m.historyCacheFilter) {
                continue
            }
        }
        // 3. Step Number Query (e.g. "14" or "014")
        if m.historyStepQuery != "" {
            stepStr := fmt.Sprintf("%d", e.StepIndex)
            if !strings.Contains(stepStr, m.historyStepQuery) {
                continue
            }
        }
        result = append(result, e)
    }
    return result
}
```

### 3. Keybindings Dispatch in History View
* **`t` / `T`**: Cycle `historyTypeFilter`
* **`c` / `C`**: Cycle `historyCacheFilter`
* **`/`**: Set `isHistorySearching = true`, type step digits (`0`~`9`, `backspace`)
* **`Esc`**: Clear search query, reset filters, or return focus to left pane
* **`Enter`**: Exit search input mode, lock selection, and inspect step

---

## 🧪 Verification & Test Suite
1. `TestHistoryFilterByType`: Verify filtering by Tool Call, Model Response, User Input.
2. `TestHistoryFilterByCacheStatus`: Verify filtering by HIT, PARTIAL, MISS, BROKEN.
3. `TestHistorySearchByStepNumber`: Verify `/` input, number matching, and dynamic scrolling to target step.
4. `TestZeroHeightVariationWithFilters`: Verify no height variation across all terminal sizes when filter bar appears/disappears.
