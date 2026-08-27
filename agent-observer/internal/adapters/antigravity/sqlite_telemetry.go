package antigravity

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// SQLiteTelemetryReader polls and extracts official Gemini generation telemetry from the local SQLite DB
type SQLiteTelemetryReader struct {
	dbPath    string
	mu        sync.RWMutex
	lastGenIdx int
	records   map[int]*GeminiGenerationMetadata // keyed by LastStepIdx
	latest    *GeminiGenerationMetadata
}

// NewSQLiteTelemetryReader creates a new SQLite telemetry reader
func NewSQLiteTelemetryReader(dbPath string) *SQLiteTelemetryReader {
	return &SQLiteTelemetryReader{
		dbPath:  dbPath,
		records: make(map[int]*GeminiGenerationMetadata),
	}
}

// PollLatest reads all new records from gen_metadata table
func (r *SQLiteTelemetryReader) PollLatest() error {
	dsn := fmt.Sprintf("file:%s?mode=ro&_journal=WAL", r.dbPath)
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

		meta, err := ParseGeminiGenMetadata(idx, data)
		if err != nil {
			continue
		}

		r.lastGenIdx = idx
		if meta.LastStepIdx > 0 {
			r.records[meta.LastStepIdx] = meta
		}
		r.latest = meta
	}

	return nil
}

// GetTelemetryForStep retrieves official telemetry for a specific step index (checking exact and adjacent input indices)
func (r *SQLiteTelemetryReader) GetTelemetryForStep(stepIdx int) *GeminiGenerationMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if meta, exists := r.records[stepIdx]; exists {
		return meta
	}
	if meta, exists := r.records[stepIdx-1]; exists {
		return meta
	}
	if meta, exists := r.records[stepIdx+1]; exists {
		return meta
	}
	return nil
}

// GetLatestTelemetry returns the most recently captured official telemetry
func (r *SQLiteTelemetryReader) GetLatestTelemetry() *GeminiGenerationMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.latest
}
