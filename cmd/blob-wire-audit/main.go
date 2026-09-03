package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"unicode"
	"unicode/utf8"

	_ "modernc.org/sqlite"

	"heimdall/internal/agent_adapters/antigravity/wire"
)

const (
	defaultSampleLimit       = 3
	maximumDecodeDepth       = 4
	maximumPrintableTextSize = 240
	prettyTextTruncateAt     = 120
	profileTabMinimumWidth   = 0
	profileTabWidth          = 2
	profileTabPadding        = 1
	profileTabPadCharacter   = ' '
	profileTabFlags          = 0
	profileMissingValue      = "-"
	wireTypeVarint           = 0
	wireTypeFixed64          = 1
	wireTypeBytes            = 2
	wireTypeFixed32          = 5
)

type blobSource struct {
	table  string
	column string
}

var blobSources = []blobSource{
	{table: "steps", column: "metadata"},
	{table: "steps", column: "error_details"},
	{table: "steps", column: "permissions"},
	{table: "steps", column: "task_details"},
	{table: "steps", column: "render_info"},
	{table: "steps", column: "step_payload"},
	{table: "executor_metadata", column: "data"},
	{table: "gen_metadata", column: "data"},
	{table: "trajectory_metadata_blob", column: "data"},
}

func main() {
	databasePath := flag.String("db", "", "read-only path to an Antigravity conversation SQLite database")
	sampleLimit := flag.Int("limit", defaultSampleLimit, "representative non-empty rows per BLOB column")
	generationIndex := flag.Int("gen-idx", -1, "inspect one gen_metadata.idx row instead of representative samples")
	usageProfile := flag.Bool("usage-profile", false, "list raw scalar paths from every gen_metadata usage-like envelope")
	rawMode := flag.Bool("raw", false, "show raw wire format with hex prefixes (default: pretty key-value pairs)")
	flag.Parse()

	if *databasePath == "" {
		fmt.Fprintln(os.Stderr, "database path is required")
		os.Exit(1)
	}
	if *sampleLimit <= 0 {
		fmt.Fprintln(os.Stderr, "limit must be positive")
		os.Exit(1)
	}

	database, openErr := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", *databasePath))
	if openErr != nil {
		fmt.Fprintln(os.Stderr, openErr)
		os.Exit(1)
	}
	defer database.Close()
	if *generationIndex >= 0 {
		if auditErr := writeGenerationAudit(os.Stdout, database, *generationIndex, *rawMode); auditErr != nil {
			fmt.Fprintln(os.Stderr, auditErr)
			os.Exit(1)
		}
		return
	}
	if *usageProfile {
		if auditErr := writeUsageProfile(os.Stdout, database); auditErr != nil {
			fmt.Fprintln(os.Stderr, auditErr)
			os.Exit(1)
		}
		return
	}

	if auditErr := writeAudit(os.Stdout, database, *sampleLimit, *rawMode); auditErr != nil {
		fmt.Fprintln(os.Stderr, auditErr)
		os.Exit(1)
	}
}

type rawUsageProfile struct {
	generationIndex int
	modelID         string
	lastStepIndex   string
	usageScalars    map[int]uint64
	contextScalars  map[int]uint64
}

func writeUsageProfile(output io.Writer, database *sql.DB) error {
	rows, err := database.Query("SELECT idx, data FROM gen_metadata ORDER BY idx ASC")
	if err != nil {
		return fmt.Errorf("query generation usage profile: %w", err)
	}
	defer rows.Close()
	writer := tabwriter.NewWriter(output, profileTabMinimumWidth, profileTabWidth, profileTabPadding, profileTabPadCharacter, profileTabFlags)
	if _, err := fmt.Fprintln(writer, "GEN IDX\tLAST STEP\tMODEL\t1.4.1\t1.4.2\t1.4.3\t1.4.4\t1.4.5\t1.4.6\t1.4.8\t1.4.10\t1.9.10.1\t1.9.10.4"); err != nil {
		return err
	}
	for rows.Next() {
		var generationIndex int
		var data []byte
		if err := rows.Scan(&generationIndex, &data); err != nil {
			return fmt.Errorf("scan generation usage profile: %w", err)
		}
		profile, found := decodeRawUsageProfile(generationIndex, data)
		if !found {
			continue
		}
		if _, err := fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			profile.generationIndex,
			displayProfileString(profile.lastStepIndex),
			displayProfileString(profile.modelID),
			displayProfileScalar(profile.usageScalars, 1),
			displayProfileScalar(profile.usageScalars, 2),
			displayProfileScalar(profile.usageScalars, 3),
			displayProfileScalar(profile.usageScalars, 4),
			displayProfileScalar(profile.usageScalars, 5),
			displayProfileScalar(profile.usageScalars, 6),
			displayProfileScalar(profile.usageScalars, 8),
			displayProfileScalar(profile.usageScalars, 10),
			displayProfileScalar(profile.contextScalars, 1),
			displayProfileScalar(profile.contextScalars, 4),
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate generation usage profile: %w", err)
	}
	return writer.Flush()
}

