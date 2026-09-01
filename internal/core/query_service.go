package core

import "sync"

// SessionSnapshotSource is implemented by provider adapters. Core owns the
// contract, so it never imports a concrete adapter package.
type SessionSnapshotSource interface {
	Session() Session
	Refresh() (SessionDelta, error)
}

// SessionQuery is the only session-data contract available to analysis and UI.
type SessionQuery interface {
	Session() Session
	StepByIndex(int) (Step, bool)
	GenerationByID(string) (Generation, bool)
	ModelByID(string) (ModelRecord, bool)
	Models() ModelCatalog
}

// QueryService is the application-facing boundary over one active session.
// It caches a provider snapshot and exposes only source-neutral domain objects.
type QueryService struct {
	mu       sync.RWMutex
	source   SessionSnapshotSource
	snapshot Session
}

// NewQueryService initializes the query boundary from an adapter source.
func NewQueryService(source SessionSnapshotSource) *QueryService {
	return &QueryService{source: source, snapshot: cloneSession(source.Session())}
}

// Refresh updates the cached snapshot after the provider observes source changes.
func (service *QueryService) Refresh() (SessionDelta, error) {
	delta, err := service.source.Refresh()
	if err != nil {
		return SessionDelta{}, err
	}
	service.mu.Lock()
	service.snapshot = cloneSession(service.source.Session())
	service.mu.Unlock()
	return delta, nil
}

// Session returns an immutable session read model.
func (service *QueryService) Session() Session {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return cloneSession(service.snapshot)
}

// StepByIndex returns one domain Step from the cached query snapshot.
func (service *QueryService) StepByIndex(stepIndex int) (Step, bool) {
	session := service.Session()
	for _, step := range session.Steps {
		if step.Index == stepIndex {
			return step, true
		}
	}
	return Step{}, false
}

// GenerationByID returns one domain Generation from the cached query snapshot.
func (service *QueryService) GenerationByID(generationID string) (Generation, bool) {
	session := service.Session()
	for _, generation := range session.Generations {
		if generation.ID == generationID {
			return generation, true
		}
	}
	return Generation{}, false
}

// ModelByID returns one dynamically discovered model table row.
func (service *QueryService) ModelByID(modelID string) (ModelRecord, bool) {
	session := service.Session()
	for _, model := range session.Models.Records {
		if model.ModelID == modelID {
			return model, true
		}
	}
	return ModelRecord{}, false
}

// Models returns the exact model catalog from the cached query snapshot.
func (service *QueryService) Models() ModelCatalog {
	return service.Session().Models
}

// DashboardReadModel returns the cached, source-neutral projection used by the
// Dashboard. Expensive context classification happens before the UI renders.
func (service *QueryService) DashboardReadModel() DashboardReadModel {
	return BuildDashboardReadModel(service.Session())
}

func cloneSession(session Session) Session {
	cloned := session
	cloned.Steps = append([]Step(nil), session.Steps...)
	cloned.Generations = append([]Generation(nil), session.Generations...)
	cloned.ContextSnapshots = append([]ContextSnapshot(nil), session.ContextSnapshots...)
	cloned.Models.Records = append([]ModelRecord(nil), session.Models.Records...)
	for index := range cloned.Models.Records {
		cloned.Models.Records[index].GenerationIDs = append([]string(nil), session.Models.Records[index].GenerationIDs...)
		cloned.Models.Records[index].Evidence = append([]Evidence(nil), session.Models.Records[index].Evidence...)
	}
	return cloned
}
