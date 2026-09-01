package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"heimdall/internal/adapters/antigravity"
)

const (
	databaseExtension = ".db"
	tabMinimumWidth   = 0
	tabWidth          = 4
	tabPadding        = 1
	tabPadCharacter   = ' '
	tabFlags          = 0
	missingValue      = "-"
)

type sessionModelAuditRow struct {
	sessionID               string
	generationModels        []string
	executorModels          []string
	usageRows               int
	unknownUsageModelRows   int
	noMeteredGenerationRows int
}

func main() {
	databaseDirectory := flag.String("dir", "", "directory containing Antigravity conversation .db files")
	flag.Parse()

	if *databaseDirectory == "" {
		fmt.Fprintln(os.Stderr, "database directory is required")
		os.Exit(1)
	}
	rows, auditErr := readSessionModelAudits(*databaseDirectory)
	if auditErr != nil {
		fmt.Fprintln(os.Stderr, auditErr)
		os.Exit(1)
	}
	if writeSessionModelAudits(os.Stdout, rows) != nil {
		fmt.Fprintln(os.Stderr, "write session model audit")
		os.Exit(1)
	}
}

func readSessionModelAudits(databaseDirectory string) ([]sessionModelAuditRow, error) {
	entries, readErr := os.ReadDir(databaseDirectory)
	if readErr != nil {
		return nil, fmt.Errorf("read database directory: %w", readErr)
	}
	rows := make([]sessionModelAuditRow, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != databaseExtension {
			continue
		}
		if entry.Name() == "conversation_summaries.db" {
			continue
		}
		databasePath := filepath.Join(databaseDirectory, entry.Name())
		audit, auditErr := antigravity.ReadPersistedUsageAudit(databasePath)
		if auditErr != nil {
			continue
		}
		executorAudit, executorErr := antigravity.ReadExecutorMetadataAudit(databasePath)
		if executorErr != nil {
			executorAudit = antigravity.ExecutorMetadataAudit{}
		}
		row := sessionModelAuditRow{
			sessionID:               strings.TrimSuffix(entry.Name(), databaseExtension),
			generationModels:        distinctGenerationModels(audit.Rows),
			executorModels:          distinctExecutorModels(executorAudit.Rows),
			usageRows:               audit.Summary.UsageRecordCount,
			unknownUsageModelRows:   audit.Summary.UnknownModelRecordCount,
			noMeteredGenerationRows: audit.Summary.NoMeteredInputRecordCount,
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(left, right int) bool {
		return rows[left].sessionID < rows[right].sessionID
	})
	return rows, nil
}

func distinctGenerationModels(rows []antigravity.PersistedUsageAuditRow) []string {
	values := make(map[string]struct{})
	for _, row := range rows {
		if row.ModelName != "" {
			values[row.ModelName] = struct{}{}
		}
	}
	return sortedKeys(values)
}

func distinctExecutorModels(rows []antigravity.ExecutorMetadataAuditRow) []string {
	values := make(map[string]struct{})
	for _, row := range rows {
		if row.ModelName != "" {
			values[row.ModelName] = struct{}{}
		}
	}
	return sortedKeys(values)
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}

func writeSessionModelAudits(output io.Writer, rows []sessionModelAuditRow) error {
	writer := tabwriter.NewWriter(output, tabMinimumWidth, tabWidth, tabPadding, tabPadCharacter, tabFlags)
	if _, err := fmt.Fprintln(writer, "SESSION ID\tGENERATION MODELS\tEXECUTOR MODELS\tUSAGE ROWS\tUNKNOWN USAGE MODELS\tNO-METERED ROWS"); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%d\t%d\t%d\n", row.sessionID, displayModels(row.generationModels), displayModels(row.executorModels), row.usageRows, row.unknownUsageModelRows, row.noMeteredGenerationRows); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func displayModels(models []string) string {
	if len(models) == 0 {
		return missingValue
	}
	return strings.Join(models, ",")
}
