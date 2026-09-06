## 2024-05-24 - Avoid O(N) string allocation when extracting ExecutionIDs
**Learning:** The parser frequently scans large binary blobs to find execution IDs. Allocating strings from byte slices unconditionally inside a tight loop across a huge db payload adds immense garbage collection overhead.
**Action:** Validate patterns on raw []byte before allocating a string inside loops.
