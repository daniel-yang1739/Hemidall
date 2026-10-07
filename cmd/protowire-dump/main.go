package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	_ "modernc.org/sqlite"
)

// dumpWire recursively traverses Protobuf Wire Format bytes and prints their hierarchy.
func dumpWire(data []byte, depth int) {
	indent := strings.Repeat("  ", depth)
	b := data
	for len(b) > 0 {
		// num: Field Number (e.g. 1, 4)
		// typ: Wire Type (e.g. 0=Varint, 2=Bytes)
		// n:   Bytes consumed by reading this tag (< 0 on EOF or parse error)
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:] // Advance cursor past tag to value

		switch typ {
		case protowire.VarintType:
			val, m := protowire.ConsumeVarint(b)
			if m < 0 {
				return
			}
			b = b[m:]
			fmt.Printf("%s• Field %d (Varint): %d\n", indent, num, val)

		case protowire.BytesType:
			bytesVal, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return
			}
			b = b[m:]
			if len(bytesVal) > 0 && isLikelyMessage(bytesVal) {
				fmt.Printf("%s• Field %d (Message, %d bytes):\n", indent, num, len(bytesVal))
				dumpWire(bytesVal, depth+1)
			} else {
				fmt.Printf("%s• Field %d (Bytes): len=%d\n", indent, num, len(bytesVal))
			}

		default:
			m := protowire.ConsumeFieldValue(num, typ, b)
			if m < 0 {
				return
			}
			b = b[m:]
		}
	}
}

func isLikelyMessage(b []byte) bool {
	_, _, n := protowire.ConsumeTag(b)
	return n > 0
}

func findLatestDB() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	pattern := filepath.Join(home, ".gemini/antigravity-cli/conversations/*.db")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}

	var latestPath string
	var latestModTime int64
	for _, m := range matches {
		if strings.Contains(m, "conversation_summaries.db") {
			continue
		}
		fi, err := os.Stat(m)
		if err != nil {
			continue
		}
		if fi.ModTime().UnixNano() > latestModTime {
			latestModTime = fi.ModTime().UnixNano()
			latestPath = m
		}
	}
	return latestPath
}

func main() {
	dbFlag := flag.String("db", "", "Path to conversation SQLite database (*.db)")
	idxFlag := flag.Int("idx", -1, "gen_metadata row index to decode (default: first non-empty row)")
	flag.Parse()

	dbPath := *dbFlag
	if dbPath == "" {
		dbPath = findLatestDB()
	}
	if dbPath == "" {
		log.Fatal("No conversation database found. Please specify -db <path>")
	}

	fmt.Printf("📂 Reading SQLite database: %s\n\n", dbPath)

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", dbPath))
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	var blob []byte
	var idx int
	if *idxFlag >= 0 {
		idx = *idxFlag
		err = db.QueryRow("SELECT data FROM gen_metadata WHERE idx = ?", idx).Scan(&blob)
	} else {
		err = db.QueryRow("SELECT idx, data FROM gen_metadata WHERE size > 0 ORDER BY idx ASC LIMIT 1").Scan(&idx, &blob)
	}

	if err != nil {
		log.Fatalf("Failed to query gen_metadata: %v", err)
	}

	fmt.Printf("=== protowire raw output (gen_metadata.idx = %d, size = %d bytes) ===\n", idx, len(blob))
	dumpWire(blob, 0)
}
