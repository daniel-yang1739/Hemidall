package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"unicode/utf8"

	_ "modernc.org/sqlite"

	"heimdall/internal/agent_adapters/antigravity/wire"
)

const (
	previewTextLimit = 100
)

func main() {
	dbPath := flag.String("db", "", "Path to SQLite conversation database (*.db)")
	genIndex := flag.Int("idx", -1, "gen_metadata row index to decode (default: 0)")
	flag.Parse()

	if *dbPath == "" {
		fmt.Println("Usage: go run ./cmd/cortex-decode -db <path.db> [-idx <number>]")
		os.Exit(1)
	}

	targetIndex := *genIndex
	if targetIndex < 0 {
		targetIndex = 0
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", *dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open SQLite database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var blobData []byte
	queryErr := db.QueryRow("SELECT data FROM gen_metadata WHERE idx = ?", targetIndex).Scan(&blobData)
	if queryErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to query gen_metadata idx=%d: %v\n", targetIndex, queryErr)
		os.Exit(1)
	}

	fmt.Printf("================================================================================\n")
	fmt.Printf("  🧬 CORTEX PROTOBUF DECODER (exa.cortex_pb.ChatModelMetadata)\n")
	fmt.Printf("================================================================================\n")
	fmt.Printf("  • Database File    : %s\n", *dbPath)
	fmt.Printf("  • gen_metadata.idx : %d (%d Raw Bytes)\n", targetIndex, len(blobData))
	fmt.Printf("--------------------------------------------------------------------------------\n\n")

	rootFields, decodeErr := wire.Decode(blobData)
	if decodeErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode root wire fields: %v\n", decodeErr)
		os.Exit(1)
	}

	// 1. Root Envelope Analysis
	fmt.Println("📦 [ROOT ENVELOPE: dbtrajectory.genMetadata]")
	for _, f := range rootFields {
		switch f.Number {
		case 1:
			fmt.Printf("  ├── Field 1 [bytes]  : payload (ChatModelMetadata, %d bytes)\n", len(f.Bytes))
		case 2:
			fmt.Printf("  ├── Field 2 [bytes]  : subsystem_id = %q\n", formatBytesOrHex(f.Bytes))
		case 4:
			fmt.Printf("  ├── Field 4 [string] : execution_id = %q\n", string(f.Bytes))
		case 8:
			fmt.Printf("  ├── Field 8 [bytes]  : state_fingerprint = 0x%s\n", hex.EncodeToString(f.Bytes[:min(len(f.Bytes), 16)]))
		case 10:
			fmt.Printf("  └── Field 10 [varint]: status = %d (OK/SUCCESS)\n", f.Integer)
		default:
			fmt.Printf("  ├── Field %d [%s]: %v\n", f.Number, wireTypeString(f.WireType), f.Integer)
		}
	}

	// 2. ChatModelMetadata (inside Root Field 1)
	chatModelData := wire.FirstBytes(rootFields, 1)
	if len(chatModelData) == 0 {
		fmt.Println("\n⚠️  No ChatModelMetadata (Field 1) found in root envelope.")
		return
	}

	fmt.Printf("\n✨ [DECODED MESSAGE: exa.cortex_pb.ChatModelMetadata]\n")
	chatFields, err := wire.Decode(chatModelData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode ChatModelMetadata: %v\n", err)
		return
	}

	for _, cf := range chatFields {
		switch cf.Number {
		case 1:
			text := string(cf.Bytes)
			fmt.Printf("  ├── [1] system_prompt: %d characters (Preview: %q...)\n", utf8.RuneCountInString(text), truncateRunes(text, previewTextLimit))
		case 2:
			fmt.Printf("  ├── [2] message_prompts: [History Turn %d bytes]\n", len(cf.Bytes))
		case 3:
			fmt.Printf("  ├── [3] model: %d (internal enum)\n", cf.Integer)
		case 4:
			fmt.Printf("  ├── [4] usage: ModelUsage (%d bytes)\n", len(cf.Bytes))
			decodeModelUsage(cf.Bytes, "  │     ")
		case 5:
			fmt.Printf("  ├── [5] model_cost: (%d bytes)\n", len(cf.Bytes))
		case 7:
			fmt.Printf("  ├── [7] tool_choice: (%d bytes)\n", len(cf.Bytes))
		case 8:
			toolName := extractToolName(cf.Bytes)
			fmt.Printf("  ├── [8] tools: ToolDefinition (name=%q, %d bytes)\n", toolName, len(cf.Bytes))
		case 9:
			fmt.Printf("  ├── [9] chat_start_metadata: ChatStartMetadata (%d bytes)\n", len(cf.Bytes))
			decodeChatStartMetadata(cf.Bytes, "  │     ")
		case 11:
			dur := decodeDuration(cf.Bytes)
			fmt.Printf("  ├── [11] time_to_first_token: %s (TTFT: %d.%09d s)\n", formatDuration(dur), dur.seconds, dur.nanos)
		case 12:
			dur := decodeDuration(cf.Bytes)
			fmt.Printf("  ├── [12] streaming_duration: %s (Total: %d.%09d s)\n", formatDuration(dur), dur.seconds, dur.nanos)
		case 13:
			fmt.Printf("  ├── [13] credit_cost: %d\n", cf.Integer)
		case 14:
			fmt.Printf("  ├── [14] retries: %d\n", cf.Integer)
		case 15:
			fmt.Printf("  ├── [15] completion_config: (%d bytes)\n", len(cf.Bytes))
		case 16:
			fmt.Printf("  ├── [16] prompt_sections: (%d bytes)\n", len(cf.Bytes))
		case 17:
			fmt.Printf("  ├── [17] retry_infos: (%d bytes)\n", len(cf.Bytes))
		case 18:
			fmt.Printf("  ├── [18] consumed_credits: %d\n", cf.Integer)
		case 19:
			fmt.Printf("  ├── [19] response_model: %q\n", string(cf.Bytes))
		case 20:
			k, v := decodeMapEntry(cf.Bytes)
			fmt.Printf("  ├── [20] custom_metadata: %q => %q\n", k, v)
		default:
			fmt.Printf("  ├── [%d] (unknown field, wire=%d)\n", cf.Number, cf.WireType)
		}
	}
	fmt.Printf("================================================================================\n")
}

