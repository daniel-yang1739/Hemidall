package conversation

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"heimdall/internal/agent_adapters"
)

const (
	fixtureGenerationIndex  = 7
	fixtureInputBoundary    = 41
	fixtureGeneratedStep    = fixtureInputBoundary + generatedStepOffset
	fixtureObservedContext  = 120_000
	fixtureContextLimit     = 160_000
	fixtureUncachedInput    = 10_000
	fixtureCachedInput      = 110_000
	fixtureTotalOutput      = 2_750
	fixtureThinkingOutput   = 1_190
	fixtureContentOutput    = 1_560
	fixtureModelCode        = 1_298
	fixtureModelID          = "gemini-3.7-flash"
	fixtureProtobufShift    = 3
	fixtureLengthWireType   = 2
	fixtureVarintWireType   = 0
	fixtureVarintThreshold  = 0x80
	fixtureVarintShift      = 7
	fixtureTotalOutputField = 3
	fixtureThinkingField    = 9
	fixtureContentField     = 10
	fixtureExecutionID      = "12345678-1234-1234-1234-123456789abc"
)

func TestParser_ParseDatabase_DecodesGenerationUsageAndDirectModel(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireNoError(t, openErr)
	defer database.Close()
	requireExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", fixtureGenerationIndex, generationFixture())

	generations, diagnostics, parseErr := (Parser{}).ParseDatabase(databasePath, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})

	requireNoError(t, parseErr)
	requireEqual(t, len(diagnostics), 0)
	requireEqual(t, len(generations), 1)
	requireEqual(t, generations[0].StepIndex, fixtureGeneratedStep)
	requireEqual(t, generations[0].ModelID, fixtureModelID)
	requireEqual(t, generations[0].Usage.ObservedContextTokens, fixtureObservedContext)
	requireEqual(t, generations[0].Usage.ContextLimit, fixtureContextLimit)
	requireEqual(t, generations[0].Usage.UncachedInputTokens, fixtureUncachedInput)
	requireEqual(t, generations[0].Usage.CachedInputTokens, fixtureCachedInput)
	requireEqual(t, generations[0].Usage.HasUncachedInputTokens, true)
	requireEqual(t, generations[0].Usage.HasCachedInputTokens, true)
	requireEqual(t, generations[0].Usage.TotalOutputTokens, fixtureTotalOutput)
	requireEqual(t, generations[0].Usage.ThinkingOutputTokens, fixtureThinkingOutput)
	requireEqual(t, generations[0].Usage.OutputContentTokens, fixtureContentOutput)
	requireEqual(t, generations[0].Usage.HasTotalOutputTokens, true)
	requireEqual(t, generations[0].Usage.HasThinkingOutputTokens, true)
	requireEqual(t, generations[0].Usage.HasOutputContentTokens, true)
}

func TestParser_ParseDatabase_ResolvesModelFromExecutorMetadata(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireNoError(t, openErr)
	defer database.Close()
	requireExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireExec(t, database, "CREATE TABLE executor_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", fixtureGenerationIndex, generationFixtureWithExecutionID())
	requireExec(t, database, "INSERT INTO executor_metadata (idx, data) VALUES (?, ?)", fixtureGenerationIndex, []byte(fixtureExecutionID+"\x00"+fixtureModelID))

	generations, diagnostics, parseErr := (Parser{}).ParseDatabase(databasePath, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})

	requireNoError(t, parseErr)
	requireEqual(t, len(diagnostics), 0)
	requireEqual(t, generations[0].ModelID, fixtureModelID)
	requireEqual(t, generations[0].Evidence[1].Level, agents.EvidenceArtifactEquality)
}

func TestParser_ParseDatabase_DoesNotTreatUsageFieldOneAsUncachedInput(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireNoError(t, openErr)
	defer database.Close()
	requireExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", fixtureGenerationIndex, generationFixtureWithModelCodeOnly())

	generations, diagnostics, parseErr := (Parser{}).ParseDatabase(databasePath, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})

	requireNoError(t, parseErr)
	requireEqual(t, len(diagnostics), 0)
	requireEqual(t, len(generations), 1)
	requireEqual(t, generations[0].Usage.HasUncachedInputTokens, false)
	requireEqual(t, generations[0].Usage.UncachedInputTokens, 0)
	requireEqual(t, generations[0].Usage.HasCachedInputTokens, true)
	requireEqual(t, generations[0].Usage.CachedInputTokens, fixtureCachedInput)
}

