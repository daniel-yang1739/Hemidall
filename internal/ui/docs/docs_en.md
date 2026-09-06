## System Architecture & Database Overview

* **Antigravity Storage Architecture** : Each conversation session is persisted to `~/.gemini/antigravity-cli/conversations/<UUID>.db`, comprising 7 core tables and official `exa.cortex_pb` Protobuf structures.
  * **`trajectory_meta`** : Root session identity (Trajectory ID, Cascade ID, source, trajectory type).
  * **`trajectory_metadata_blob`** : Local workspace path, Git Remote, branch, and environment fingerprint (`CortexTrajectoryMetadata`).
  * **`executor_metadata`** : File modification airbag; stores before/after file Diff snapshots to support atomic rollback on user interruption (`Ctrl+C`).
  * **`gen_metadata`** : Cloud inference telemetry receipts (token usage, TTFT latency, Vertex Trace ID, and active context snapshot).
  * **`steps`** : Timeline core table with 6 BLOB columns (`metadata`, `step_payload`, `render_info`, `permissions`, `task_details`, `error_details`).
  * **`parent_references`** : Subagent derivation topology (parent conversation ID, parent step index, workspace isolation mode).
  * **`battle_mode_infos`** : Multi-model A/B evaluation forks and winning conversation telemetry.

## System Prompt Historical Reconstruction (Solution 1: Snapshot Baseline)

* **Snapshot Rolling Lifecycle** : To prevent SQLite database bloat (avoiding 400MB+ per session), Antigravity rolls historical snapshots.
  * **Historical Entries (`idx < MAX`)** : Compacted into ~1.1 KB receipts, retaining token usage, TTFT, duration, and step boundary, while `system_prompt` and `tools` are physically overwritten.
  * **Latest Entry (`idx = MAX`)** : Persisted as a full 400KB~850KB active snapshot containing the 25k-word System Prompt, 17 Tool Schemas, and conversation history.
* **Solution 1: Global Snapshot Baseline Reconstruction** :
  * **Semantic Invariance** : In a given workspace session, System Prompt (project rules `AGENTS.md`, system guidelines, tool schemas) remains 99.9% invariant across all turns.
  * **Heimdall Strategy** : Heimdall extracts the authoritative System Prompt from the latest `MAX(idx)` snapshot as the Global Baseline, projecting it backwards across historical turns.
  * **Honest Telemetry Policy** : The UI clearly indicates: "Historical full context compacted by client; currently displaying baseline inherited from active snapshot."

## Dual-Layer Token Architecture

* **Dual-Layer Data Flow** : Two complementary token pipelines exist in SQLite for audit integrity and instantaneous UI rendering.
  * **`gen_metadata` (Cloud Generation Telemetry Layer)** : One authoritative receipt per cloud generation recording uncached input (`F4.2`), cached input (`F4.5`), thinking tokens (`F4.3`), and Vertex Trace ID (`req_vrtx_*`).
  * **`steps.metadata` (UI Instantaneous Badge Layer)** : When a step is `PLANNER_RESPONSE`, the system projects the exact `ModelUsage` directly into `metadata.model_usage`.
  * **Zero-Latency UI Benefit** : The UI renders timeline badges ("⚡ 17.5k cached / 409 thinking") instantly without requiring cross-table SQL JOINs.

## Prompt Caching Lifecycle & Economics

* **Prefix Caching & Pricing Dynamics** :
  * **Strict Prefix Trie** : Rooted in Transformer causal attention masks, caching matches continuously from token 0; any intermediate modification invalidates downstream cache.
  * **Pricing Model** : 1.25x surcharge on cache creation / write, 90% discount (0.1x) on cache reads / hits; reusing a prefix 2+ times yields massive cost reductions.
* **Why Turn 0 Omits User Prompt from Cache** :
  * **Global Prefix Sharing** : `System Rules + Tools` (17,535 tokens) are identical across all workspace sessions and subagents; anchoring at tool end guarantees instant cache hits.
  * **Interruption & Rollback Protection** : Turn 0 user prompts are mutable and interruptible (`Ctrl+C`); caching uncommitted turns would trigger prefix hash mismatches and cache purges.
  * **Rolling Prefix Expansion** : Once a turn is finalized, conversation history is committed into the cache prefix, escalating cache hit rates past 99% in extended sessions.

## Latency & Performance Telemetry

* **TTFT & Generation Speed Analytics** :
  * **`time_to_first_token` (TTFT, `F11`)** : Latency to first token, standard `google.protobuf.Duration` `{1: seconds, 2: nanos}`, consistently measured at 1.2s ~ 2.5s.
  * **`streaming_duration` (`F12`)** : Total streaming generation time, linearly proportional to output length and thinking tokens (~50-70 tokens/sec).
  * **Upstream Trace ID (`F4.11`)** : Every generation is stamped with a Google Cloud Vertex AI request ID (`req_vrtx_*`) for cloud audit reconciliation.

## UI Views Guide (Dashboard & Context)

* **Dashboard View** :
  * **Track 1 (Cloud Telemetry)** : Displays authoritative token billing, TTFT, and duration linked to the selected model event.
  * **Track 2 (Timeline Evidence)** : Reconstructs readable System, Tools, History, and Inbound prompt evidence visible at each step.
  * **USD Cost Estimation** : Computes real-time inference cost and cache savings based on official vendor pricing for uncached input, cached input, and output tokens.
* **Context View** :
  * **Five-Part Section Display** : System & Rules, Tool Schemas, Active History, Compacted Checkpoints, and Latest Inbound Prompt.
  * **Raw Evidence Mode** : Provides structured JSON evidence tagged with data sources, faithfully reflecting underlying Protobuf fields.
