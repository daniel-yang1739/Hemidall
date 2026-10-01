package antigravity

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/core"
)

const (
	fixtureSessionID     = "session-fixture"
	fixtureFirstStep     = 11
	fixtureSecondStep    = 12
	fixtureToolStep      = 13
	fixtureFileMode      = 0o600
	fixtureDirectoryMode = 0o700
	fixtureTimestamp     = "2026-09-01T01:02:03Z"
	secondRevision       = 2
)

func TestSessionStoreStepOutput_Pos_UsesPersistedArtifactContent(t *testing.T) {
	store, stepOutputsPath := newFixtureStoreWithStepOutputs(t, toolResultTranscriptLine(fixtureToolStep, "transcript content")+"\n")
	writeStepOutputFixture(t, stepOutputsPath, fixtureToolStep, stepOutputFileName, "persisted tool output")

	_, refreshErr := store.Refresh()
	step := store.Session().Steps[0]

	requireNoError(t, refreshErr)
	requireEqual(t, step.Content, "persisted tool output")
	requireEqual(t, step.ContentSource, core.SourceKindArtifacts)
	requireEqual(t, step.Evidence[len(step.Evidence)-1].Note, stepOutputEvidenceNote)
}

func TestSessionStoreStepOutput_Pos_DiscoversSiblingArtifactsForExplicitTranscript(t *testing.T) {
	root := t.TempDir()
	logsPath := filepath.Join(root, systemGeneratedDirectory, logsDirectoryName)
	stepOutputsPath := filepath.Join(root, systemGeneratedDirectory, stepsDirectoryName)
	requireNoError(t, os.MkdirAll(logsPath, fixtureDirectoryMode))
	requireNoError(t, os.MkdirAll(stepOutputsPath, fixtureDirectoryMode))
	transcriptPath := filepath.Join(logsPath, fullTranscriptFileName)
	writeFixtureFile(t, transcriptPath, toolResultTranscriptLine(fixtureToolStep, "transcript content")+"\n")
	writeStepOutputFixture(t, stepOutputsPath, fixtureToolStep, stepOutputFileName, "persisted tool output")
	store := newFixtureStore(t, transcriptPath)

	_, refreshErr := store.Refresh()
	step := store.Session().Steps[0]

	requireNoError(t, refreshErr)
	requireEqual(t, step.Content, "persisted tool output")
	requireEqual(t, step.ContentSource, core.SourceKindArtifacts)
}

func TestSessionStoreStepOutput_TN_PreservesTranscriptWithoutArtifact(t *testing.T) {
	store, _ := newFixtureStoreWithStepOutputs(t, toolResultTranscriptLine(fixtureToolStep, "transcript content")+"\n")

	_, refreshErr := store.Refresh()
	step := store.Session().Steps[0]

	requireNoError(t, refreshErr)
	requireEqual(t, step.Content, "transcript content")
	requireEqual(t, step.ContentSource, core.SourceKindTranscript)
}

func TestSessionStoreStepOutput_FP_IgnoresNonOutputArtifact(t *testing.T) {
	store, stepOutputsPath := newFixtureStoreWithStepOutputs(t, toolResultTranscriptLine(fixtureToolStep, "transcript content")+"\n")
	writeStepOutputFixture(t, stepOutputsPath, fixtureToolStep, "content.md", "unrelated artifact")

	_, refreshErr := store.Refresh()
	step := store.Session().Steps[0]

	requireNoError(t, refreshErr)
	requireEqual(t, step.Content, "transcript content")
	requireEqual(t, step.ContentSource, core.SourceKindTranscript)
}

func TestSessionStoreStepOutput_FN_RefreshesArtifactWithoutTranscriptChange(t *testing.T) {
	store, stepOutputsPath := newFixtureStoreWithStepOutputs(t, toolResultTranscriptLine(fixtureToolStep, "running transcript content")+"\n")
	outputPath := writeStepOutputFixture(t, stepOutputsPath, fixtureToolStep, stepOutputFileName, "initial output")
	firstDelta, firstRefreshErr := store.Refresh()
	writeFixtureFile(t, outputPath, "final background output")
	secondDelta, secondRefreshErr := store.Refresh()
	step := store.Session().Steps[0]

	requireNoError(t, firstRefreshErr)
	requireNoError(t, secondRefreshErr)
	requireEqual(t, secondDelta.Revision > firstDelta.Revision, true)
	requireEqual(t, secondDelta.ChangedStepIndexes[0], fixtureToolStep)
	requireEqual(t, step.Content, "final background output")
	requireEqual(t, step.ContentSource, core.SourceKindArtifacts)
}

