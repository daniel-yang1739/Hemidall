package core

import (
	"encoding/json"
	"sort"
	"sync"
)

// AnalysisService derives reusable analysis exclusively from SessionQuery.
type AnalysisService struct {
	query          SessionQuery
	mu             sync.Mutex
	cachedID       string
	cachedRevision uint64
	cachedJSON     []byte
}

func NewAnalysisService(query SessionQuery) *AnalysisService { return &AnalysisService{query: query} }

// SessionMetrics is the compatibility aggregate entry point. Both projections
// use generationMetrics so several generations at one step remain billable.
func (service *AnalysisService) SessionMetrics() SessionAggregateMetrics {
	metrics := generationMetrics(service.query.Session().Generations)
	sort.Slice(metrics.ModelStats, func(i, j int) bool { return metrics.ModelStats[i].ModelName < metrics.ModelStats[j].ModelName })
	return metrics
}

// Report caches tokenization and ranking by session revision. Decoding a fresh
// copy keeps callers from mutating the cached report's nested slices.
func (service *AnalysisService) Report() SessionAnalysisReport {
	service.mu.Lock()
	defer service.mu.Unlock()
	session := service.query.Session()
	if service.cachedJSON == nil || service.cachedID != session.Ref.SessionID || service.cachedRevision != session.Revision {
		report := BuildSessionAnalysisReport(session)
		service.cachedJSON, _ = json.Marshal(report)
		service.cachedID, service.cachedRevision = session.Ref.SessionID, session.Revision
	}
	var report SessionAnalysisReport
	_ = json.Unmarshal(service.cachedJSON, &report)
	return report
}
