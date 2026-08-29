package core

import "time"

// Cross-File Shared Domain Constants
// Strictly adheres to AGENTS.md Article V: Constant & Configuration Hygiene.

const (
	// TokensPerMillion is the standard divisor for per-million token pricing calculations.
	TokensPerMillion = 1000000.0

	// DefaultUSDtoTWDExchangeRate is the baseline fallback USD to TWD fiat currency exchange rate.
	DefaultUSDtoTWDExchangeRate = 32.0

	// DefaultDailyQuotaRPDPro is the standard Google AI Pro tier daily requests quota limit (RPD).
	DefaultDailyQuotaRPDPro = 5000

	// CacheHitRateThresholdHit is the minimum cache ratio (80.0%) to be classified as [CACHE HIT].
	CacheHitRateThresholdHit = 80.0

	// CacheHitRateThresholdPartial is the minimum cache ratio (0.1%) to be classified as [CACHE PARTIAL].
	CacheHitRateThresholdPartial = 0.1

	// CompactionHighWatermarkRatio represents the fraction of context usage (default 0.95 or 0.5)
	// that triggers dual-watermark context compaction.
	CompactionHighWatermarkRatio = 0.95

	// DefaultFallbackSystemTokens is the safe fallback token count for static system instructions
	// when genuine session transcripts or filesystem access are unavailable.
	DefaultFallbackSystemTokens = 3806

	// DefaultFallbackToolsDefTokens is the safe fallback token count for MCP tool schema definitions
	// when live tool schema extraction is unavailable.
	DefaultFallbackToolsDefTokens = 1377

	// DefaultFallbackAgentWindow is the default active agent context window limit (256k)
	// used when neither host settings nor model registry specifies a custom limit.
	DefaultFallbackAgentWindow = 256000

	// DefaultSessionRetentionMaxAge is the standard retention period before archiving inactive sessions.
	DefaultSessionRetentionMaxAge = 24 * time.Hour
)
