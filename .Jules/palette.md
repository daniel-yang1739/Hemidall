## 2024-09-06 - Unifying UI State Notifications
**Learning:** Found an instance where identical semantic behaviors (toast notifications in the UI footer) were split across duplicate variables (`m.statusMessage` and `m.clipboardStatus`), causing notifications to overwrite one another or rendering incorrect text contexts. Unifying UI status fields reduces visual glitches and creates a more consistent notification timeline.
**Action:** Always maintain a single source of truth for transient UI elements like toast messages to ensure deterministic UX states.
