package core

import "testing"

const queryTestStepIndex = 7

func TestQueryService_RefreshesOnlyThroughSourceContract(t *testing.T) {
	source := &testSessionSource{session: Session{Ref: SessionRef{SessionID: "first"}, Steps: []Step{{Index: queryTestStepIndex, Content: "before"}}}}
	service := NewQueryService(source)
	source.session = Session{Ref: SessionRef{SessionID: "second"}, Revision: 1, Steps: []Step{{Index: queryTestStepIndex, Content: "after"}}}
	source.delta = SessionDelta{SessionID: "second", Revision: 1, ChangedStepIndexes: []int{queryTestStepIndex}}

	delta, refreshErr := service.Refresh()
	step, found := service.StepByIndex(queryTestStepIndex)

	requireQueryNoError(t, refreshErr)
	requireQueryEqual(t, delta.SessionID, "second")
	requireQueryEqual(t, found, true)
	requireQueryEqual(t, step.Content, "after")
	requireQueryEqual(t, source.refreshCount, 1)
}

func TestQueryService_SessionReturnsIndependentSlices(t *testing.T) {
	source := &testSessionSource{session: Session{Steps: []Step{{Index: queryTestStepIndex}}}}
	service := NewQueryService(source)

	first := service.Session()
	first.Steps = nil
	second := service.Session()

	requireQueryEqual(t, len(second.Steps), 1)
}

func TestQueryService_ModelByIDReturnsDynamicModelTableRow(t *testing.T) {
	source := &testSessionSource{session: Session{Models: ModelCatalog{Records: []ModelRecord{{ModelID: "gemini-3.7-flash", GenerationIDs: []string{"7"}}}}}}
	service := NewQueryService(source)

	model, found := service.ModelByID("gemini-3.7-flash")

	requireQueryEqual(t, found, true)
	requireQueryEqual(t, model.GenerationIDs[0], "7")
}

type testSessionSource struct {
	session      Session
	delta        SessionDelta
	refreshCount int
}

func (source *testSessionSource) Session() Session {
	return source.session
}

func (source *testSessionSource) Refresh() (SessionDelta, error) {
	source.refreshCount++
	return source.delta, nil
}

func requireQueryNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireQueryEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
