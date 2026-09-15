package core

import (
	"encoding/json"
	"sync"
)

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
	dashboardMu       sync.Mutex
	dashboardRevision uint64
	dashboardJSON     []byte
	refreshMu         sync.Mutex
	mu                sync.RWMutex
	source            SessionSnapshotSource
	snapshot          Session
}

// NewQueryService initializes the query boundary from an adapter source.
func NewQueryService(source SessionSnapshotSource) *QueryService {
	return &QueryService{source: source, snapshot: cloneSession(source.Session())}
}

// Refresh updates the cached snapshot after the provider observes source changes.
func (service *QueryService) Refresh() (SessionDelta, error) {
	service.refreshMu.Lock()
	defer service.refreshMu.Unlock()
	delta, err := service.source.Refresh()
	if err != nil {
		return SessionDelta{}, err
	}
	service.mu.Lock()
	if service.snapshot.Revision != delta.Revision {
		service.snapshot = CloneSession(service.source.Session())
	}
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
	service.dashboardMu.Lock()
	defer service.dashboardMu.Unlock()
	session := service.Session()
	if service.dashboardJSON == nil || service.dashboardRevision != session.Revision {
		model := BuildDashboardReadModel(session)
		encoded, err := json.Marshal(model)
		if err != nil {
			return model
		}
		service.dashboardJSON, service.dashboardRevision = encoded, session.Revision
	}
	var model DashboardReadModel
	_ = json.Unmarshal(service.dashboardJSON, &model)
	return model
}

func cloneSession(session Session) Session {
	return CloneSession(session)
}

// CloneSession isolates all mutable domain data at a snapshot boundary.
func CloneSession(session Session) Session {
	cloned := session
	cloned.Steps = append([]Step(nil), session.Steps...)
	for i := range cloned.Steps {
		cloned.Steps[i].ConsumedStepIndexes = append([]int(nil), session.Steps[i].ConsumedStepIndexes...)
		cloned.Steps[i].Evidence = append([]Evidence(nil), session.Steps[i].Evidence...)
		cloned.Steps[i].ToolCalls = append([]ToolCall(nil), session.Steps[i].ToolCalls...)
		for j := range cloned.Steps[i].ToolCalls {
			cloned.Steps[i].ToolCalls[j].Args = cloneJSONMap(session.Steps[i].ToolCalls[j].Args)
		}
	}
	cloned.Generations = append([]Generation(nil), session.Generations...)
	for i := range cloned.Generations {
		cloned.Generations[i].Evidence = append([]Evidence(nil), session.Generations[i].Evidence...)
	}
	cloned.ContextSnapshots = append([]ContextSnapshot(nil), session.ContextSnapshots...)
	for i := range cloned.ContextSnapshots {
		cloned.ContextSnapshots[i].Evidence = append([]Evidence(nil), session.ContextSnapshots[i].Evidence...)
		cloned.ContextSnapshots[i].PersistedContextRecords = append([]PersistedContextRecord(nil), session.ContextSnapshots[i].PersistedContextRecords...)
		cloned.ContextSnapshots[i].NativeTools = append([]ToolSignature(nil), session.ContextSnapshots[i].NativeTools...)
		for j := range cloned.ContextSnapshots[i].NativeTools {
			cloned.ContextSnapshots[i].NativeTools[j].Required = append([]string(nil), session.ContextSnapshots[i].NativeTools[j].Required...)
		}
	}
	cloned.Models.Records = append([]ModelRecord(nil), session.Models.Records...)
	for index := range cloned.Models.Records {
		cloned.Models.Records[index].GenerationIDs = append([]string(nil), session.Models.Records[index].GenerationIDs...)
		cloned.Models.Records[index].Evidence = append([]Evidence(nil), session.Models.Records[index].Evidence...)
	}
	return cloned
}

func cloneJSONMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = cloneJSONValue(value)
	}
	return result
}

func cloneJSONValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return cloneJSONMap(value)
	case []any:
		result := make([]any, len(value))
		for i, item := range value {
			result[i] = cloneJSONValue(item)
		}
		return result
	case []string:
		return append([]string(nil), value...)
	case []byte:
		return append([]byte(nil), value...)
	default:
		return value
	}
}
