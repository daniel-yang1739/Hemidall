## 2025-03-11 - Context-Aware Toast Notification Colors
**Learning:** Found that all toast messages (status and clipboard) in Heimdall UI previously shared a single `ColorSuccess` background. While creating a single helper to conditionally color these based on string prefixes (⚠️, 💡, 🔴, ❌) works, it required mapping string intent to visual language.
**Action:** When working on CLI/TUI UI libraries like Lipgloss, standardize a "Toast" component rendering helper rather than hardcoding inline styles in `View()` methods.
