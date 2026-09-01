package antigravity

import (
	"database/sql"
	"fmt"
)

const persistedUsageAuditRowsQuery = "SELECT idx, data FROM gen_metadata ORDER BY idx ASC"

// PersistedUsageAuditRow is one metered usage observation decoded from a
// gen_metadata row. CachedContentTokens is zero when HasCachedContentTokens is
// false, matching proto3 scalar-default behavior.
type PersistedUsageAuditRow struct {
	GenIndex               int
	InputBoundaryStepIndex int
	HasInputBoundary       bool
	ModelName              string
	MeteredInputTokens     int
	CachedContentTokens    int
	HasCachedContentTokens bool
}

// PersistedUsageAuditSummary makes every inclusion and exclusion explicit so a
// caller can verify that the total is not silently derived from a subset.
type PersistedUsageAuditSummary struct {
	ScannedRecordCount          int
	UsageRecordCount            int
	NoMeteredInputRecordCount   int
	SkippedMalformedRecordCount int
	ExplicitCacheRecordCount    int
	DefaultZeroCacheRecordCount int
	ExecutorModelMatchCount     int
	EnumModelMatchCount         int
	UnknownModelRecordCount     int
	MeteredInputTokenSum        int
	CachedContentTokenSum       int
}

// ProcessedTokenSum returns the sum used as the cache-content percentage
// denominator: metered input plus cached content.
func (summary PersistedUsageAuditSummary) ProcessedTokenSum() int {
	return summary.MeteredInputTokenSum + summary.CachedContentTokenSum
}

// PersistedUsageAudit is a chronological, read-only view of all decodable
// metered usage observations in one Antigravity conversation database.
type PersistedUsageAudit struct {
	Rows    []PersistedUsageAuditRow
	Summary PersistedUsageAuditSummary
}

// ReadPersistedUsageAudit reads every gen_metadata row in index order. It does
// not alter the database, normalize values, or estimate tokens. Malformed rows
// and rows without a metered-input field are counted in Summary but excluded
// from Rows because they cannot contribute to the uncached-input total.
func ReadPersistedUsageAudit(databasePath string) (PersistedUsageAudit, error) {
	if databasePath == "" {
		return PersistedUsageAudit{}, fmt.Errorf("database path is required")
	}

	database, openErr := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databasePath))
	if openErr != nil {
		return PersistedUsageAudit{}, fmt.Errorf("open persisted usage database: %w", openErr)
	}
	defer database.Close()

	rows, queryErr := database.Query(persistedUsageAuditRowsQuery)
	if queryErr != nil {
		return PersistedUsageAudit{}, fmt.Errorf("query persisted usage rows: %w", queryErr)
	}
	defer rows.Close()

	audit := PersistedUsageAudit{}
	modelResolver := newPersistedModelNameResolver()
	metadataRows := make([]*PersistedGenerationMetadata, 0)
	for rows.Next() {
		var generationIndex int
		var data []byte
		if scanErr := rows.Scan(&generationIndex, &data); scanErr != nil {
			return PersistedUsageAudit{}, fmt.Errorf("scan persisted usage row: %w", scanErr)
		}

		audit.Summary.ScannedRecordCount++
		metadata, parseErr := ParsePersistedGenerationMetadata(generationIndex, data)
		if parseErr != nil {
			audit.Summary.SkippedMalformedRecordCount++
			continue
		}
		modelResolver.observe(metadata)
		metadataRows = append(metadataRows, metadata)
	}
	if rowErr := rows.Err(); rowErr != nil {
		return PersistedUsageAudit{}, fmt.Errorf("iterate persisted usage rows: %w", rowErr)
	}
	if closeErr := rows.Close(); closeErr != nil {
		return PersistedUsageAudit{}, fmt.Errorf("close persisted usage rows: %w", closeErr)
	}
	executorModels, executorErr := loadExecutorModelNameIndex(database)
	if executorErr != nil {
		return PersistedUsageAudit{}, executorErr
	}

	for _, metadata := range metadataRows {
		if executorModels.resolve(metadata) {
			audit.Summary.ExecutorModelMatchCount++
		} else if metadata.ModelName == "" {
			modelResolver.resolve(metadata)
			if metadata.ModelName != "" {
				audit.Summary.EnumModelMatchCount++
			}
		}
		if !metadata.HasMeteredInputTokens {
			audit.Summary.NoMeteredInputRecordCount++
			continue
		}

		auditRow := PersistedUsageAuditRow{
			GenIndex:               metadata.GenIndex,
			InputBoundaryStepIndex: metadata.InputBoundaryStepIndex,
			HasInputBoundary:       metadata.HasInputBoundary,
			ModelName:              metadata.ModelName,
			MeteredInputTokens:     metadata.MeteredInputTokens,
			CachedContentTokens:    metadata.CachedContentTokens,
			HasCachedContentTokens: metadata.HasCachedContentTokens,
		}
		audit.Rows = append(audit.Rows, auditRow)
		audit.Summary.UsageRecordCount++
		audit.Summary.MeteredInputTokenSum += auditRow.MeteredInputTokens
		audit.Summary.CachedContentTokenSum += auditRow.CachedContentTokens
		if auditRow.HasCachedContentTokens {
			audit.Summary.ExplicitCacheRecordCount++
		} else {
			audit.Summary.DefaultZeroCacheRecordCount++
		}
		if auditRow.ModelName == "" {
			audit.Summary.UnknownModelRecordCount++
		}
	}

	return audit, nil
}
