package antigravity

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const (
	telemetryGenerationIndex       = 1
	telemetryInputBoundaryStep     = 41
	telemetryGeneratedStep         = telemetryInputBoundaryStep + 1
	telemetryClaudeModelID         = "claude-sonnet-4-6"
	telemetryFirstGenerationIndex  = initialGenerationMetadataIndex + 1
	telemetryInputBoundaryString   = "41"
	telemetryExecutorMetadataIndex = 1
	telemetryExecutorExecutionID   = "bc5f926b-22c9-4c9a-9d0b-fab5c69ab9ea"
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

func TestSQLiteTelemetry_Pos_ReadsFirstMetadataRowAndPersistsClaudeModel(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", initialGenerationMetadataIndex+1, persistedModelFixture(telemetryClaudeModelID, persistedClaudeEnum, "0"))

	reader := NewSQLiteTelemetryReader(databasePath)
	pollErr := reader.PollLatest()
	metadata := reader.TakeTelemetryForGeneratedStep(initialGenerationMetadataIndex + 2)

	requireAntigravityNoError(t, pollErr)
	requireAntigravityPresent(t, metadata)
	requireAntigravityEqual(t, telemetryClaudeModelID, metadata.ModelName)
}

func TestSQLiteTelemetry_Pos_InfersMissingModelNameFromUniqueSessionEnum(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", initialGenerationMetadataIndex+1, persistedModelFixture(persistedClaudeModelID, persistedClaudeEnum, "0"))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", initialGenerationMetadataIndex+2, persistedModelFixture("", persistedClaudeEnum, "1"))

	reader := NewSQLiteTelemetryReader(databasePath)
	pollErr := reader.PollLatest()
	metadata := reader.TakeTelemetryForGeneratedStep(initialGenerationMetadataIndex + 3)

	requireAntigravityNoError(t, pollErr)
	requireAntigravityPresent(t, metadata)
	requireAntigravityEqual(t, telemetryClaudeModelID, metadata.ModelName)
}

func TestSQLiteTelemetry_Neg_DoesNotInferAmbiguousSessionEnum(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", initialGenerationMetadataIndex+1, persistedModelFixture(persistedGeminiModelID, persistedGeminiEnum, "0"))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", initialGenerationMetadataIndex+2, persistedModelFixture(persistedGPTModelID, persistedGeminiEnum, "1"))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", initialGenerationMetadataIndex+3, persistedModelFixture("", persistedGeminiEnum, "2"))

	reader := NewSQLiteTelemetryReader(databasePath)
	pollErr := reader.PollLatest()
	metadata := reader.TakeTelemetryForGeneratedStep(initialGenerationMetadataIndex + 4)

	requireAntigravityNoError(t, pollErr)
	requireAntigravityPresent(t, metadata)
	requireAntigravityEqual(t, "", metadata.ModelName)
}

func TestSQLiteTelemetry_Pos_UsesExecutorMetadataForMissingDirectModelName(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "CREATE TABLE executor_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", telemetryFirstGenerationIndex, persistedTelemetryExecutionFixture())
	requireAntigravityExec(t, database, "INSERT INTO executor_metadata (idx, data) VALUES (?, ?)", telemetryExecutorMetadataIndex, persistedTelemetryExecutorMetadataFixture())
	requireAntigravityNoError(t, database.Close())

	reader := NewSQLiteTelemetryReader(databasePath)
	pollErr := reader.PollLatest()
	metadata := reader.TakeTelemetryForGeneratedStep(telemetryGeneratedStep)

	requireAntigravityNoError(t, pollErr)
	requireAntigravityPresent(t, metadata)
	requireAntigravityEqual(t, persistedGeminiModelID, metadata.ModelName)
}

func persistedTelemetryExecutionFixture() []byte {
	return protoBytes(persistedMetadataRootFieldNumber, protoMessage(
		protoBytes(persistedMetadataAttributesFieldNumber, persistedMetadataMapEntry(persistedLastExecutionIDKey, telemetryExecutorExecutionID)),
		protoBytes(persistedMetadataAttributesFieldNumber, persistedMetadataMapEntry(persistedLastStepIndexKey, telemetryInputBoundaryString)),
	))
}

func persistedTelemetryExecutorMetadataFixture() []byte {
	return []byte(persistedLastExecutionIDKey + "\x12\x24" + telemetryExecutorExecutionID + "\x00" + persistedGeminiModelID)
}
