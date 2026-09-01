package antigravity

import (
	"database/sql"
	"fmt"
	"strings"
)

const (
	executorMetadataTableExistsQuery = "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'executor_metadata')"
	executorMetadataRowsQuery        = "SELECT idx, data FROM executor_metadata ORDER BY idx ASC"
	executorMetadataRowsSinceQuery   = "SELECT idx, data FROM executor_metadata WHERE idx > ? ORDER BY idx ASC"
	executorExecutionIDLength        = 36
	executorExecutionIDFirstHyphen   = 8
	executorExecutionIDSecondHyphen  = 13
	executorExecutionIDThirdHyphen   = 18
	executorExecutionIDFourthHyphen  = 23
)

// executorModelNameIndex provides an execution-ID join from generation
// metadata to a direct model ID stored in executor_metadata. Conflicting
// observations remain unresolved rather than being guessed.
type executorModelNameIndex struct {
	namesByExecutionID    map[string]string
	ambiguousExecutionIDs map[string]struct{}
}

func newExecutorModelNameIndex() executorModelNameIndex {
	return executorModelNameIndex{
		namesByExecutionID:    make(map[string]string),
		ambiguousExecutionIDs: make(map[string]struct{}),
	}
}

func loadExecutorModelNameIndex(database *sql.DB) (executorModelNameIndex, error) {
	index := newExecutorModelNameIndex()
	hasTable, existsErr := hasExecutorMetadataTable(database)
	if existsErr != nil {
		return executorModelNameIndex{}, existsErr
	}
	if !hasTable {
		return index, nil
	}

	rows, queryErr := database.Query(executorMetadataRowsQuery)
	if queryErr != nil {
		return executorModelNameIndex{}, fmt.Errorf("query executor metadata: %w", queryErr)
	}
	defer rows.Close()

	for rows.Next() {
		var indexNumber int
		var data []byte
		if scanErr := rows.Scan(&indexNumber, &data); scanErr != nil {
			return executorModelNameIndex{}, fmt.Errorf("scan executor metadata row: %w", scanErr)
		}
		index.observe(data)
	}
	if rowErr := rows.Err(); rowErr != nil {
		return executorModelNameIndex{}, fmt.Errorf("iterate executor metadata rows: %w", rowErr)
	}
	return index, nil
}

func hasExecutorMetadataTable(database *sql.DB) (bool, error) {
	var hasExecutorMetadataTable int
	if existsErr := database.QueryRow(executorMetadataTableExistsQuery).Scan(&hasExecutorMetadataTable); existsErr != nil {
		return false, fmt.Errorf("check executor metadata table: %w", existsErr)
	}
	return hasExecutorMetadataTable != 0, nil
}

func (index *executorModelNameIndex) observe(data []byte) {
	modelName, hasModelName := executorDirectModelName(data)
	if !hasModelName {
		return
	}
	for _, executionID := range executorObservedUUIDs(data) {
		index.observeModelName(executionID, modelName)
	}
}

func (index *executorModelNameIndex) observeModelName(executionID, modelName string) {
	if _, ambiguous := index.ambiguousExecutionIDs[executionID]; ambiguous {
		return
	}
	knownModelName, known := index.namesByExecutionID[executionID]
	if !known || knownModelName == modelName {
		index.namesByExecutionID[executionID] = modelName
		return
	}
	delete(index.namesByExecutionID, executionID)
	index.ambiguousExecutionIDs[executionID] = struct{}{}
}

func (index *executorModelNameIndex) resolve(metadata *PersistedGenerationMetadata) bool {
	if metadata.ModelName != "" || metadata.LastExecutionID == "" {
		return false
	}
	modelName, found := index.namesByExecutionID[metadata.LastExecutionID]
	if !found {
		return false
	}
	metadata.ModelName = modelName
	return true
}

func isCanonicalExecutionID(value string) bool {
	if len(value) != executorExecutionIDLength {
		return false
	}
	for index, character := range value {
		if isExecutionIDHyphenIndex(index) {
			if character != '-' {
				return false
			}
			continue
		}
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') && (character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}

func isExecutionIDHyphenIndex(index int) bool {
	return index == executorExecutionIDFirstHyphen ||
		index == executorExecutionIDSecondHyphen ||
		index == executorExecutionIDThirdHyphen ||
		index == executorExecutionIDFourthHyphen
}

func executorDirectModelName(data []byte) (string, bool) {
	matches := embeddedModelNamePattern.FindAllString(string(data), -1)
	if len(matches) == 0 {
		return "", false
	}
	modelName := strings.ToLower(matches[0])
	for _, match := range matches[1:] {
		if strings.ToLower(match) != modelName {
			return "", false
		}
	}
	return modelName, true
}