func TestSessionStore_RefreshReadsOnlyAppendedTranscriptData(t *testing.T) {
	transcriptPath := writeFixtureTranscript(t, transcriptLine(fixtureFirstStep, "first")+"\n")
	store := newFixtureStore(t, transcriptPath)

	firstDelta, firstRefreshErr := store.Refresh()
	firstSession := store.Session()
	secondDelta, secondRefreshErr := store.Refresh()
	appendFixtureTranscript(t, transcriptPath, transcriptLine(fixtureSecondStep, "second")+"\n")
	thirdDelta, thirdRefreshErr := store.Refresh()
	thirdSession := store.Session()

	requireNoError(t, firstRefreshErr)
	requireEqual(t, firstDelta.Revision, uint64(1))
	requireEqual(t, firstDelta.ChangedStepIndexes[0], fixtureFirstStep)
	requireEqual(t, len(firstSession.Steps), 1)
	requireNoError(t, secondRefreshErr)
	requireEqual(t, secondDelta.Revision, uint64(1))
	requireEqual(t, len(secondDelta.ChangedStepIndexes), 0)
	requireNoError(t, thirdRefreshErr)
	requireEqual(t, thirdDelta.Revision, uint64(2))
	requireEqual(t, thirdDelta.ChangedStepIndexes[0], fixtureSecondStep)
	requireEqual(t, len(thirdSession.Steps), 2)
	requireEqual(t, thirdSession.Steps[0].Index, fixtureFirstStep)
	requireEqual(t, thirdSession.Steps[1].Index, fixtureSecondStep)
}

func TestSessionStore_RefreshWaitsForCompletedTranscriptLine(t *testing.T) {
	transcriptPath := writeFixtureTranscript(t, `{"step_index":11`)
	store := newFixtureStore(t, transcriptPath)

	firstDelta, firstRefreshErr := store.Refresh()
	firstSession := store.Session()
	appendFixtureTranscript(t, transcriptPath, `,"source":"USER","type":"USER_INPUT","status":"DONE","created_at":"2026-09-01T01:02:03Z","content":"first"}`+"\n")
	secondDelta, secondRefreshErr := store.Refresh()
	secondSession := store.Session()

	requireNoError(t, firstRefreshErr)
	requireEqual(t, len(firstDelta.ChangedStepIndexes), 0)
	requireEqual(t, len(firstSession.Steps), 0)
	requireNoError(t, secondRefreshErr)
	requireEqual(t, secondDelta.ChangedStepIndexes[0], fixtureFirstStep)
	requireEqual(t, len(secondSession.Steps), 1)
}

func TestSessionStore_RefreshClassifiesPlannerResponseAsCloudStep(t *testing.T) {
	transcriptPath := writeFixtureTranscript(t, `{"step_index":11,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"`+fixtureTimestamp+`","content":"response"}`+"\n")
	store := newFixtureStore(t, transcriptPath)

	_, refreshErr := store.Refresh()
	session := store.Session()

	requireNoError(t, refreshErr)
	requireEqual(t, session.Steps[0].Kind, string(core.StepTypeModelResponse))
	requireEqual(t, session.Steps[0].Scope, core.ScopeCloudInference)
}

func TestSessionStore_RefreshLinksLocalStepToTriggeringPlannerResponseToolCall(t *testing.T) {
	lines := `{"step_index":0,"source":"USER","type":"USER_INPUT","status":"DONE","created_at":"` + fixtureTimestamp + `","content":"run it"}` + "\n" +
		`{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"` + fixtureTimestamp + `","content":"calling tool","tool_calls":[{"name":"run_command","args":{"command":"ls"}}]}` + "\n" +
		`{"step_index":2,"source":"MODEL","type":"GENERIC","status":"DONE","created_at":"` + fixtureTimestamp + `","content":"total 0"}` + "\n"
	transcriptPath := writeFixtureTranscript(t, lines)
	store := newFixtureStore(t, transcriptPath)

	_, refreshErr := store.Refresh()
	session := store.Session()

	requireNoError(t, refreshErr)
	requireEqual(t, len(session.Steps), 3)
	requireEqual(t, session.Steps[1].Kind, string(core.StepTypeToolCall))
	requireEqual(t, session.Steps[1].Scope, core.ScopeCloudInference)
	requireEqual(t, session.Steps[2].Kind, string(core.StepTypeGeneric))
	requireEqual(t, session.Steps[2].Scope, core.ScopeLocalExecution)
	requireEqual(t, session.Steps[2].ParentStepIndex, 1)

	readModel := core.BuildDashboardReadModel(session)
	requireEqual(t, readModel.Inspections[2].TriggeredCloudStep, 1)
	requireEqual(t, readModel.Inspections[2].LocalAction, "run_command")
}

