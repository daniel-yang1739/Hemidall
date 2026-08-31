## What Heimdall Can Prove

Heimdall observes local Antigravity files. It does not capture an HTTP request and does not know provider billing or KV-cache behaviour unless those values are present in the persisted data.

* **Transcript**: `transcript_full.jsonl` supplies ordered local events: user input, model responses, tool calls, local tool output, and checkpoint-like records.
* **Generation metadata**: `conversations/<session>.db`, table `gen_metadata`, supplies protobuf blobs. Heimdall decodes the observed `last_step_index`, total-context, context-limit, and selected cache fields where they are present.
* **Persisted context snapshot**: Heimdall selects a `gen_metadata` blob above its safety threshold, newest-first, only when wire path `1.1` decodes as a system prompt. It can contain the system prompt, repeated context entries, and tool declarations. It is evidence of a saved context state, not a serialized provider request body.

## How a Generation Record Is Linked

The decoded `last_step_index` is an **input boundary**: the last transcript step included before a model generation. The generated model event is therefore the following transcript step.

`input boundary step N` → `generated transcript step N + 1`

This is a fixed relationship observed in Antigravity's persisted records. Heimdall does not use a nearest-step or plus/minus-one search. If either side is missing, the usage is shown as unavailable.

## Token Numbers Have Two Separate Sources

### Persisted context observation

For a linked generation record, Heimdall can display:

* **Observed context tokens**: the decoded stored total for that generation.
* **Observed context limit**: only when the protobuf field is present.
* **Cache fields**: only when both decoded values are internally valid. No cache hit rate or uncached value is manufactured from prior turns, idle time, or a model name.

These fields are schema-inferred local observations. They are not an official API response, token invoice, price, or cache guarantee.

### Local transcript estimate

`cl100k_base` counts text found in an individual transcript event. Heimdall uses it for a local event delta and an append-only transcript-content total.

The append-only total is **not** the next request's context. It can be far larger than the active context because old transcript rows may have been compacted, excluded, or represented differently in the persisted snapshot.

## Visible context evidence estimate

Track 2 keeps a five-part table of **readable context evidence for the selected playback step**. Heimdall prepares one local transcript estimate for every observed event when history changes, so moving the Dashboard cursor does not query SQLite or recount the whole transcript. For a cloud-generation event, the estimate uses evidence observed immediately before that generation; for every other event, it uses evidence observed through that event. It is not a reconstructed provider request:

* system instruction;
* tool definitions;
* staged tool buffers;
* history evidence; and
* the latest inbound prompt observed in the transcript.

Each value is a local `cl100k_base` estimate. If the selected step exactly matches the generated step linked by the newest persisted snapshot, Track 2 replaces system, tools, and history with those snapshot values; its selected-step buffers and inbound remain transcript observations. Without such a snapshot match, all five dimensions stay available and are labelled `transcript`. The five values make visible evidence and its source easier to inspect, but they are not a reconstructed provider request and must not be compared numerically with Track 1's persisted context total.

## Dashboard

* **Track 1 — Persisted Cloud Usage Observation** shows only the generation metadata linked to the selected model event.
* **Track 2 — Playback Context Evidence** follows the selected event. It uses a cached local transcript estimate for that step, and uses persisted snapshot dimensions only when the snapshot maps exactly to that generated step. When `last_step_index` is available, the match is `last_step_index + 1`. Otherwise the step remains a clearly labelled transcript estimate. Moving playback changes both Track 1 and Track 2 without SQLite I/O.
* **Session aggregates** sum observed context values across persisted generations. They are useful for comparison, but are not an API bill or a count of unique tokens.

## Context View

The Context view does not choose only one source. It first builds a transcript baseline for what happened in the session, then overlays only the fields actually saved in a snapshot: system text, rules, skills, tools, and persisted records. Runtime metadata remains local to the Heimdall process. Raw mode is Heimdall-generated evidence JSON with per-field source labels, not Antigravity's original request JSON.

The Context view distinguishes these categories:

* **System and rules**: text decoded from the persisted snapshot when available.
* **Tools**: repeated persisted tool entries when available; otherwise names and argument keys observed in the transcript.
* **Compacted checkpoint**: the decoded `<CONTEXT_SUMMARY>` record, if one is persisted.
* **Active history**: persisted context records in the snapshot, ordered by their saved sequence. Selecting one renders only that record; raw mode does not silently show unrelated records.
* **Latest inbound and staged buffers**: transcript observations, not proof of an outbound HTTP payload.
* **MCP**: only MCP-related text visible in the snapshot system prompt is shown. The Context view does not read `mcp_config.json`, `settings.json`, or project configuration, and cannot attribute a tool to a specific MCP server.

The `field 2` count is an occurrence count of a repeated wire field. Heimdall safely displays readable text and wire observations from some records, but without an official protobuf schema it cannot assign an official role or complete meaning to every record.

## Cache Status

`HIT`, `PARTIAL`, and `MISS` can appear only when a linked record has a valid decoded total and cache field. `UNKNOWN` means that the stored data is absent or not comparable. Heimdall does not infer startup writes, TTL expiry, invoices, storage fees, or output charges from local evidence.

## Cache-adjusted input projection

* **Effective Input**: `uncached + cached × cache-input price ratio`, calculated only for the rows with comparable total/cache fields and only for an exact model ID with a verified official pricing profile.
* **Gemini 3.7 Flash profile**: Google Gemini Developer API paid standard pricing currently lists `$0.75/M` standard input and `$0.075/M` cached input, so its cache-input price ratio is `0.10×`.
* **Scope**: this is a per-model standard-input-price equivalent. It is not a reconstructed request, cross-model total, Antigravity invoice, storage fee, output charge, or proof that Antigravity uses the Google API Standard tier.

## Supported Source

The current product discovers and watches **Antigravity** sessions only. The session switcher lists Antigravity conversations from the local `.gemini/antigravity-cli` installation. Other agent products are not represented as supported sources.
