package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

const (
	defaultSampleLimit       = 3
	maximumDecodeDepth       = 4
	maximumPrintableTextSize = 240
	wireTypeVarint           = 0
	wireTypeFixed64          = 1
	wireTypeBytes            = 2
	wireTypeFixed32          = 5
)

type blobSource struct {
	table  string
	column string
}

type wireField struct {
	number   int
	wireType int
	integer  uint64
	bytes    []byte
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

	if auditErr := writeAudit(os.Stdout, database, *sampleLimit); auditErr != nil {
		fmt.Fprintln(os.Stderr, auditErr)
		os.Exit(1)
	}
}

func writeAudit(output io.Writer, database *sql.DB, sampleLimit int) error {
	for _, source := range blobSources {
		if err := writeSourceAudit(output, database, source, sampleLimit); err != nil {
			return err
		}
	}
	return nil
}

func writeSourceAudit(output io.Writer, database *sql.DB, source blobSource, sampleLimit int) error {
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
		if err := writeWireFields(output, data, 0); err != nil {
			if _, writeErr := fmt.Fprintf(output, "WIRE_DECODE_ERROR=%v\n", err); writeErr != nil {
				return writeErr
			}
		}
	}
	return rows.Err()
}

func writeWireFields(output io.Writer, data []byte, depth int) error {
	fields, decodeErr := decodeFields(data)
	if decodeErr != nil {
		return decodeErr
	}
	for _, field := range fields {
		indent := strings.Repeat("  ", depth)
		switch field.wireType {
		case wireTypeVarint:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=varint VALUE=%d\n", indent, field.number, field.integer); err != nil {
				return err
			}
		case wireTypeFixed64:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=fixed64 HEX=%s\n", indent, field.number, hex.EncodeToString(field.bytes)); err != nil {
				return err
			}
		case wireTypeFixed32:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=fixed32 HEX=%s\n", indent, field.number, hex.EncodeToString(field.bytes)); err != nil {
				return err
			}
		case wireTypeBytes:
			if _, err := fmt.Fprintf(output, "%sFIELD=%d WIRE=bytes LEN=%d HEX_PREFIX=%s", indent, field.number, len(field.bytes), hex.EncodeToString(field.bytes[:min(len(field.bytes), maximumPrintableTextSize)])); err != nil {
				return err
			}
			if text := printableText(field.bytes); text != "" {
				if _, err := fmt.Fprintf(output, " TEXT=%q", text); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(output); err != nil {
				return err
			}
			if depth < maximumDecodeDepth && printableText(field.bytes) == "" {
				if nested, nestedErr := decodeFields(field.bytes); nestedErr == nil && len(nested) > 0 {
					if _, err := fmt.Fprintf(output, "%sNESTED_MESSAGE\n", indent); err != nil {
						return err
					}
					if err := writeWireFields(output, field.bytes, depth+1); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func decodeFields(data []byte) ([]wireField, error) {
	fields := make([]wireField, 0)
	for offset := 0; offset < len(data); {
		tag, consumed := readVarint(data[offset:])
		if consumed == 0 || tag>>3 == 0 {
			return nil, fmt.Errorf("invalid tag at byte %d", offset)
		}
		offset += consumed
		field := wireField{number: int(tag >> 3), wireType: int(tag & 7)}
		switch field.wireType {
		case wireTypeVarint:
			value, valueBytes := readVarint(data[offset:])
			if valueBytes == 0 {
				return nil, fmt.Errorf("invalid varint at byte %d", offset)
			}
			field.integer = value
			offset += valueBytes
		case wireTypeFixed64:
			if offset+8 > len(data) {
				return nil, fmt.Errorf("truncated fixed64 at byte %d", offset)
			}
			field.bytes = data[offset : offset+8]
			offset += 8
		case wireTypeBytes:
			length, lengthBytes := readVarint(data[offset:])
			if lengthBytes == 0 || length > uint64(len(data)-offset-lengthBytes) {
				return nil, fmt.Errorf("invalid byte length at byte %d", offset)
			}
			offset += lengthBytes
			field.bytes = data[offset : offset+int(length)]
			offset += int(length)
		case wireTypeFixed32:
			if offset+4 > len(data) {
				return nil, fmt.Errorf("truncated fixed32 at byte %d", offset)
			}
			field.bytes = data[offset : offset+4]
			offset += 4
		default:
			return nil, fmt.Errorf("unsupported wire type %d", field.wireType)
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func readVarint(data []byte) (uint64, int) {
	var value uint64
	var shift uint
	for index, byteValue := range data {
		value |= uint64(byteValue&0x7f) << shift
		if byteValue&0x80 == 0 {
			return value, index + 1
		}
		shift += 7
		if shift >= 64 {
			return 0, 0
		}
	}
	return 0, 0
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

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
