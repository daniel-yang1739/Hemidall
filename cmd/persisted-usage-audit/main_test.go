package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"heimdall/internal/adapters/antigravity"
)

const (
	commandAuditFirstGenerationIndex  = 7
	commandAuditSecondGenerationIndex = 8
	commandAuditFirstMeteredTokens    = 30
	commandAuditSecondMeteredTokens   = 20
	commandAuditFirstCachedTokens     = 70
	commandAuditGeneratedStep         = 42
	commandAuditTimestampYear         = 2026
	commandAuditTimestampMonth        = time.August
	commandAuditTimestampDay          = 31
	commandAuditTimestampHour         = 9
	commandAuditTimestampMinute       = 10
	commandAuditTimestampSecond       = 11
)

func TestWriteUsageAudit_Pos_ListsRowsAndRunningUncachedTotal(t *testing.T) {
	audit := antigravity.PersistedUsageAudit{
		Rows: []antigravity.PersistedUsageAuditRow{
			{GenIndex: commandAuditFirstGenerationIndex, InputBoundaryStepIndex: 41, HasInputBoundary: true, ModelName: "gemini-3.7-flash", MeteredInputTokens: commandAuditFirstMeteredTokens, CachedContentTokens: commandAuditFirstCachedTokens, HasCachedContentTokens: true},
			{GenIndex: commandAuditSecondGenerationIndex, MeteredInputTokens: commandAuditSecondMeteredTokens},
		},
		Summary: antigravity.PersistedUsageAuditSummary{
			ScannedRecordCount:          2,
			UsageRecordCount:            2,
			ExplicitCacheRecordCount:    1,
			DefaultZeroCacheRecordCount: 1,
			MeteredInputTokenSum:        commandAuditFirstMeteredTokens + commandAuditSecondMeteredTokens,
			CachedContentTokenSum:       commandAuditFirstCachedTokens,
		},
	}
	var output bytes.Buffer
	timestamps := transcriptTimestampIndex{byGeneratedStep: map[int]time.Time{
		commandAuditGeneratedStep: time.Date(commandAuditTimestampYear, commandAuditTimestampMonth, commandAuditTimestampDay, commandAuditTimestampHour, commandAuditTimestampMinute, commandAuditTimestampSecond, 0, time.Local),
	}}

	writeErr := writeUsageAudit(&output, audit, timestamps)

	requireCommandAuditNoError(t, writeErr)
	requireCommandAuditContains(t, output.String(), "GEN IDX")
	requireCommandAuditContains(t, output.String(), "TRANSCRIPT TIME")
	requireCommandAuditColumnOrder(t, output.String())
	requireCommandAuditContains(t, output.String(), "7")
	requireCommandAuditContains(t, output.String(), "42")
	requireCommandAuditContains(t, output.String(), "2026-08-31 09:10:11")
	requireCommandAuditContains(t, output.String(), "gemini-3.7-flash")
	requireCommandAuditContains(t, output.String(), "30")
	requireCommandAuditContains(t, output.String(), "default-zero")
	requireCommandAuditContains(t, output.String(), "TOTAL")
	requireCommandAuditContains(t, output.String(), "50")
	requireCommandAuditContains(t, output.String(), "SCANNED=2 USAGE_ROWS=2")
}

func TestFormatAuditModelName_Boundary_TruncatesOnlyOverlongModelIDs(t *testing.T) {
	shortModel := formatAuditModelName("gemini-3.7-flash")
	longModel := formatAuditModelName("gemini-3.7-flash-safety-le")

	requireCommandAuditEqual(t, "gemini-3.7-flash", shortModel)
	requireCommandAuditEqual(t, "gemini-3.7-flash-sa…", longModel)
}

func requireCommandAuditNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireCommandAuditContains(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Fatalf("expected output to contain %q, got %s", expected, output)
	}
}

func requireCommandAuditEqual(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func requireCommandAuditColumnOrder(t *testing.T, output string) {
	t.Helper()
	header := strings.Split(output, "\n")[0]
	genIndexColumn := strings.Index(header, "GEN IDX")
	generatedStepColumn := strings.Index(header, "GENERATED STEP")
	modelColumn := strings.Index(header, "MODEL")
	uncachedColumn := strings.Index(header, "UNCACHED")
	cachedColumn := strings.Index(header, "CACHED")
	runningUncachedColumn := strings.Index(header, "RUNNING UNCACHED")
	runningCachedColumn := strings.Index(header, "RUNNING CACHED")
	cacheFieldColumn := strings.Index(header, "CACHE FIELD")
	timestampColumn := strings.Index(header, "TRANSCRIPT TIME")
	if genIndexColumn < 0 || generatedStepColumn <= genIndexColumn || modelColumn <= generatedStepColumn || uncachedColumn <= modelColumn || cachedColumn <= uncachedColumn || runningUncachedColumn <= cachedColumn || runningCachedColumn <= runningUncachedColumn || cacheFieldColumn <= runningCachedColumn || timestampColumn <= cacheFieldColumn {
		t.Fatalf("unexpected audit column order: %s", header)
	}
}
