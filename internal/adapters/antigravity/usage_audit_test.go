package antigravity

import (
	"database/sql"
	"path/filepath"
	"testing"
)

const (
	usageAuditExplicitRecordIndex         = 11
	usageAuditDefaultRecordIndex          = 12
	usageAuditNoUsageRecordIndex          = 13
	usageAuditMalformedRecordIndex        = 14
	usageAuditDirectModelRecordIndex      = 21
	usageAuditInferredModelRecordIndex    = 22
	usageAuditAmbiguousGeminiRecordIndex  = 31
	usageAuditAmbiguousGPTRecordIndex     = 32
	usageAuditAmbiguousUnknownRecordIndex = 33
	usageAuditExecutorResolvedRecordIndex = 33
	usageAuditExplicitMeteredTokens       = 30
	usageAuditExplicitCachedTokens        = 70
	usageAuditDefaultMeteredTokens        = 20
	usageAuditModelMeteredTokens          = 10
	usageAuditModelCachedTokens           = 90
)

func TestReadPersistedUsageAudit_Pos_AccountsForExplicitAndDefaultZeroCacheRows(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditExplicitRecordIndex, persistedUsageAuditFixture(usageAuditExplicitMeteredTokens, usageAuditExplicitCachedTokens, true))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditDefaultRecordIndex, persistedUsageAuditFixture(usageAuditDefaultMeteredTokens, 0, false))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditNoUsageRecordIndex, []byte("metadata"))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditMalformedRecordIndex, []byte{})
	requireAntigravityNoError(t, database.Close())

	audit, auditErr := ReadPersistedUsageAudit(databasePath)

	requireAntigravityNoError(t, auditErr)
	requireAntigravityEqual(t, 2, len(audit.Rows))
	requireAntigravityEqual(t, usageAuditExplicitRecordIndex, audit.Rows[0].GenIndex)
	requireAntigravityEqual(t, usageAuditExplicitMeteredTokens, audit.Rows[0].MeteredInputTokens)
	requireAntigravityEqual(t, usageAuditExplicitCachedTokens, audit.Rows[0].CachedContentTokens)
	requireAntigravityEqual(t, true, audit.Rows[0].HasCachedContentTokens)
	requireAntigravityEqual(t, usageAuditDefaultRecordIndex, audit.Rows[1].GenIndex)
	requireAntigravityEqual(t, usageAuditDefaultMeteredTokens, audit.Rows[1].MeteredInputTokens)
	requireAntigravityEqual(t, 0, audit.Rows[1].CachedContentTokens)
	requireAntigravityEqual(t, false, audit.Rows[1].HasCachedContentTokens)
	requireAntigravityEqual(t, 4, audit.Summary.ScannedRecordCount)
	requireAntigravityEqual(t, 2, audit.Summary.UsageRecordCount)
	requireAntigravityEqual(t, 1, audit.Summary.NoMeteredInputRecordCount)
	requireAntigravityEqual(t, 1, audit.Summary.SkippedMalformedRecordCount)
	requireAntigravityEqual(t, 1, audit.Summary.ExplicitCacheRecordCount)
	requireAntigravityEqual(t, 1, audit.Summary.DefaultZeroCacheRecordCount)
	requireAntigravityEqual(t, usageAuditExplicitMeteredTokens+usageAuditDefaultMeteredTokens, audit.Summary.MeteredInputTokenSum)
	requireAntigravityEqual(t, usageAuditExplicitCachedTokens, audit.Summary.CachedContentTokenSum)
	requireAntigravityEqual(t, usageAuditExplicitMeteredTokens+usageAuditDefaultMeteredTokens+usageAuditExplicitCachedTokens, audit.Summary.ProcessedTokenSum())
}

func TestReadPersistedUsageAudit_Pos_InfersUniqueModelFromSessionEnum(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditDirectModelRecordIndex, persistedUsageAuditModelFixture(persistedGeminiModelID, persistedGeminiEnum))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditInferredModelRecordIndex, persistedUsageAuditModelFixture("", persistedGeminiEnum))
	requireAntigravityNoError(t, database.Close())

	audit, auditErr := ReadPersistedUsageAudit(databasePath)

	requireAntigravityNoError(t, auditErr)
	requireAntigravityEqual(t, persistedGeminiModelID, audit.Rows[usageAuditInferredModelRecordIndex-usageAuditDirectModelRecordIndex].ModelName)
}

