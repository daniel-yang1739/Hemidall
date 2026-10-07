# Local Forensic Commands

These commands are read-only diagnostics for artifacts created by Antigravity.
They never modify a transcript, SQLite database, WAL file, or provider state.

| Command | Primary source | Purpose |
| --- | --- | --- |
| `blob-wire-audit` | Multiple SQLite BLOB columns | Shows representative wire-level shapes without assigning an official schema. |
| `cortex-decode` | `gen_metadata.data` | Decodes ChatModelMetadata and ModelUsage into human-readable semantic fields. |
| `executor-metadata-audit` | `executor_metadata.data` | Lists observed execution IDs, model-like strings, and printable byte runs. |
| `persisted-usage-audit` | `gen_metadata.data` plus transcript timestamps | Lists schema-inferred input and cache counters in generation order. |
| `protowire-dump` | `gen_metadata.data` | Uses official google.golang.org/protobuf/encoding/protowire to dump raw wire format tree. |
| `session-model-audit` | `gen_metadata` and `executor_metadata` across a directory | Compares model observations per session. |

## Architecture

Commands own only flag parsing, output formatting, and report-specific aggregation.
All SQLite access, protobuf decoding, model resolution, and transcript parsing must
come from `internal/agent_adapters/antigravity`. A command must not import
`internal/legacy` or duplicate a provider parser.

## Evidence rules

Output is local observation, not a captured provider HTTP request or invoice.
Unknown protobuf field meanings remain labelled as wire observations. Model IDs are
accepted only from direct observed fields or explicit cross-artifact joins.
