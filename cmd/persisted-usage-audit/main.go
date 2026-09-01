package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"heimdall/internal/agent_adapters/antigravity/forensics"
	"heimdall/internal/agent_adapters/antigravity/parsers/transcript"
)

const (
	auditTabMinimumWidth          = 0
	auditTabWidth                 = 4
	auditTabPadding               = 1
	auditTabPadCharacter          = ' '
	auditTabFlags                 = 0
	auditModelDisplayWidth        = 20
	auditModelTruncationSuffix    = "…"
	transcriptScanBufferBytes     = 64 * 1024
	transcriptMaximumLineBytes    = 4 * 1024 * 1024
	unknownModelLabel             = "unknown"
	missingStepLabel              = "-"
	missingTimestampLabel         = "-"
	explicitCacheLabel            = "explicit"
	inferredZeroLabel             = "inferred-zero"
	conversationDatabaseExtension = ".db"
	conversationDirectoryName     = "conversations"
	brainDirectoryName            = "brain"
	systemGeneratedDirectoryName  = ".system_generated"
	logsDirectoryName             = "logs"
	fullTranscriptFilename        = "transcript_full.jsonl"
	compactTranscriptFilename     = "transcript.jsonl"
)

func main() {
	databasePath := flag.String("db", "", "read-only path to an Antigravity conversation SQLite database")
	transcriptPath := flag.String("transcript", "", "path to transcript_full.jsonl; derived from -db when omitted")
	flag.Parse()

	audit, auditErr := forensics.ReadPersistedUsageAudit(*databasePath)
	if auditErr != nil {
		fmt.Fprintln(os.Stderr, auditErr)
		os.Exit(1)
	}
	resolvedTranscriptPath, resolveErr := resolveTranscriptPath(*databasePath, *transcriptPath)
	if resolveErr != nil {
		fmt.Fprintln(os.Stderr, resolveErr)
		os.Exit(1)
	}
	timestamps, timestampErr := readTranscriptTimestamps(resolvedTranscriptPath)
	if timestampErr != nil {
		fmt.Fprintln(os.Stderr, timestampErr)
		os.Exit(1)
	}
	if err := writeUsageAudit(os.Stdout, audit, timestamps); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeUsageAudit(output io.Writer, audit forensics.PersistedUsageAudit, timestamps transcriptTimestampIndex) error {
	writer := tabwriter.NewWriter(output, auditTabMinimumWidth, auditTabWidth, auditTabPadding, auditTabPadCharacter, auditTabFlags)
	if _, err := fmt.Fprintln(writer, "GEN IDX\tGENERATED STEP\tMODEL\tUNCACHED\tCACHED\tRUNNING UNCACHED\tRUNNING CACHED\tCACHE FIELD\tTRANSCRIPT TIME"); err != nil {
		return err
	}

	runningUncached := 0
	runningCached := 0
	for _, row := range audit.Rows {
		runningUncached += row.UncachedInputTokens
		runningCached += row.CachedInputTokens
		step := formatGeneratedStep(row)
		timestamp := timestamps.format(row)
		modelName := formatAuditModelName(row.ModelName)
		cacheField := formatCacheField(row.HasCachedInputTokens)
		if _, err := fmt.Fprintf(writer, "%d\t%s\t%s\t%d\t%d\t%d\t%d\t%s\t%s\n", row.GenIndex, step, modelName, row.UncachedInputTokens, row.CachedInputTokens, runningUncached, runningCached, cacheField, timestamp); err != nil {
			return err
		}
	}

	summary := audit.Summary
	if _, err := fmt.Fprintf(writer, "TOTAL\t-\t-\t%d\t%d\t%d\t%d\t-\t-\n", summary.UncachedInputTokenSum, summary.CachedInputTokenSum, runningUncached, runningCached); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "SCANNED=%d COMPLETE_USAGE_ROWS=%d NO_UNCACHED_INPUT=%d MALFORMED=%d PROCESSED=%d EXPLICIT_CACHE=%d INFERRED_ZERO_CACHE=%d EXECUTOR_MODEL_MATCHES=%d ENUM_MODEL_MATCHES=%d UNKNOWN_MODELS=%d TRANSCRIPT_ROWS=%d VALID_TIMESTAMPS=%d AMBIGUOUS_STEPS=%d\n", summary.ScannedRecordCount, summary.UsageRecordCount, summary.NoUncachedInputRecordCount, summary.SkippedMalformedRecordCount, summary.ProcessedTokenSum(), summary.ExplicitCacheRecordCount, summary.InferredZeroCacheRecordCount, summary.ExecutorModelMatchCount, summary.EnumModelMatchCount, summary.UnknownModelRecordCount, timestamps.scannedRows, timestamps.validTimestampRows, timestamps.ambiguousStepCount); err != nil {
		return err
	}
	return writer.Flush()
}

