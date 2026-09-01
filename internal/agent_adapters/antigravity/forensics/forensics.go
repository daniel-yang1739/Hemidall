package forensics

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/agent_adapters/antigravity/parsers/conversation"
)

const (
	executorRowsQuery         = "SELECT idx, data FROM executor_metadata ORDER BY idx ASC"
	executorRowQuery          = "SELECT data FROM executor_metadata WHERE idx = ?"
	executorTableQuery        = "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'executor_metadata')"
	executionIDLength         = 36
	printableRunMinimumLength = 3
)

// PersistedUsageAuditRow is one complete schema-inferred input usage observation.
type PersistedUsageAuditRow struct {
	GenIndex               int
	InputBoundaryStepIndex int
	HasInputBoundary       bool
	ModelName              string
	UncachedInputTokens    int
	CachedInputTokens      int
	HasCachedInputTokens   bool
}

// PersistedUsageAuditSummary makes audit inclusion explicit.
type PersistedUsageAuditSummary struct {
	ScannedRecordCount           int
	UsageRecordCount             int
	NoUncachedInputRecordCount   int
	SkippedMalformedRecordCount  int
	ExplicitCacheRecordCount     int
	InferredZeroCacheRecordCount int
	ExecutorModelMatchCount      int
	EnumModelMatchCount          int
	UnknownModelRecordCount      int
	UncachedInputTokenSum        int
	CachedInputTokenSum          int
}

func (summary PersistedUsageAuditSummary) ProcessedTokenSum() int {
	return summary.UncachedInputTokenSum + summary.CachedInputTokenSum
}

// PersistedUsageAudit is the ordered local observation report for gen_metadata.
type PersistedUsageAudit struct {
	Rows    []PersistedUsageAuditRow
	Summary PersistedUsageAuditSummary
}

// ExecutorMetadataAuditRow summarizes one executor metadata BLOB.
type ExecutorMetadataAuditRow struct {
	Index                  int
	ByteCount              int
	ObservedExecutionUUIDs []string
	ModelName              string
}

// ExecutorMetadataAudit is the ordered local observation report for executor_metadata.
type ExecutorMetadataAudit struct{ Rows []ExecutorMetadataAuditRow }

// ReadPersistedUsageAudit uses the production conversation parser and exposes
// a command-friendly chronological local report.
func ReadPersistedUsageAudit(databasePath string) (PersistedUsageAudit, error) {
	database, err := openReadOnly(databasePath)
	if err != nil {
		return PersistedUsageAudit{}, err
	}
	defer database.Close()
	var scanned int
	if err := database.QueryRow("SELECT COUNT(*) FROM gen_metadata").Scan(&scanned); err != nil {
		return PersistedUsageAudit{}, fmt.Errorf("count generation metadata: %w", err)
	}
	generations, diagnostics, err := (conversation.Parser{}).ParseDatabase(databasePath, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})
	if err != nil {
		return PersistedUsageAudit{}, err
	}
	audit := PersistedUsageAudit{Summary: PersistedUsageAuditSummary{ScannedRecordCount: scanned, SkippedMalformedRecordCount: len(diagnostics)}}
	for _, generation := range generations {
		if !generation.Usage.HasUncachedInputTokens {
			audit.Summary.NoUncachedInputRecordCount++
			continue
		}
		generationIndex, convertErr := strconv.Atoi(generation.ID)
		if convertErr != nil {
			continue
		}
		row := PersistedUsageAuditRow{GenIndex: generationIndex, InputBoundaryStepIndex: generation.StepIndex - 1, HasInputBoundary: generation.StepIndex > 0, ModelName: generation.ModelID, UncachedInputTokens: generation.Usage.UncachedInputTokens, CachedInputTokens: generation.Usage.CachedInputTokens, HasCachedInputTokens: generation.Usage.HasCachedInputTokens}
		audit.Rows = append(audit.Rows, row)
		audit.Summary.UsageRecordCount++
		audit.Summary.UncachedInputTokenSum += row.UncachedInputTokens
		audit.Summary.CachedInputTokenSum += row.CachedInputTokens
		if row.HasCachedInputTokens {
			audit.Summary.ExplicitCacheRecordCount++
		} else {
			audit.Summary.InferredZeroCacheRecordCount++
		}
		if row.ModelName == "" {
			audit.Summary.UnknownModelRecordCount++
		}
	}
	return audit, nil
}

