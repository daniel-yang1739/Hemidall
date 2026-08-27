## 5 Dimensions of Context Anatomy (Track 2)
* **1. System Instruction** : Base system prompt, developer rules, invariant guidelines, and safety constraints that guide agent behavior.
* **2. MCP Tools Schema** : Function calling JSON schemas defining all available tools, parameter types, descriptions, and required fields.
* **3. Tool Results / Diff** : Payloads returned from tool executions (bash stdout/stderr, file reads, directory listings, code diffs).
* **4. Conversation Hist** : Past user-assistant dialogue turns and prior tool call summaries retained within the active context window.
* **5. Active Turn / CoT** : The latest turn user prompt + Model thinking (Chain of Thought / CoT reasoning) + Active tool call payload.

## Multi-Agent Architecture & Roles
* **👑 MAIN PLANNER** : Primary planning agent interacting directly with the human user and establishing master plans, maintaining its own context window and prefix cache.
* **👥 SUBAGENT WORKER** : Background autonomous worker spawned via invoke_subagent. Maintains an isolated context window, separate system prompt, and independent API quota; local tool steps consume 0 GPU tokens.
* **⚙️ INTERNAL HARNESS** : System-level background routines and initialization tasks (environment probes, config setup) executed automatically by the harness.
* **🛡️ BLOCKED / DENIED** : Security boundary interceptions. Triggered when accessing protected files (e.g. settings.json), user rejects sensitive commands, or task aborts (0 GPU tokens).

## Core Metrics & Ground Truth Billing (Track 1)
* **Total Active Context** : The exact mathematical token count sent to the LLM in the current HTTP request window (e.g. 185k / 256k limit).
* **Prefix Cache Hit** : Identical prefix tokens reused directly from GPU High-Bandwidth Memory (HBM) ($0.00 / 50-80% cost discount).
* **New Billable Tokens** : Uncached tokens in the current turn (new prompt + new tool results) requiring full GPU prefill matrix compute.
* **Raw Log Accumulated** : Total uncompressed tokens in local append-only history files (e.g. 1.7M tokens), before cloud sliding-window compaction or truncation.
* **TTL Cold Start** : Google GPU memory evicts idle KV cache after ~5 min; subsequent turns incur full cold-start prefill billing.

## Cache Status Badges & Billing Semantics
* **[CACHE HIT]** : Prefix Cache Hit Rate >= 80.0%. High-efficiency state where existing prompt prefix is fully reused from GPU HBM (75% cost discount, near-zero TTFT).
* **[PARTIAL HIT]** : Cache Hit Rate between 0.1% and 79.9%. Prefix cache matched existing history, but large new payloads (huge tool outputs, file diffs) diluted the hit ratio.
* **[CACHE WRITE]** : First-ever turn in session (0.0% hit). System instructions and tool definitions are computed and written to GPU KV cache memory.
* **[TTL EXPIRED]** : Idle timeout exceeded (> 5 minutes). GPU evicted previous KV cache to free HBM, requiring full cold-start re-prefill.
* **[CACHE MISS]** : Zero cache hit (0.0%). Cache invalidation occurred due to altered prefix headers or cross-model routing.

## Aggregate Metrics & Pricing Algorithms
* **Total Processed Volume** : Cumulative tokens processed across all cloud turns (Σ TotalTokens), measuring total compute throughput.
* **Cache Hit Volume** : Total tokens successfully served from GPU KV cache (Σ CachedTokens), representing the bulk of cost reduction.
* **Uncached Inbound Volume** : Total cold un-cached inbound tokens (Σ NewTokens) requiring full GPU prefill matrix compute.
* **Effective Tokens** : Discount-weighted billable equivalent tokens: Effective = Σ [Cached × (1 - Discount) + New], reflecting actual financial weight.
* **Cached Saved %** : Percentage of tokens saved via GPU prefix caching: Saved = Cached × Discount, measuring prompt caching efficiency.
* **Multi-Model Discount Matrix** : Gemini 3.7 Flash receives 75% cache discount; Gemini 2.5 Pro receives 75% discount; Claude 3.7 Sonnet receives 90% discount.
* **Google AI Pro Quota (5000 RPD)** : Daily allowance of 5,000 cloud inference requests under Google AI Pro plan, tracked as Turns / 5000 × 100%.

## Context Mechanics & Memory Physics
* **Context Compaction** : Asynchronous summarization triggered at 95% High Watermark (~245k tokens) to compact history down to 48% Low Watermark (~120k).
* **Reverse Sliding Window** : Algorithm allocating tokens backwards from the newest step to match Google official active budget, pruning ancient steps.
* **Longest Common Prefix** : LCP algorithm comparing sequential steps to determine exact byte-level shared prefix for KV cache reuse.

## Step Types & Agent Lifecycle
* **👤 USER_INPUT** : Natural language prompt from the human user, initiating a new reasoning turn. Telemetry is settled on the subsequent cloud turn.
* **🤖 MODEL_RESPONSE** : Cloud LLM natural language response containing chain-of-thought reasoning and synthesis, benefiting from prefix cache acceleration.
* **🛠️ TOOL_CALL** : Structural tool invocation instructions dispatched from cloud model to local machine (e.g. run_command, view_file, edit_file).
* **💻 OUTPUT** : Execution payloads from local operations (stdout/stderr, file buffers, diffs). Offline (0 tokens), staged for next cloud turn.
* **⚙️ CHECKPOINT** : Context Truncation & Compaction Checkpoint. Injected by Antigravity to compress long conversation transcripts, resetting prefix cache.
* **⚠️ ERROR_MESSAGE** : System errors, runtime network interruptions, command timeout, or process failure events.
