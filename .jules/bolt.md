## 2024-05-24 - MeasureDynamicBaselineTokens Optimization
**Learning:** Found that `MeasureDynamicBaselineTokens()` re-reads files (`AGENTS.md`) and recalculates tokens for constant static schemas (like skills and internal native tool bindings) on every uninitialized session. It adds multiple milliseconds per initialization.
**Action:** Encapsulated BPE calculations for the constants in `MeasureDynamicBaselineTokens()` with `sync.Once`. Use `sync.Once` for any operations relying on purely statically determined files or static string tokenizations within the application lifecycle.