func TestReadPersistedUsageAudit_Boundary_RetainsUnknownForAmbiguousSessionEnum(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditAmbiguousGeminiRecordIndex, persistedUsageAuditModelFixture(persistedGeminiModelID, persistedGeminiEnum))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditAmbiguousGPTRecordIndex, persistedUsageAuditModelFixture(persistedGPTModelID, persistedGeminiEnum))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditAmbiguousUnknownRecordIndex, persistedUsageAuditModelFixture("", persistedGeminiEnum))
	requireAntigravityNoError(t, database.Close())

	audit, auditErr := ReadPersistedUsageAudit(databasePath)

	requireAntigravityNoError(t, auditErr)
	requireAntigravityEqual(t, "", audit.Rows[usageAuditAmbiguousUnknownRecordIndex-usageAuditAmbiguousGeminiRecordIndex].ModelName)
}

func TestReadPersistedUsageAudit_Pos_UsesExecutorMetadataBeforeAmbiguousEnum(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "conversation.db")
	database, openErr := sql.Open("sqlite", databasePath)
	requireAntigravityNoError(t, openErr)
	defer database.Close()
	requireAntigravityExec(t, database, "CREATE TABLE gen_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "CREATE TABLE executor_metadata (idx INTEGER PRIMARY KEY, data BLOB)")
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditAmbiguousGeminiRecordIndex, persistedUsageAuditModelFixture(persistedGeminiModelID, persistedGeminiEnum))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditAmbiguousGPTRecordIndex, persistedUsageAuditModelFixture(persistedGPTModelID, persistedGeminiEnum))
	requireAntigravityExec(t, database, "INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", usageAuditExecutorResolvedRecordIndex, persistedUsageAuditExecutionFixture(executorAuditExecutionID, persistedGeminiEnum))
	requireAntigravityExec(t, database, "INSERT INTO executor_metadata (idx, data) VALUES (?, ?)", executorAuditRecordIndex, executorMetadataAuditFixture(executorAuditExecutionID, persistedGeminiModelID))
	requireAntigravityNoError(t, database.Close())

	audit, auditErr := ReadPersistedUsageAudit(databasePath)

	requireAntigravityNoError(t, auditErr)
	requireAntigravityEqual(t, persistedGeminiModelID, audit.Rows[usageAuditExecutorResolvedRecordIndex-usageAuditAmbiguousGeminiRecordIndex].ModelName)
	requireAntigravityEqual(t, 1, audit.Summary.ExecutorModelMatchCount)
	requireAntigravityEqual(t, 0, audit.Summary.UnknownModelRecordCount)
}

func persistedUsageAuditFixture(meteredTokens, cachedTokens int, includeCachedTokens bool) []byte {
	usageFields := [][]byte{protoVarint(persistedUsageInputFieldNumber, uint64(meteredTokens))}
	if includeCachedTokens {
		usageFields = append(usageFields, protoVarint(persistedUsageCachedContentFieldNumber, uint64(cachedTokens)))
	}
	return protoBytes(persistedMetadataRootFieldNumber, protoMessage(protoBytes(4, protoMessage(usageFields...))))
}

func persistedUsageAuditModelFixture(modelID, modelEnum string) []byte {
	metadataFields := [][]byte{
		protoBytes(persistedMetadataAttributesFieldNumber, persistedMetadataMapEntry(persistedModelEnumKey, modelEnum)),
		protoBytes(4, protoMessage(
			protoVarint(persistedUsageInputFieldNumber, usageAuditModelMeteredTokens),
			protoVarint(persistedUsageCachedContentFieldNumber, usageAuditModelCachedTokens),
		)),
	}
	if modelID != "" {
		metadataFields = append(metadataFields, protoString(persistedMetadataModelFieldNumber, modelID))
	}
	return protoBytes(persistedMetadataRootFieldNumber, protoMessage(metadataFields...))
}

func persistedUsageAuditExecutionFixture(executionID, modelEnum string) []byte {
	metadataFields := [][]byte{
		protoBytes(persistedMetadataAttributesFieldNumber, persistedMetadataMapEntry(persistedModelEnumKey, modelEnum)),
		protoBytes(persistedMetadataAttributesFieldNumber, persistedMetadataMapEntry(persistedLastExecutionIDKey, executionID)),
		protoBytes(4, protoMessage(
			protoVarint(persistedUsageInputFieldNumber, usageAuditModelMeteredTokens),
			protoVarint(persistedUsageCachedContentFieldNumber, usageAuditModelCachedTokens),
		)),
	}
	return protoBytes(persistedMetadataRootFieldNumber, protoMessage(metadataFields...))
}