func formatGeneratedStep(row forensics.PersistedUsageAuditRow) string {
	if !row.HasInputBoundary {
		return missingStepLabel
	}
	return fmt.Sprintf("%d", row.InputBoundaryStepIndex+1)
}

func formatAuditModelName(modelName string) string {
	if modelName == "" {
		return unknownModelLabel
	}
	runes := []rune(modelName)
	if len(runes) <= auditModelDisplayWidth {
		return modelName
	}
	prefixLength := auditModelDisplayWidth - len([]rune(auditModelTruncationSuffix))
	return string(runes[:prefixLength]) + auditModelTruncationSuffix
}

func formatCacheField(hasExplicitCacheValue bool) string {
	if hasExplicitCacheValue {
		return explicitCacheLabel
	}
	return inferredZeroLabel
}

type transcriptTimestampIndex struct {
	byGeneratedStep    map[int]time.Time
	scannedRows        int
	validTimestampRows int
	ambiguousStepCount int
}

func (index transcriptTimestampIndex) format(row forensics.PersistedUsageAuditRow) string {
	if !row.HasInputBoundary {
		return missingTimestampLabel
	}
	generatedStep := row.InputBoundaryStepIndex + 1
	timestamp, found := index.byGeneratedStep[generatedStep]
	if !found {
		return missingTimestampLabel
	}
	return timestamp.Local().Format(time.DateTime)
}

func resolveTranscriptPath(databasePath, explicitTranscriptPath string) (string, error) {
	if explicitTranscriptPath != "" {
		return explicitTranscriptPath, nil
	}
	if filepath.Ext(databasePath) != conversationDatabaseExtension {
		return "", fmt.Errorf("derive transcript path: expected a %s conversation database", conversationDatabaseExtension)
	}
	sessionID := strings.TrimSuffix(filepath.Base(databasePath), conversationDatabaseExtension)
	installationPath := filepath.Dir(filepath.Dir(databasePath))
	if filepath.Base(filepath.Dir(databasePath)) != conversationDirectoryName {
		return "", fmt.Errorf("derive transcript path: expected database inside %s", conversationDirectoryName)
	}
	logsPath := filepath.Join(installationPath, brainDirectoryName, sessionID, systemGeneratedDirectoryName, logsDirectoryName)
	fullTranscriptPath := filepath.Join(logsPath, fullTranscriptFilename)
	if _, statErr := os.Stat(fullTranscriptPath); statErr == nil {
		return fullTranscriptPath, nil
	}
	compactTranscriptPath := filepath.Join(logsPath, compactTranscriptFilename)
	if _, statErr := os.Stat(compactTranscriptPath); statErr == nil {
		return compactTranscriptPath, nil
	}
	return "", fmt.Errorf("derive transcript path: no transcript found for session %s", sessionID)
}

func readTranscriptTimestamps(transcriptPath string) (transcriptTimestampIndex, error) {
	file, openErr := os.Open(transcriptPath)
	if openErr != nil {
		return transcriptTimestampIndex{}, fmt.Errorf("open transcript: %w", openErr)
	}
	defer file.Close()

	index := transcriptTimestampIndex{byGeneratedStep: make(map[int]time.Time)}
	ambiguousSteps := make(map[int]struct{})
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, transcriptScanBufferBytes), transcriptMaximumLineBytes)
	for scanner.Scan() {
		index.scannedRows++
		var raw transcript.RawTranscriptLine
		if decodeErr := json.Unmarshal(scanner.Bytes(), &raw); decodeErr != nil {
			continue
		}
		timestamp, parseErr := time.Parse(time.RFC3339, raw.CreatedAt)
		if parseErr != nil {
			continue
		}
		index.validTimestampRows++
		if _, ambiguous := ambiguousSteps[raw.StepIndex]; ambiguous {
			continue
		}
		if _, alreadyPresent := index.byGeneratedStep[raw.StepIndex]; alreadyPresent {
			delete(index.byGeneratedStep, raw.StepIndex)
			ambiguousSteps[raw.StepIndex] = struct{}{}
			index.ambiguousStepCount++
			continue
		}
		index.byGeneratedStep[raw.StepIndex] = timestamp
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return transcriptTimestampIndex{}, fmt.Errorf("scan transcript: %w", scanErr)
	}
	return index, nil
}