func ReadExecutorMetadataAudit(databasePath string) (ExecutorMetadataAudit, error) {
	database, err := openReadOnly(databasePath)
	if err != nil {
		return ExecutorMetadataAudit{}, err
	}
	defer database.Close()
	if err := requireExecutorTable(database); err != nil {
		return ExecutorMetadataAudit{}, err
	}
	rows, err := database.Query(executorRowsQuery)
	if err != nil {
		return ExecutorMetadataAudit{}, fmt.Errorf("query executor metadata: %w", err)
	}
	defer rows.Close()
	audit := ExecutorMetadataAudit{}
	for rows.Next() {
		var index int
		var data []byte
		if err := rows.Scan(&index, &data); err != nil {
			return ExecutorMetadataAudit{}, err
		}
		audit.Rows = append(audit.Rows, inspectExecutorRow(index, data))
	}
	return audit, rows.Err()
}

func ReadExecutorMetadataPrintableStrings(databasePath string, index int) ([]string, error) {
	database, err := openReadOnly(databasePath)
	if err != nil {
		return nil, err
	}
	defer database.Close()
	var data []byte
	if err := database.QueryRow(executorRowQuery, index).Scan(&data); err != nil {
		return nil, fmt.Errorf("read executor metadata row %d: %w", index, err)
	}
	return printableRuns(data), nil
}

func openReadOnly(databasePath string) (*sql.DB, error) {
	if databasePath == "" {
		return nil, fmt.Errorf("database path is required")
	}
	database, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databasePath))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return database, nil
}
func requireExecutorTable(database *sql.DB) error {
	var exists int
	if err := database.QueryRow(executorTableQuery).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("executor_metadata table is not present")
	}
	return nil
}
func inspectExecutorRow(index int, data []byte) ExecutorMetadataAuditRow {
	return ExecutorMetadataAuditRow{Index: index, ByteCount: len(data), ObservedExecutionUUIDs: observedExecutionIDs(data), ModelName: embeddedModel(data)}
}
func embeddedModel(data []byte) string {
	fields := strings.FieldsFunc(string(data), func(value rune) bool {
		return !((value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') || value == '-' || value == '.')
	})
	for _, field := range fields {
		lower := strings.ToLower(field)
		if strings.HasPrefix(lower, "gemini-") || strings.HasPrefix(lower, "claude-") || strings.HasPrefix(lower, "gpt-") {
			return lower
		}
	}
	return ""
}
func observedExecutionIDs(data []byte) []string {
	values := make([]string, 0)
	seen := make(map[string]struct{})
	for start := 0; start+executionIDLength <= len(data); start++ {
		candidate := string(data[start : start+executionIDLength])
		if !canonicalExecutionID(candidate) {
			continue
		}
		if _, found := seen[candidate]; found {
			continue
		}
		seen[candidate] = struct{}{}
		values = append(values, candidate)
	}
	return values
}
func canonicalExecutionID(value string) bool {
	if len(value) != executionIDLength {
		return false
	}
	for index, value := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if value != '-' {
				return false
			}
			continue
		}
		if (value < '0' || value > '9') && (value < 'a' || value > 'f') && (value < 'A' || value > 'F') {
			return false
		}
	}
	return true
}
func printableRuns(data []byte) []string {
	values := make([]string, 0)
	seen := make(map[string]struct{})
	for index := 0; index < len(data); index++ {
		if data[index] < ' ' || data[index] > '~' {
			continue
		}
		start := index
		for index < len(data) && data[index] >= ' ' && data[index] <= '~' {
			index++
		}
		value := string(data[start:index])
		if len(value) < printableRunMinimumLength {
			continue
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}
