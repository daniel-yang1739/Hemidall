package antigravity

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

const (
	executorAuditRecordIndex  = 52
	executorAuditExecutionID  = "bc5f926b-22c9-4c9a-9d0b-fab5c69ab9ea"
	executorAuditTrajectoryID = "5eaed723-5d73-4605-a9b3-e0f99431df2f"
	executorAuditSessionID    = "-3750763034362895579"
)

func TestReadExecutorMetadataAudit_Pos_DecodesObservedIdentityFields(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE executor_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO executor_metadata (idx, data) VALUES (?, ?)", executorAuditRecordIndex, executorMetadataAuditFixture(executorAuditExecutionID, persistedGeminiModelID))
	requireAntigravityNoError(t, database.Close())

	audit, auditErr := ReadExecutorMetadataAudit(databasePath)
	printableStrings, stringsErr := ReadExecutorMetadataPrintableStrings(databasePath, executorAuditRecordIndex)

	requireAntigravityNoError(t, auditErr)
	requireAntigravityNoError(t, stringsErr)
	requireAntigravityEqual(t, 1, len(audit.Rows))
	requireAntigravityEqual(t, executorAuditRecordIndex, audit.Rows[0].Index)
	requireAntigravityEqual(t, 2, len(audit.Rows[0].ObservedExecutionUUIDs))
	requireAntigravityEqual(t, executorAuditExecutionID, audit.Rows[0].ObservedExecutionUUIDs[0])
	requireAntigravityEqual(t, executorAuditTrajectoryID, audit.Rows[0].ObservedExecutionUUIDs[1])
	requireAntigravityEqual(t, persistedGeminiModelID, audit.Rows[0].ModelName)
	requireExecutorAuditContains(t, printableStrings, persistedGeminiModelID)
}

func executorMetadataAuditFixture(executionID, modelID string) []byte {
	return []byte(persistedLastExecutionIDKey + "\x12\x24" + executionID + "\x00trajectory_id\x12\x24" + executorAuditTrajectoryID + "\x00sessionID\x12\x14" + executorAuditSessionID + "\x00" + modelID)
}

func requireExecutorAuditContains(t *testing.T, values []string, expected string) {
	t.Helper()
	if !strings.Contains(strings.Join(values, "\n"), expected) {
		t.Fatalf("expected values to contain %q, got %v", expected, values)
	}
}
