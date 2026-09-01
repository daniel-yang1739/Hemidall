## 2024-05-18 - Avoid repeated regexp compilation
**Learning:** `regexp.MustCompile` inside loop or frequently called functions is a common performance bottleneck in Go. It evaluates compilation every single time.
**Action:** Always move `regexp.MustCompile` to global package scope as `var globalRegex = regexp.MustCompile(...)`.