func TestParser_ParseDatabase_RecognizesCompleteColdUsage(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireNoError(t, openErr)
	defer database.Close()
	requireExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", fixtureGenerationIndex, generationFixtureWithUncachedInputOnly())

	generations, diagnostics, parseErr := (Parser{}).ParseDatabase(databasePath, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})

	requireNoError(t, parseErr)
	requireEqual(t, len(diagnostics), 0)
	requireEqual(t, len(generations), 1)
	requireEqual(t, generations[0].Usage.HasUncachedInputTokens, true)
	requireEqual(t, generations[0].Usage.UncachedInputTokens, fixtureUncachedInput)
	requireEqual(t, generations[0].Usage.HasCachedInputTokens, false)
	requireEqual(t, generations[0].Usage.CachedInputTokens, 0)
}

func generationFixture() []byte {
	metadata := concatFields(
		bytesField(modelFieldNumber, []byte(fixtureModelID)),
		bytesField(attributesFieldNumber, mapEntry(lastStepIndexKey, "41")),
		bytesField(contextUsageFieldNumber, bytesField(contextUsageDetailFieldNumber, concatFields(varintField(contextTokenFieldNumber, fixtureObservedContext), varintField(contextLimitFieldNumber, fixtureContextLimit)))),
		bytesField(inputUsageFieldNumber, concatFields(
			varintField(uncachedInputFieldNumber, fixtureUncachedInput),
			varintField(cachedInputFieldNumber, fixtureCachedInput),
			varintField(fixtureTotalOutputField, fixtureTotalOutput),
			varintField(fixtureThinkingField, fixtureThinkingOutput),
			varintField(fixtureContentField, fixtureContentOutput),
		)),
	)
	return bytesField(rootFieldNumber, metadata)
}

func generationFixtureWithExecutionID() []byte {
	metadata := concatFields(
		bytesField(attributesFieldNumber, mapEntry(lastStepIndexKey, "41")),
		bytesField(attributesFieldNumber, mapEntry(lastExecutionIDKey, fixtureExecutionID)),
		bytesField(inputUsageFieldNumber, concatFields(varintField(uncachedInputFieldNumber, fixtureUncachedInput), varintField(cachedInputFieldNumber, fixtureCachedInput))),
	)
	return bytesField(rootFieldNumber, metadata)
}

func generationFixtureWithModelCodeOnly() []byte {
	metadata := concatFields(
		bytesField(inputUsageFieldNumber, concatFields(varintField(contextTokenFieldNumber, fixtureModelCode), varintField(cachedInputFieldNumber, fixtureCachedInput))),
	)
	return bytesField(rootFieldNumber, metadata)
}

func generationFixtureWithUncachedInputOnly() []byte {
	metadata := concatFields(
		bytesField(inputUsageFieldNumber, varintField(uncachedInputFieldNumber, fixtureUncachedInput)),
	)
	return bytesField(rootFieldNumber, metadata)
}

func mapEntry(key, value string) []byte {
	return concatFields(bytesField(mapKeyFieldNumber, []byte(key)), bytesField(mapValueFieldNumber, []byte(value)))
}

func bytesField(number int, value []byte) []byte {
	return append(append(varint(uint64(number<<fixtureProtobufShift|fixtureLengthWireType)), varint(uint64(len(value)))...), value...)
}

func varintField(number int, value int) []byte {
	return append(varint(uint64(number<<fixtureProtobufShift|fixtureVarintWireType)), varint(uint64(value))...)
}

func concatFields(fields ...[]byte) []byte {
	combined := make([]byte, 0)
	for _, field := range fields {
		combined = append(combined, field...)
	}
	return combined
}

func varint(value uint64) []byte {
	encoded := make([]byte, 0)
	for value >= fixtureVarintThreshold {
		encoded = append(encoded, byte(value)|fixtureVarintThreshold)
		value >>= fixtureVarintShift
	}
	return append(encoded, byte(value))
}

func requireExec(t *testing.T, database *sql.DB, query string, arguments ...any) {
	t.Helper()
	_, err := database.Exec(query, arguments...)
	requireNoError(t, err)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
