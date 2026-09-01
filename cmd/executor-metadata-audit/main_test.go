package main

import (
	"bytes"
	"strings"
	"testing"

	"heimdall/internal/agent_adapters/antigravity/forensics"
)

const (
	commandExecutorAuditIndex       = 52
	commandExecutorAuditBytes       = 5442
	commandExecutorAuditExecutionID = "bc5f926b-22c9-4c9a-9d0b-fab5c69ab9ea"
	commandExecutorAuditModelName   = "gemini-3.7-flash"
)

func TestWriteExecutorMetadataAudit_Pos_ListsObservedSummary(t *testing.T) {
	audit := forensics.ExecutorMetadataAudit{Rows: []forensics.ExecutorMetadataAuditRow{{
		Index:                  commandExecutorAuditIndex,
		ByteCount:              commandExecutorAuditBytes,
		ObservedExecutionUUIDs: []string{commandExecutorAuditExecutionID},
		ModelName:              commandExecutorAuditModelName,
	}}}
	var output bytes.Buffer

	writeErr := writeExecutorMetadataAudit(&output, audit)

	requireExecutorCommandNoError(t, writeErr)
	requireExecutorCommandContains(t, output.String(), "EXECUTOR IDX")
	requireExecutorCommandContains(t, output.String(), commandExecutorAuditExecutionID)
	requireExecutorCommandContains(t, output.String(), commandExecutorAuditModelName)
	requireExecutorCommandContains(t, output.String(), "TOTAL")
}

func requireExecutorCommandNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireExecutorCommandContains(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Fatalf("expected output to contain %q, got %s", expected, output)
	}
}
