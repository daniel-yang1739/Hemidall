package antigravity

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// SQLiteTelemetryReader polls and extracts schema-inferred generation metadata
// from the local Antigravity SQLite database.
type SQLiteTelemetryReader struct {
	dbPath     string
	mu         sync.RWMutex
	lastGenIdx int
	records    map[int]*PersistedGenerationMetadata // keyed by the generated transcript step
}

// NewSQLiteTelemetryReader creates a new SQLite telemetry reader
func NewSQLiteTelemetryReader(dbPath string) *SQLiteTelemetryReader {
	return &SQLiteTelemetryReader{
		dbPath:  dbPath,
		records: make(map[int]*PersistedGenerationMetadata),
	}
}

// PollLatest reads all new records from gen_metadata table
func (r *SQLiteTelemetryReader) PollLatest() error {
	dsn := fmt.Sprintf("file:%s?mode=ro", r.dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open sqlite db: %w", err)
	}
	defer db.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	query := "SELECT idx, data FROM gen_metadata WHERE idx > ? ORDER BY idx ASC"
	rows, err := db.Query(query, r.lastGenIdx)
	if err != nil {
		return fmt.Errorf("failed to query gen_metadata: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var idx int
		var data []byte
		if err := rows.Scan(&idx, &data); err != nil {
			continue
		}

		meta, err := ParsePersistedGenerationMetadata(idx, data)
		if err != nil {
			r.lastGenIdx = idx
			continue
		}

		r.lastGenIdx = idx
		if strings.Contains(meta.ModelName, "safety-le") {
			continue
		}

		if meta.HasInputBoundary {
			generatedStepIndex := meta.InputBoundaryStepIndex + 1
			r.records[generatedStepIndex] = meta
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate gen_metadata: %w", err)
	}
	return nil
}

// TakeTelemetryForGeneratedStep returns and removes a record after its
// transcript event has consumed it. The event itself retains the copied usage,
// so keeping every historical metadata blob in the reader is unnecessary.
func (r *SQLiteTelemetryReader) TakeTelemetryForGeneratedStep(stepIdx int) *PersistedGenerationMetadata {
	r.mu.Lock()
	defer r.mu.Unlock()
	metadata, exists := r.records[stepIdx]
	if !exists {
		return nil
	}
	delete(r.records, stepIdx)
	return metadata
}
