package antigravity

import (
	"database/sql"
	"heimdall/internal/core"
	"os"
	"path/filepath"
	"testing"
)

var (
	recoveryFirstUsage   = []byte{0x0a, 0x04, 0x22, 0x02, 0x10, 0x0a}
	recoveryUpdatedUsage = []byte{0x0a, 0x04, 0x22, 0x02, 0x10, 0x14}
)

func TestSessionStoreRollsBackTranscriptCursorAfterDatabaseFailure(t *testing.T) {
	store, database, path := recoveryStore(t)
	_, err := store.Refresh()
	requireNoError(t, err)
	initial := store.Session()
	appendFixtureTranscript(t, path, transcriptLine(fixtureSecondStep, "new")+"\n")
	execRecoverySQL(t, database, "ALTER TABLE gen_metadata RENAME TO unavailable")
	_, failed := store.Refresh()
	requireError(t, failed)
	requireEqual(t, len(store.Session().Steps), 1)
	requireEqual(t, store.Session().Revision, initial.Revision)
	execRecoverySQL(t, database, "ALTER TABLE unavailable RENAME TO gen_metadata")
	_, recovered := store.Refresh()
	requireNoError(t, recovered)
	requireEqual(t, len(store.Session().Steps), 2)
	requireEqual(t, store.Session().Steps[1].Content, "new")
}

func TestSessionStoreRefreshesExistingGenerationWithoutNewTranscript(t *testing.T) {
	store, database, _ := recoveryStore(t)
	_, err := store.Refresh()
	requireNoError(t, err)
	initial := store.Session()
	execRecoverySQL(t, database, "UPDATE gen_metadata SET data = ? WHERE idx = 1", recoveryUpdatedUsage)
	delta, refreshErr := store.Refresh()
	requireNoError(t, refreshErr)
	requireEqual(t, len(store.Session().Generations), 1)
	requireEqual(t, store.Session().Generations[0].Usage.UncachedInputTokens, 20)
	if delta.Revision <= initial.Revision {
		t.Fatal("updated generation did not invalidate analysis")
	}
	_, unchangedErr := store.Refresh()
	requireNoError(t, unchangedErr)
	requireEqual(t, len(store.Session().Generations), 1)
}

func TestSessionStoreReplacesSameSizeTranscriptByIdentity(t *testing.T) {
	path := writeFixtureTranscript(t, transcriptLine(fixtureFirstStep, "first")+"\n")
	store := newFixtureStore(t, path)
	_, err := store.Refresh()
	requireNoError(t, err)
	writeFixtureFile(t, path+".replacement", transcriptLine(fixtureSecondStep, "other")+"\n")
	requireNoError(t, os.Rename(path+".replacement", path))
	delta, err := store.Refresh()
	requireNoError(t, err)
	requireEqual(t, delta.Reset, true)
	requireEqual(t, len(store.Session().Steps), 1)
	requireEqual(t, store.Session().Steps[0].Index, fixtureSecondStep)
}

func recoveryStore(t *testing.T) (*SessionStore, *sql.DB, string) {
	t.Helper()
	path := writeFixtureTranscript(t, transcriptLine(fixtureFirstStep, "first")+"\n")
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, err := sql.Open("sqlite", databasePath)
	requireNoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	execRecoverySQL(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	execRecoverySQL(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (1, ?)", recoveryFirstUsage)
	store, err := NewSessionStore(core.SessionRef{SessionID: fixtureSessionID}, []core.SourceRef{{Kind: core.SourceKindTranscript, Path: path}, {Kind: core.SourceKindConversation, Path: databasePath}})
	requireNoError(t, err)
	return store, database, path
}

func execRecoverySQL(t *testing.T, database *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := database.Exec(query, args...)
	requireNoError(t, err)
}
