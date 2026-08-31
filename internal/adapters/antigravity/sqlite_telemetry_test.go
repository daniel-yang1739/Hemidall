package antigravity

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const (
	telemetryGenerationIndex   = 1
	telemetryInputBoundaryStep = 41
	telemetryGeneratedStep     = telemetryInputBoundaryStep + 1
)

func TestSQLiteTelemetry_Pos_MapsInputBoundaryToFollowingGeneratedStep(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", telemetryGenerationIndex, []byte("last_step_index\x12\x0241"))

	reader := NewSQLiteTelemetryReader(databasePath)
	pollErr := reader.PollLatest()

	requireAntigravityNoError(t, pollErr)
	requireAntigravityPresent(t, reader.TakeTelemetryForGeneratedStep(telemetryGeneratedStep))
	requireAntigravityAbsent(t, reader.TakeTelemetryForGeneratedStep(telemetryGeneratedStep))
	requireAntigravityAbsent(t, reader.TakeTelemetryForGeneratedStep(telemetryInputBoundaryStep))
}
