package antigravity

import "testing"

const (
	persistedGeminiModelID = "gemini-3.7-flash"
	persistedClaudeModelID = "claude-sonnet-4-6"
	persistedGPTModelID    = "gpt-5.6-terra"
	persistedGeminiEnum    = "MODEL_PLACEHOLDER_M298"
	persistedClaudeEnum    = "MODEL_PLACEHOLDER_M35"
	persistedBoundary      = "42"
)

func TestPersistedModelName_Pos_ExtractsGeminiModelFromStructuredEnvelope(t *testing.T) {
	metadata, parseErr := ParsePersistedGenerationMetadata(1, persistedModelFixture(persistedGeminiModelID, persistedGeminiEnum, persistedBoundary))

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, persistedGeminiModelID, metadata.ModelName)
	requireAntigravityEqual(t, persistedGeminiEnum, metadata.ModelEnum)
	requireAntigravityEqual(t, true, metadata.HasInputBoundary)
	requireAntigravityEqual(t, 42, metadata.InputBoundaryStepIndex)
}

func TestPersistedModelName_Pos_ExtractsClaudeModelFromStructuredEnvelope(t *testing.T) {
	metadata, parseErr := ParsePersistedGenerationMetadata(1, persistedModelFixture(persistedClaudeModelID, persistedClaudeEnum, persistedBoundary))

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, persistedClaudeModelID, metadata.ModelName)
}

func TestPersistedModelName_Pos_ExtractsGPTModelFromStructuredEnvelope(t *testing.T) {
	metadata, parseErr := ParsePersistedGenerationMetadata(1, persistedModelFixture(persistedGPTModelID, persistedGeminiEnum, persistedBoundary))

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, persistedGPTModelID, metadata.ModelName)
}

func TestPersistedModelName_Boundary_RetainsEnumWhenDirectModelIDIsMissing(t *testing.T) {
	metadata, parseErr := ParsePersistedGenerationMetadata(1, persistedModelFixture("", persistedClaudeEnum, persistedBoundary))

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, "", metadata.ModelName)
	requireAntigravityEqual(t, persistedClaudeEnum, metadata.ModelEnum)
}

func TestPersistedModelName_Neg_DoesNotTreatTranscriptTextAsSelectedModel(t *testing.T) {
	raw := protoBytes(persistedMetadataRootFieldNumber, protoMessage(protoString(persistedMetadataModelFieldNumber, persistedClaudeModelID)))
	metadata, parseErr := ParsePersistedGenerationMetadata(1, raw)

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, "", metadata.ModelName)
}

func TestPersistedModelName_Neg_RejectsUndeclaredModelName(t *testing.T) {
	metadata, parseErr := ParsePersistedGenerationMetadata(1, persistedModelFixture("unverified-model", persistedGeminiEnum, persistedBoundary))

	requireAntigravityNoError(t, parseErr)
	requireAntigravityEqual(t, "", metadata.ModelName)
}

func persistedModelFixture(modelID, modelEnum, inputBoundary string) []byte {
	envelopeFields := [][]byte{
		protoBytes(persistedMetadataAttributesFieldNumber, persistedMetadataMapEntry(persistedModelEnumKey, modelEnum)),
		protoBytes(persistedMetadataAttributesFieldNumber, persistedMetadataMapEntry(persistedLastStepIndexKey, inputBoundary)),
	}
	if modelID != "" {
		envelopeFields = append(envelopeFields, protoString(persistedMetadataModelFieldNumber, modelID))
	}
	return protoBytes(persistedMetadataRootFieldNumber, protoMessage(envelopeFields...))
}

func persistedMetadataMapEntry(key, value string) []byte {
	return protoMessage(
		protoString(persistedMetadataMapKeyFieldNumber, key),
		protoString(persistedMetadataMapValueFieldNumber, value),
	)
}