type protoDuration struct {
	seconds int64
	nanos   int32
}

func decodeDuration(data []byte) protoDuration {
	fields, err := wire.Decode(data)
	if err != nil {
		return protoDuration{}
	}
	sec, _ := wire.Varint(fields, 1)
	nan, _ := wire.Varint(fields, 2)
	return protoDuration{seconds: int64(sec), nanos: int32(nan)}
}

func formatDuration(d protoDuration) string {
	totalSec := float64(d.seconds) + float64(d.nanos)/1e9
	return fmt.Sprintf("%.3fs", totalSec)
}

func decodeModelUsage(data []byte, indent string) {
	fields, err := wire.Decode(data)
	if err != nil {
		return
	}
	for _, f := range fields {
		switch f.Number {
		case 1:
			fmt.Printf("%s├── [1] model_code: %d\n", indent, f.Integer)
		case 2:
			fmt.Printf("%s├── [2] uncached_prompt_tokens: %d\n", indent, f.Integer)
		case 3:
			fmt.Printf("%s├── [3] thinking_output_tokens: %d\n", indent, f.Integer)
		case 5:
			fmt.Printf("%s├── [5] cached_content_token_count: %d (CACHE HIT)\n", indent, f.Integer)
		case 6:
			fmt.Printf("%s├── [6] provider_tier: %d\n", indent, f.Integer)
		case 7:
			fmt.Printf("%s├── [7] bot_id: %q\n", indent, string(f.Bytes))
		case 8:
			k, v := decodeMapEntry(f.Bytes)
			fmt.Printf("%s├── [8] session_metadata: %q => %q\n", indent, k, v)
		case 9:
			fmt.Printf("%s├── [9] output_content_tokens: %d\n", indent, f.Integer)
		case 10:
			fmt.Printf("%s├── [10] reasoning_token_mirror: %d\n", indent, f.Integer)
		case 11:
			fmt.Printf("%s└── [11] upstream_request_id: %q\n", indent, string(f.Bytes))
		}
	}
}

func decodeChatStartMetadata(data []byte, indent string) {
	fields, err := wire.Decode(data)
	if err != nil {
		return
	}
	for _, f := range fields {
		if f.Number == 10 && f.WireType == 2 {
			subFields, _ := wire.Decode(f.Bytes)
			obs, _ := wire.Varint(subFields, 1)
			limit, _ := wire.Varint(subFields, 4)
			fmt.Printf("%s├── [10.1] context_window_observed_tokens: %d\n", indent, obs)
			fmt.Printf("%s└── [10.4] context_window_limit_tokens: %d\n", indent, limit)
		}
	}
}

func decodeMapEntry(data []byte) (string, string) {
	fields, err := wire.Decode(data)
	if err != nil {
		return "", ""
	}
	key := string(wire.FirstBytes(fields, 1))
	val := string(wire.FirstBytes(fields, 2))
	return key, val
}

func extractToolName(data []byte) string {
	fields, err := wire.Decode(data)
	if err != nil {
		return ""
	}
	return string(wire.FirstBytes(fields, 1))
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}

func formatBytesOrHex(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	return hex.EncodeToString(b)
}

func wireTypeString(wt int) string {
	switch wt {
	case 0:
		return "varint"
	case 1:
		return "fixed64"
	case 2:
		return "bytes"
	case 5:
		return "fixed32"
	default:
		return "unknown"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
