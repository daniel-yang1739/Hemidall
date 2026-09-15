package main

import (
	"bytes"
	"encoding/json"
	"heimdall/internal/core"
	"io"
	"os"
	"path/filepath"
	"testing"
)

const reportCLITestMode = 0o600

func TestReportCLIReadsExplicitSourcesAndEmitsOnlyJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(path, []byte("{\"step_index\":1,\"type\":\"RUN_COMMAND\",\"created_at\":\"2026-09-01T01:02:03Z\",\"content\":\"private output\"}\n"), reportCLITestMode); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"-report", "-file", path, "-session", "fixture", "-format", "json"}, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	var report core.SessionAnalysisReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Session.SessionID != "fixture" || report.SchemaVersion != core.ReportSchemaVersion || report.Health.State != core.MonitorSnapshot {
		t.Fatalf("invalid CLI report: %+v", report)
	}
	if bytes.Contains(output.Bytes(), []byte("private output")) {
		t.Fatal("report exposed raw content")
	}
	if len(report.ToolOutputs) != 1 {
		t.Fatal("CLI did not analyze the transcript")
	}
}

func TestReportCLIRejectsInvalidOptionsAndSources(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"format", []string{"-report", "-format", "invalid"}},
		{"mode", []string{"-report", "-plain"}},
		{"server", []string{"-report", "-port", "8080"}},
		{"source", []string{"-report", "-file", filepath.Join(t.TempDir(), "missing"), "-session", "fixture"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := run(tc.args, io.Discard, io.Discard); err == nil {
				t.Fatal("expected invalid request to fail")
			}
		})
	}
}
