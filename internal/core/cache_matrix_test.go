package core_test

import (
	"testing"

	"heimdall/internal/core"
)

func TestCacheClassificationUsesDecodedUsageOnly(t *testing.T) {
	hit := core.PersistedUsageObservation{Available: true, HasTotalTokens: true, TotalTokens: 100, HasCachedTokens: true, CachedTokens: 80}
	miss := core.PersistedUsageObservation{Available: true, HasTotalTokens: true, TotalTokens: 100, HasCachedTokens: true, CachedTokens: 0}
	missing := core.PersistedUsageObservation{Available: true, HasTotalTokens: true, TotalTokens: 100}

	requireEqual(t, "HIT", core.ClassifyCacheStatus(mustRate(hit), hit.CachedTokens, hit.TotalTokens))
	requireEqual(t, "MISS", core.ClassifyCacheStatus(mustRate(miss), miss.CachedTokens, miss.TotalTokens))
	requireEqual(t, false, hasRate(missing))
}

func TestCacheClassificationRejectsInvalidDecodedRelationship(t *testing.T) {
	invalid := core.PersistedUsageObservation{Available: true, HasTotalTokens: true, TotalTokens: 10, HasCachedTokens: true, CachedTokens: 11}
	_, hasRate := invalid.CacheHitRate()

	requireEqual(t, false, hasRate)
}

func TestCacheStatusFromUsage_UsesOnlyValidPersistedPair(t *testing.T) {
	valid := core.PersistedUsageObservation{Available: true, HasTotalTokens: true, TotalTokens: 100, HasCachedTokens: true, CachedTokens: 80}
	missing := core.PersistedUsageObservation{Available: true, HasTotalTokens: true, TotalTokens: 100}

	requireEqual(t, "HIT", core.CacheStatusFromUsage(valid))
	requireEqual(t, "UNKNOWN", core.CacheStatusFromUsage(missing))
}

func mustRate(observation core.PersistedUsageObservation) float64 {
	rate, _ := observation.CacheHitRate()
	return rate
}

func hasRate(observation core.PersistedUsageObservation) bool {
	_, ok := observation.CacheHitRate()
	return ok
}
