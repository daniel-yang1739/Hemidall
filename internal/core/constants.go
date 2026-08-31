package core

// Cross-File Shared Domain Constants
// Strictly adheres to AGENTS.md Article V: Constant & Configuration Hygiene.

const (
	// CacheHitRateThresholdHit is the minimum cache ratio (80.0%) to be classified as [CACHE HIT].
	CacheHitRateThresholdHit = 80.0

	// CacheHitRateThresholdPartial is the minimum cache ratio (0.1%) to be classified as [CACHE PARTIAL].
	CacheHitRateThresholdPartial = 0.1
)
