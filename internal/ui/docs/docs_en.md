## What Heimdall Can Prove

Heimdall observes local Antigravity files. It does not capture an HTTP request and does not know provider billing or KV-cache behaviour unless those values are present in the persisted data.

* **Transcript**: `transcript_full.jsonl` supplies ordered local events: user input, model responses, tool calls, local tool output, and checkpoint-like records.
* **Generation metadata**: `conversations/<session>.db`, table `gen_metadata`, supplies protobuf blobs. Heimdall decodes the observed `last_step_index`, observed context, context limit, and a paired metered-input / cached-content usage message.
* **Persisted context snapshot**: Heimdall selects a `gen_metadata` blob above its safety threshold, newest-first, only when wire path `1.1` decodes as a system prompt. It can contain the system prompt, repeated context entries, and tool declarations. It is evidence of a saved context state, not a serialized provider request body.

## How a Generation Record Is Linked

The decoded `last_step_index` is an **input boundary**: the last transcript step included before a model generation. The generated model event is therefore the following transcript step.

`input boundary step N` → `generated transcript step N + 1`

This is a fixed relationship observed in Antigravity's persisted records. Heimdall does not use a nearest-step or plus/minus-one search. If either side is missing, the usage is shown as unavailable.

## Token Numbers Have Two Separate Sources

### Persisted usage and context observation

For a linked generation record, Heimdall can display:

* **Observed context tokens**: the decoded stored context value for that generation.
* **Observed context limit**: only when the protobuf field is present.
* **Metered input tokens** and **cached-content tokens**: decoded from the same observed nested usage-message path. Heimdall uses these paired counters for the Dashboard's cache-adjusted input estimate.

The observed-context value uses a separate inferred wire path and is used for the current context-window display only. Heimdall does **not** subtract it from cached content. The Dashboard's effective-input formula is `metered input + cached content × the exact configured model cache-price ratio`. It is a price-equivalent projection, not an Antigravity invoice. A missing serialized cache scalar is interpreted as proto3's default integer value, zero; the UI also reports how many records encoded a non-default cache scalar explicitly.

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
* **Session aggregates** sum paired metered-input and cached-content counters across persisted generations. The Dashboard calculates each known model's effective input using that model's configured cache-price ratio, then presents a model-weighted aggregate. These are useful efficiency projections, but not an API bill or a count of unique tokens.

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

## Cache-adjusted input projection

The dashboard shows total processed input, cached-content volume, metered input, effective input, and cache savings. The effective-input projection is calculated separately for each exact model ID from its configured official cache-input price ratio. Records that omit the proto3 cache scalar contribute zero cached tokens; the dashboard separately displays the number of explicit scalars for inspection. This is not a provider invoice, a unique-token count, a cache TTL signal, or a cache-write signal.

## Supported Source

The current product discovers and watches **Antigravity** sessions only. The session switcher lists Antigravity conversations from the local `.gemini/antigravity-cli` installation. Other agent products are not represented as supported sources.
