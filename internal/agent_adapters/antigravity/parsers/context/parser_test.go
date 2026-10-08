package context

import (
	"testing"

	"heimdall/internal/agent_adapters"
)

const (
	fixtureGenerationIndex = 9
	fixtureFieldShift      = 3
	fixtureLengthWire      = 2
	fixtureVarintWire      = 0
	fixtureVarintThreshold = 0x80
	fixtureVarintShift     = 7

	// Proto field numbers according to cortex.proto
	protoRootPayloadField     = 1
	protoSystemPromptField    = 1
	protoMessagePromptsField  = 2
	protoMessageRoleField     = 2
	protoMessageContentField  = 3
	protoToolsField           = 8
	protoToolNameField        = 1
	protoToolDescriptionField = 2
)

func TestParseSnapshot_DecodesSystemRecordsAndTools(t *testing.T) {
	data := bytesField(protoRootPayloadField, concatenateFields(
		bytesField(protoSystemPromptField, []byte("<identity>agent</identity><mcp_servers>chrome-devtools</mcp_servers><user_rules>rules</user_rules>")),
		bytesField(protoMessagePromptsField, concatenateFields(varintField(protoMessageRoleField, 2), bytesField(protoMessageContentField, []byte("<CONTEXT_SUMMARY>summary")))),
		bytesField(protoToolsField, concatenateFields(bytesField(protoToolNameField, []byte("read_file")), bytesField(protoToolDescriptionField, []byte("read a file")))),
	))

	snapshot, parseErr := ParseSnapshot(fixtureGenerationIndex, data, agents.SourceRef{Kind: agents.SourceKindConversation, Path: "fixture.db"})

	requireContextNoError(t, parseErr)
	requireContextEqual(t, snapshot.IdentityPrompt, "agent")
	requireContextEqual(t, snapshot.ConstitutionDoc, "rules")
	requireContextEqual(t, snapshot.MCPSection, "chrome-devtools")
	requireContextEqual(t, len(snapshot.PersistedContextRecords), 1)
	requireContextEqual(t, snapshot.PersistedContextRecords[0].IsCompactedCheckpoint, true)
	requireContextEqual(t, snapshot.PersistedContextRecords[0].RoleName, "ASSISTANT")
	requireContextEqual(t, snapshot.NativeTools[0].Name, "read_file")
}

func TestParseSnapshot_RejectsMissingSystemPrompt(t *testing.T) {
	data := bytesField(protoRootPayloadField, bytesField(protoMessagePromptsField, []byte("record")))

	_, parseErr := ParseSnapshot(fixtureGenerationIndex, data, agents.SourceRef{})

	requireContextError(t, parseErr)
}

func bytesField(number int, value []byte) []byte {
	return append(append(encodeVarint(uint64(number<<fixtureFieldShift|fixtureLengthWire)), encodeVarint(uint64(len(value)))...), value...)
}

func varintField(number, value int) []byte {
	return append(encodeVarint(uint64(number<<fixtureFieldShift|fixtureVarintWire)), encodeVarint(uint64(value))...)
}

func concatenateFields(fields ...[]byte) []byte {
	combined := make([]byte, 0)
	for _, field := range fields {
		combined = append(combined, field...)
	}
	return combined
}

func encodeVarint(value uint64) []byte {
	encoded := make([]byte, 0)
	for value >= fixtureVarintThreshold {
		encoded = append(encoded, byte(value)|fixtureVarintThreshold)
		value >>= fixtureVarintShift
	}
	return append(encoded, byte(value))
}

func requireContextNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireContextError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
}

func requireContextEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
