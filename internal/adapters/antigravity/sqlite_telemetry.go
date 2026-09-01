package antigravity

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

const (
	initialGenerationMetadataIndex = -1
	initialExecutorMetadataIndex   = -1
)

// SQLiteTelemetryReader polls and extracts schema-inferred generation metadata
// from the local Antigravity SQLite database.
type SQLiteTelemetryReader struct {
	dbPath                  string
	mu                      sync.RWMutex
	lastGenIdx              int
	lastExecutorMetadataIdx int
	records                 map[int]*PersistedGenerationMetadata // keyed by the generated transcript step
	modelResolver           persistedModelNameResolver
	executorModelNameIndex  executorModelNameIndex
}

// persistedModelNameResolver learns only session-local, unambiguous enum to
// model-ID mappings. A conflicting mapping remains unresolved instead of
// guessing which model produced a record without a direct ID.
type persistedModelNameResolver struct {
	namesByEnum    map[string]string
	ambiguousEnums map[string]struct{}
}

func newPersistedModelNameResolver() persistedModelNameResolver {
	return persistedModelNameResolver{
		namesByEnum:    make(map[string]string),
		ambiguousEnums: make(map[string]struct{}),
	}
}

// NewSQLiteTelemetryReader creates a new SQLite telemetry reader
func NewSQLiteTelemetryReader(dbPath string) *SQLiteTelemetryReader {
	return &SQLiteTelemetryReader{
		dbPath:                  dbPath,
		lastGenIdx:              initialGenerationMetadataIndex,
		lastExecutorMetadataIdx: initialExecutorMetadataIndex,
		records:                 make(map[int]*PersistedGenerationMetadata),
		modelResolver:           newPersistedModelNameResolver(),
		executorModelNameIndex:  newExecutorModelNameIndex(),
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
	if executorErr := r.pollExecutorMetadata(db); executorErr != nil {
		return executorErr
	}

	query := "SELECT idx, data FROM gen_metadata WHERE idx > ? ORDER BY idx ASC"
	rows, err := db.Query(query, r.lastGenIdx)
	if err != nil {
		return fmt.Errorf("failed to query gen_metadata: %w", err)
	}
	defer rows.Close()

	var metadataRows []*PersistedGenerationMetadata
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
		r.observeModelEnumMapping(meta)
		metadataRows = append(metadataRows, meta)
	}
	for _, meta := range metadataRows {
		r.resolveModelName(meta)
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

func (r *SQLiteTelemetryReader) observeModelEnumMapping(meta *PersistedGenerationMetadata) {
	r.modelResolver.observe(meta)
}

func (resolver *persistedModelNameResolver) observe(meta *PersistedGenerationMetadata) {
	if meta.ModelEnum == "" || meta.ModelName == "" {
		return
	}
	if _, ambiguous := resolver.ambiguousEnums[meta.ModelEnum]; ambiguous {
		return
	}
	knownModelName, known := resolver.namesByEnum[meta.ModelEnum]
	if !known || knownModelName == meta.ModelName {
		resolver.namesByEnum[meta.ModelEnum] = meta.ModelName
		return
	}
	delete(resolver.namesByEnum, meta.ModelEnum)
	resolver.ambiguousEnums[meta.ModelEnum] = struct{}{}
}

func (r *SQLiteTelemetryReader) resolveModelNameFromEnum(meta *PersistedGenerationMetadata) {
	r.resolveModelName(meta)
}

func (r *SQLiteTelemetryReader) resolveModelName(meta *PersistedGenerationMetadata) {
	if r.executorModelNameIndex.resolve(meta) {
		return
	}
	r.modelResolver.resolve(meta)
}

func (resolver *persistedModelNameResolver) resolve(meta *PersistedGenerationMetadata) {
	if meta.ModelName != "" || meta.ModelEnum == "" {
		return
	}
	if _, ambiguous := resolver.ambiguousEnums[meta.ModelEnum]; ambiguous {
		return
	}
	meta.ModelName = resolver.namesByEnum[meta.ModelEnum]
}

func (r *SQLiteTelemetryReader) pollExecutorMetadata(database *sql.DB) error {
	hasTable, existsErr := hasExecutorMetadataTable(database)
	if existsErr != nil {
		return existsErr
	}
	if !hasTable {
		return nil
	}

	rows, queryErr := database.Query(executorMetadataRowsSinceQuery, r.lastExecutorMetadataIdx)
	if queryErr != nil {
		return fmt.Errorf("query executor metadata: %w", queryErr)
	}
	defer rows.Close()
	for rows.Next() {
		var index int
		var data []byte
		if scanErr := rows.Scan(&index, &data); scanErr != nil {
			return fmt.Errorf("scan executor metadata row: %w", scanErr)
		}
		r.lastExecutorMetadataIdx = index
		r.executorModelNameIndex.observe(data)
	}
	if rowErr := rows.Err(); rowErr != nil {
		return fmt.Errorf("iterate executor metadata rows: %w", rowErr)
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