func decodeRawUsageProfile(generationIndex int, data []byte) (rawUsageProfile, bool) {
	root, err := wire.Decode(data)
	if err != nil {
		return rawUsageProfile{}, false
	}
	metadata, found := nestedFields(root, 1)
	if !found {
		return rawUsageProfile{}, false
	}
	profile := rawUsageProfile{
		generationIndex: generationIndex,
		modelID:         string(wire.FirstBytes(metadata, 19)),
		lastStepIndex:   attributeString(metadata, "last_step_index"),
		usageScalars:    scalarFields(metadata, 4),
		contextScalars:  nestedScalarFields(metadata, 9, 10),
	}
	return profile, len(profile.usageScalars) > 0 || len(profile.contextScalars) > 0
}

func nestedFields(fields []wire.Field, number int) ([]wire.Field, bool) {
	data := wire.FirstBytes(fields, number)
	if len(data) == 0 {
		return nil, false
	}
	nested, err := wire.Decode(data)
	if err != nil {
		return nil, false
	}
	return nested, true
}

func nestedScalarFields(fields []wire.Field, numbers ...int) map[int]uint64 {
	current := fields
	for _, number := range numbers {
		nested, found := nestedFields(current, number)
		if !found {
			return map[int]uint64{}
		}
		current = nested
	}
	return scalarFieldMap(current)
}

func scalarFields(fields []wire.Field, number int) map[int]uint64 {
	nested, found := nestedFields(fields, number)
	if !found {
		return map[int]uint64{}
	}
	return scalarFieldMap(nested)
}

func scalarFieldMap(fields []wire.Field) map[int]uint64 {
	values := make(map[int]uint64)
	for _, field := range fields {
		if field.WireType == wireTypeVarint {
			values[field.Number] = field.Integer
		}
	}
	return values
}

func attributeString(fields []wire.Field, key string) string {
	for _, field := range fields {
		if field.Number != 20 {
			continue
		}
		entry, err := wire.Decode(field.Bytes)
		if err != nil || string(wire.FirstBytes(entry, 1)) != key {
			continue
		}
		return string(wire.FirstBytes(entry, 2))
	}
	return ""
}

func displayProfileString(value string) string {
	if value == "" {
		return profileMissingValue
	}
	return value
}

func displayProfileScalar(values map[int]uint64, number int) string {
	value, found := values[number]
	if !found {
		return profileMissingValue
	}
	return fmt.Sprintf("%d", value)
}

func writeGenerationAudit(output io.Writer, database *sql.DB, generationIndex int, rawMode bool) error {
	var data []byte
	if err := database.QueryRow("SELECT data FROM gen_metadata WHERE idx = ?", generationIndex).Scan(&data); err != nil {
		return fmt.Errorf("read gen_metadata.idx=%d: %w", generationIndex, err)
	}
	if _, err := fmt.Fprintf(output, "## gen_metadata.idx=%d BYTES=%d\n", generationIndex, len(data)); err != nil {
		return err
	}
	if rawMode {
		return writeWireFields(output, data, 0)
	}
	return writeWireFieldsPretty(output, data, 0)
}

func writeAudit(output io.Writer, database *sql.DB, sampleLimit int, rawMode bool) error {
	for _, source := range blobSources {
		if err := writeSourceAudit(output, database, source, sampleLimit, rawMode); err != nil {
			return err
		}
	}
	return nil
}

func writeSourceAudit(output io.Writer, database *sql.DB, source blobSource, sampleLimit int, rawMode bool) error {
	query := fmt.Sprintf("SELECT rowid, %s FROM %s WHERE %s IS NOT NULL AND length(%s) > 0 ORDER BY length(%s) ASC, rowid ASC LIMIT ?", source.column, source.table, source.column, source.column, source.column)
	rows, queryErr := database.Query(query, sampleLimit)
	if queryErr != nil {
		return fmt.Errorf("query %s.%s: %w", source.table, source.column, queryErr)
	}
	defer rows.Close()

	if _, err := fmt.Fprintf(output, "\n## %s.%s\n", source.table, source.column); err != nil {
		return err
	}
	for rows.Next() {
		var rowID int
		var data []byte
		if scanErr := rows.Scan(&rowID, &data); scanErr != nil {
			return fmt.Errorf("scan %s.%s: %w", source.table, source.column, scanErr)
		}
		if _, err := fmt.Fprintf(output, "ROWID=%d BYTES=%d RAW_PREFIX=%s\n", rowID, len(data), hex.EncodeToString(data[:min(len(data), maximumPrintableTextSize)])); err != nil {
			return err
		}
		var decodeErr error
		if rawMode {
			decodeErr = writeWireFields(output, data, 0)
		} else {
			decodeErr = writeWireFieldsPretty(output, data, 0)
		}
		if decodeErr != nil {
			if _, writeErr := fmt.Fprintf(output, "WIRE_DECODE_ERROR=%v\n", decodeErr); writeErr != nil {
				return writeErr
			}
		}
	}
	return rows.Err()
}