func TestNewSessionStore_RequiresTranscriptSource(t *testing.T) {
	store, err := NewSessionStore(agents.SessionRef{SessionID: fixtureSessionID}, nil)

	requireEqual(t, store == nil, true)
	requireError(t, err)
}

func TestSessionStore_RefreshRebuildsAfterTranscriptTruncation(t *testing.T) {
	transcriptPath := writeFixtureTranscript(t, transcriptLine(fixtureFirstStep, "first")+"\n"+transcriptLine(fixtureSecondStep, "second")+"\n")
	store := newFixtureStore(t, transcriptPath)
	_, initialRefreshErr := store.Refresh()
	replaceErr := os.WriteFile(transcriptPath, []byte(transcriptLine(fixtureFirstStep, "replacement")+"\n"), fixtureFileMode)
	resetDelta, resetRefreshErr := store.Refresh()
	resetSession := store.Session()

	requireNoError(t, initialRefreshErr)
	requireNoError(t, replaceErr)
	requireNoError(t, resetRefreshErr)
	requireEqual(t, resetDelta.Revision, uint64(secondRevision))
	requireEqual(t, len(resetSession.Steps), 1)
	requireEqual(t, resetSession.Steps[0].Content, "replacement")
}

func newFixtureStore(t *testing.T, transcriptPath string) *SessionStore {
	t.Helper()
	store, err := NewSessionStore(agents.SessionRef{
		AgentID:      agents.AgentID("antigravity"),
		SessionID:    fixtureSessionID,
		DiscoveredAt: time.Time{},
	}, []agents.SourceRef{{Kind: agents.SourceKindTranscript, Path: transcriptPath}})
	requireNoError(t, err)
	return store
}

func newFixtureStoreWithStepOutputs(t *testing.T, transcriptContent string) (*SessionStore, string) {
	t.Helper()
	root := t.TempDir()
	logsPath := filepath.Join(root, ".system_generated", "logs")
	stepOutputsPath := filepath.Join(root, ".system_generated", "steps")
	requireNoError(t, os.MkdirAll(logsPath, fixtureDirectoryMode))
	requireNoError(t, os.MkdirAll(stepOutputsPath, fixtureDirectoryMode))
	transcriptPath := filepath.Join(logsPath, "transcript_full.jsonl")
	writeFixtureFile(t, transcriptPath, transcriptContent)
	store, err := NewSessionStore(agents.SessionRef{
		AgentID: agents.AgentID("antigravity"), SessionID: fixtureSessionID,
	}, []agents.SourceRef{
		{Kind: agents.SourceKindTranscript, Path: transcriptPath},
		{Kind: agents.SourceKindArtifacts, Path: stepOutputsPath},
	})
	requireNoError(t, err)
	return store, stepOutputsPath
}

func writeStepOutputFixture(t *testing.T, root string, stepIndex int, name, content string) string {
	t.Helper()
	directory := filepath.Join(root, strconv.Itoa(stepIndex))
	requireNoError(t, os.MkdirAll(directory, fixtureDirectoryMode))
	path := filepath.Join(directory, name)
	writeFixtureFile(t, path, content)
	return path
}

func writeFixtureTranscript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	writeFixtureFile(t, path, content)
	return path
}

func appendFixtureTranscript(t *testing.T, path, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, fixtureFileMode)
	requireNoError(t, err)
	_, writeErr := file.WriteString(content)
	closeErr := file.Close()
	requireNoError(t, writeErr)
	requireNoError(t, closeErr)
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), fixtureFileMode)
	requireNoError(t, err)
}

func transcriptLine(stepIndex int, content string) string {
	return `{"step_index":` + strconv.Itoa(stepIndex) + `,"source":"USER","type":"USER_INPUT","status":"DONE","created_at":"` + fixtureTimestamp + `","content":"` + content + `"}`
}

func toolResultTranscriptLine(stepIndex int, content string) string {
	return `{"step_index":` + strconv.Itoa(stepIndex) + `,"source":"MODEL","type":"GENERIC","status":"DONE","created_at":"` + fixtureTimestamp + `","content":"` + content + `"}`
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
}

func requireEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
