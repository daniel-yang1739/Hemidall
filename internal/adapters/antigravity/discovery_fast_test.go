package antigravity

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	latestFixtureOlderSession   = "older-session"
	latestFixtureNewerSession   = "newer-session"
	fixtureFilePermissions      = 0o600
	fixtureDirectoryPermissions = 0o700
)

var (
	latestFixtureOlderTime = time.Date(2026, time.August, 30, 10, 0, 0, 0, time.UTC)
	latestFixtureNewerTime = latestFixtureOlderTime.Add(time.Hour)
)

func TestDiscoverLatestSession_Pos_OnlyReadsMostRecentCandidate(t *testing.T) {
	home := t.TempDir()
	createLatestSessionFixture(t, home, latestFixtureOlderSession, latestFixtureOlderTime)
	createLatestSessionFixture(t, home, latestFixtureNewerSession, latestFixtureNewerTime)

	latest, discoverErr := discoverLatestSessionAtHome(home)

	requireAntigravityNoError(t, discoverErr)
	requireAntigravityEqual(t, latestFixtureNewerSession, latest.SessionID)
	requireAntigravitySameInstant(t, latestFixtureNewerTime, latest.LastModified)
}

func TestDiscoverAllSessions_Boundary_DoesNotClaimUnknownStepCounts(t *testing.T) {
	home := t.TempDir()
	createLatestSessionFixture(t, home, latestFixtureOlderSession, latestFixtureOlderTime)
	createLatestSessionFixture(t, home, latestFixtureNewerSession, latestFixtureNewerTime)

	sessions, discoverErr := discoverAllSessionsAtHome(home)

	requireAntigravityNoError(t, discoverErr)
	requireAntigravityEqual(t, 2, len(sessions))
	requireAntigravityEqual(t, latestFixtureNewerSession, sessions[0].SessionID)
	requireAntigravityEqual(t, false, sessions[0].StepCountAvailable)
}

func createLatestSessionFixture(t *testing.T, home, sessionID string, modifiedAt time.Time) {
	t.Helper()
	databasePath := ConversationDatabasePath(home, sessionID)
	logPath, _ := TranscriptPaths(home, sessionID)
	requireAntigravityNoError(t, os.MkdirAll(filepath.Dir(databasePath), fixtureDirectoryPermissions))
	requireAntigravityNoError(t, os.MkdirAll(filepath.Dir(logPath), fixtureDirectoryPermissions))
	requireAntigravityNoError(t, os.WriteFile(databasePath, nil, fixtureFilePermissions))
	requireAntigravityNoError(t, os.Chtimes(databasePath, modifiedAt, modifiedAt))
	requireAntigravityNoError(t, os.WriteFile(logPath, []byte(`{"type":"USER_INPUT","content":"<USER_REQUEST>fixture</USER_REQUEST>"}`), fixtureFilePermissions))
}