func writeWireFields(output io.Writer, data []byte, depth int) error {
	fields, decodeErr := wire.Decode(data)
	if decodeErr != nil {
		return decodeErr
	}
	for _, field := range fields {
		indent := strings.Repeat("  ", depth)
		switch field.WireType {
		case wireTypeVarint:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=varint VALUE=%d\n", indent, field.Number, field.Integer); err != nil {
				return err
			}
		case wireTypeFixed64:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=fixed64 HEX=%s\n", indent, field.Number, hex.EncodeToString(field.Bytes)); err != nil {
				return err
			}
		case wireTypeFixed32:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=fixed32 HEX=%s\n", indent, field.Number, hex.EncodeToString(field.Bytes)); err != nil {
				return err
			}
		case wireTypeBytes:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=bytes LEN=%d HEX_PREFIX=%s", indent, field.Number, len(field.Bytes), hex.EncodeToString(field.Bytes[:min(len(field.Bytes), maximumPrintableTextSize)])); err != nil {
				return err
			}
			if text := printableText(field.Bytes); text != "" {
				if _, err := fmt.Fprintf(output, " TEXT=%q", text); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(output); err != nil {
				return err
			}
			if depth < maximumDecodeDepth && printableText(field.Bytes) == "" {
				if nested, nestedErr := wire.Decode(field.Bytes); nestedErr == nil && len(nested) > 0 {
					if _, err := fmt.Fprintf(output, "%sNESTED_MESSAGE\n", indent); err != nil {
						return err
					}
					if err := writeWireFields(output, field.Bytes, depth+1); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func printableText(data []byte) string {
	if len(data) == 0 || len(data) > maximumPrintableTextSize || !utf8.Valid(data) {
		return ""
	}
	for _, character := range string(data) {
		if !unicode.IsPrint(character) && !unicode.IsSpace(character) {
			return ""
		}
	}
	return string(data)
}

// isPrintableUTF8 returns true if data is non-empty valid UTF-8 containing only
// printable or whitespace runes. Unlike printableText it has no size limit, so
// it correctly classifies long system-prompt blobs as text in pretty mode.
func isPrintableUTF8(data []byte) bool {
	if len(data) == 0 || !utf8.Valid(data) {
		return false
	}
	for _, r := range string(data) {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// writeWireFieldsPretty renders decoded fields as indented key-value pairs.
// Text values are shown directly (truncated to prettyTextTruncateAt characters).
// Non-text bytes fields are recursed into when possible; binary blobs are
// labelled with their byte size. Hex prefixes are omitted.
func writeWireFieldsPretty(output io.Writer, data []byte, depth int) error {
	fields, decodeErr := wire.Decode(data)
	if decodeErr != nil {
		return decodeErr
	}
	indent := strings.Repeat("  ", depth)
	for _, field := range fields {
		switch field.WireType {
		case wireTypeVarint:
			if _, err := fmt.Fprintf(output, "%sFIELD[%d]: %d\n", indent, field.Number, field.Integer); err != nil {
				return err
			}
		case wireTypeFixed64:
			if _, err := fmt.Fprintf(output, "%sFIELD[%d]: (fixed64) %s\n", indent, field.Number, hex.EncodeToString(field.Bytes)); err != nil {
				return err
			}
		case wireTypeFixed32:
			if _, err := fmt.Fprintf(output, "%sFIELD[%d]: (fixed32) %s\n", indent, field.Number, hex.EncodeToString(field.Bytes)); err != nil {
				return err
			}
		case wireTypeBytes:
			if isPrintableUTF8(field.Bytes) {
				text := string(field.Bytes)
				display := text
				truncated := false
				if len([]rune(text)) > prettyTextTruncateAt {
					display = string([]rune(text)[:prettyTextTruncateAt])
					truncated = true
				}
				if truncated {
					if _, err := fmt.Fprintf(output, "%sFIELD[%d]: %q ... (%d more chars)\n", indent, field.Number, display, len([]rune(text))-prettyTextTruncateAt); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(output, "%sFIELD[%d]: %q\n", indent, field.Number, display); err != nil {
						return err
					}
				}
			} else if depth < maximumDecodeDepth {
				nested, nestedErr := wire.Decode(field.Bytes)
				if nestedErr == nil && len(nested) > 0 {
					if _, err := fmt.Fprintf(output, "%sFIELD[%d]: (%dB)\n", indent, field.Number, len(field.Bytes)); err != nil {
						return err
					}
					if err := writeWireFieldsPretty(output, field.Bytes, depth+1); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(output, "%sFIELD[%d]: (binary, %dB)\n", indent, field.Number, len(field.Bytes)); err != nil {
						return err
					}
				}
			} else {
				if _, err := fmt.Fprintf(output, "%sFIELD[%d]: (%dB, max depth)\n", indent, field.Number, len(field.Bytes)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
