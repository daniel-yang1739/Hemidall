package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteSessionModelAudits_Pos_ListsGenerationAndExecutorModels(t *testing.T) {
	var output bytes.Buffer
	writeErr := writeSessionModelAudits(&output, []sessionModelAuditRow{{
		sessionID:             "session-1",
		generationModels:      []string{"gemini-3.7-flash"},
		executorModels:        []string{"gemini-3.7-flash-high"},
		usageRows:             4,
		unknownUsageModelRows: 0,
	}})

	if writeErr != nil {
		t.Fatal(writeErr)
	}
	for _, expected := range []string{"SESSION ID", "session-1", "gemini-3.7-flash", "gemini-3.7-flash-high"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("expected output to contain %q, got %s", expected, output.String())
		}
	}
}

func TestDisplayModels_Neg_UsesMissingValueForNoModels(t *testing.T) {
	if displayModels(nil) != missingValue {
		t.Fatalf("expected %q for empty models", missingValue)
	}
}
