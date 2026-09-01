package antigravity

import (
	"database/sql"
	"fmt"
)

const (
	executorMetadataRowByIndexQuery      = "SELECT data FROM executor_metadata WHERE idx = ?"
	executorPrintableStringMinimumLength = 3
)

// ExecutorMetadataAuditRow is a schema-inferred summary of one executor
// metadata blob. The identifiers and model name are observed values, not an
// official Antigravity schema contract.
type ExecutorMetadataAuditRow struct {
	Index                  int
	ByteCount              int
	ObservedExecutionUUIDs []string
	ModelName              string
}

// ExecutorMetadataAudit is the ordered summary of executor_metadata rows.
type ExecutorMetadataAudit struct {
	Rows []ExecutorMetadataAuditRow
}

// ReadExecutorMetadataAudit reads the executor_metadata table in index order
// without modifying the database.
func ReadExecutorMetadataAudit(databasePath string) (ExecutorMetadataAudit, error) {
	database, openErr := openReadOnlyAntigravityDatabase(databasePath)
	if openErr != nil {
		return ExecutorMetadataAudit{}, openErr
	}
	defer database.Close()

	hasTable, existsErr := hasExecutorMetadataTable(database)
	if existsErr != nil {
		return ExecutorMetadataAudit{}, existsErr
	}
	if !hasTable {
		return ExecutorMetadataAudit{}, fmt.Errorf("executor_metadata table is not present")
	}

	rows, queryErr := database.Query(executorMetadataRowsQuery)
	if queryErr != nil {
		return ExecutorMetadataAudit{}, fmt.Errorf("query executor metadata: %w", queryErr)
	}
	defer rows.Close()

	audit := ExecutorMetadataAudit{}
	for rows.Next() {
		var index int
		var data []byte
		if scanErr := rows.Scan(&index, &data); scanErr != nil {
			return ExecutorMetadataAudit{}, fmt.Errorf("scan executor metadata row: %w", scanErr)
		}
		audit.Rows = append(audit.Rows, newExecutorMetadataAuditRow(index, data))
	}
	if rowErr := rows.Err(); rowErr != nil {
		return ExecutorMetadataAudit{}, fmt.Errorf("iterate executor metadata rows: %w", rowErr)
	}
	return audit, nil
}

// ReadExecutorMetadataPrintableStrings returns printable byte runs for one
// blob. It is a forensic aid; it is intentionally separate from typed fields
// because the executor protobuf schema is not public.
func ReadExecutorMetadataPrintableStrings(databasePath string, index int) ([]string, error) {
	database, openErr := openReadOnlyAntigravityDatabase(databasePath)
	if openErr != nil {
		return nil, openErr
	}
	defer database.Close()

	var data []byte
	if queryErr := database.QueryRow(executorMetadataRowByIndexQuery, index).Scan(&data); queryErr != nil {
		return nil, fmt.Errorf("read executor metadata row %d: %w", index, queryErr)
	}
	return printableByteRuns(data), nil
}

func newExecutorMetadataAuditRow(index int, data []byte) ExecutorMetadataAuditRow {
	modelName, _ := executorDirectModelName(data)
	return ExecutorMetadataAuditRow{
		Index:                  index,
		ByteCount:              len(data),
		ObservedExecutionUUIDs: executorObservedUUIDs(data),
		ModelName:              modelName,
	}
}

func openReadOnlyAntigravityDatabase(databasePath string) (*sql.DB, error) {
	if databasePath == "" {
		return nil, fmt.Errorf("database path is required")
	}
	database, openErr := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databasePath))
	if openErr != nil {
		return nil, fmt.Errorf("open Antigravity database: %w", openErr)
	}
	return database, nil
}

func executorObservedUUIDs(data []byte) []string {
	values := make([]string, 0)
	seen := make(map[string]struct{})
	for candidateStart := 0; candidateStart+executorExecutionIDLength <= len(data); candidateStart++ {
		candidate := string(data[candidateStart : candidateStart+executorExecutionIDLength])
		if !isCanonicalExecutionID(candidate) {
			continue
		}
		if _, alreadySeen := seen[candidate]; alreadySeen {
			continue
		}
		seen[candidate] = struct{}{}
		values = append(values, candidate)
	}
	return values
}

func printableByteRuns(data []byte) []string {
	values := make([]string, 0)
	seen := make(map[string]struct{})
	for index := 0; index < len(data); index++ {
		if !isPrintableASCII(data[index]) {
			continue
		}
		valueStart := index
		for index < len(data) && isPrintableASCII(data[index]) {
			index++
		}
		value := string(data[valueStart:index])
		if len(value) < executorPrintableStringMinimumLength {
			continue
		}
		if _, alreadySeen := seen[value]; alreadySeen {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func isPrintableASCII(value byte) bool {
	return value >= ' ' && value <= '~'
}
