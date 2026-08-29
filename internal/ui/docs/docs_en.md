## Local 5-Dimension Context Anatomy (Track 2: Payload Analysis)
* **1. System Instruction** : Base system instructions, developer instructions, repository constitution (AGENTS.md), and safety boundaries.
  - **Physical Location** : Anchored at the absolute prefix (Offset 0) of GPU KV Cache VRAM throughout the conversation lifecycle.
  - **Window Ratio** : Typically represents 2% ~ 8% of the total window (~5k to ~20k Tokens) based on loaded skill extensions and role definitions.
  - **Cache Impact** : Enjoys 100% prefix reuse across turns, forming the foundational bedrock for high cache hit rates and low TTFT latency.

* **2. MCP Tools Schema** : JSON Schema definitions specifying tool invocation contracts.
  - **Schema Scope** : Details all available tools (e.g. view_file, run_command, replace_file_content), argument types, descriptions, and required properties.
  - **Execution Role** : Serves as the structural guidance for the model to emit function calls; transmitted on every cloud inference turn.
  - **Window Ratio** : Typically occupies 3% ~ 10% of the active context window (~8k to ~25k Tokens).

* **3. Tool Results / Diff** : Tangible output data returned from local subprocess executions.
  - **Payload Sources** : Terminal stdout/stderr outputs, file read buffers, directory tree listings, and git code diffs.
  - **Growth Dynamic** : Expands rapidly as tools are executed, representing the primary consumer of context tokens (60% ~ 85% of total window).
  - **Billing Nature** : Billed at 0 GPU Tokens upon local execution; batched and billed as uncached/prefill tokens during the subsequent cloud turn.

* **4. Conversation Hist** : Preserved history of past user prompts, model responses, and tool invocation summaries within the active window.
  - **Window Retention** : Maintained within the 256k physical window budget via the Reverse Sliding Window algorithm.
  - **Window Ratio** : Represents approximately 10% ~ 25% of total active context.

* **5. Active Turn / CoT** : Transient payload of the ongoing conversational turn.
  - **Components** : Latest user prompt + model Chain-of-Thought (CoT) reasoning + active tool call instructions.
  - **Lifecycle** : Settles into permanent conversation history upon completion of the turn.

## Track 1 Panel Title Variations & Scenarios
* **☁️ TRACK 1: OFFICIAL CLOUD TELEMETRY** : Main Planner cloud inference with verified official billing.
  - **Trigger Scenario** : Main Planner executes model reasoning, and SQLite generation metadata successfully records official Google Protobuf token usage.
  - **Key Metrics** : Total Context Window (e.g. 160k), Step Delta (new inbound tokens), Prefix Cache Savings (hit rate & 75% cost discount), Turn Financial Cost ($ USD / NT$).
  - **Physical Role** : The exclusive step producing actual cloud API billing and GPU VRAM throughput.

* **👥 TRACK 1: SUBAGENT CLOUD TELEMETRY** : Parallel subagent worker cloud inference.
  - **Trigger Scenario** : Independent background subagent spawned via `invoke_subagent` executes its own cloud inference turn.
  - **Key Metrics** : Independent context window, independent cache hit rate, isolated billing and token quotas.
  - **Architecture Role** : Maintains an isolated System Prompt and execution workspace, with billing tracked separately from the Main Agent.

* **👤 TRACK 1: USER INTERACTION (CLIENT PROMPT)** : Natural language prompt submitted by human engineer.
  - **Trigger Scenario** : User submits a prompt in the terminal or selects an option in the clarification modal.
  - **Key Metrics** : Total Context is Nil (pre-inference staging), Step Delta (prompt token length), status marked as `Staged locally ⏳`.
  - **Physical Role** : Inbound intent staged in local queue prior to GPU dispatch; holds no active context payload.

* **💻 TRACK 1: LOCAL EXECUTION STEP (OFFLINE)** : Local tool execution on host Mac.
  - **Trigger Scenario** : Local tool execution (`run_command` terminal execution, `view_file` read, `replace_file_content` edit).
  - **Key Metrics** : Total Context is Nil (local machine subprocess, 0 GPU tokens billed), Step Delta (local BPE buffer), status marked as `Staged for Next Cloud Turn`.
  - **Physical Role** : Offline local execution consuming 0 API tokens; output is staged for batch ingestion on the next cloud turn.

* **🛡️ TRACK 1: PERMISSION BOUNDARY (BLOCKED)** : Security guardrail intercepting unauthorized actions.
  - **Trigger Scenario** : Agent attempts to access protected configurations, or user cancels dangerous shell commands.
  - **Key Metrics** : Total Context is Nil, 0 GPU Tokens Billed, `Blocked by System Permission Guard 🛡️`.
  - **Protection Role** : Intercepts commands locally without network dispatch, ensuring zero token waste and complete security.

* **🦿 TRACK 1: INTERNAL HARNESS BACKGROUND TASK** : Host harness automated background event.
  - **Trigger Scenario** : Background async task completion `<SYSTEM_MESSAGE>`, timer wake-up `schedule`, linter / IDE diagnostics.
  - **Key Metrics** : Total Context is Nil, local coordination event, packaged to trigger a Reactive Wakeup for the LLM on the next turn.
  - **Coordination Role** : Formats async events into durable transcript steps for subsequent model consumption.

* **⚙️ TRACK 1: SYSTEM COMPACTION (CHECKPOINT)** : Sidecar context garbage collection checkpoint.
  - **Trigger Scenario** : Context reaches 256k physical window limit; Antigravity triggers sidecar context GC, compressing 200k+ tokens into a ~10k checkpoint summary.
  - **Key Metrics** : Summary Size, Step Delta (summary payload), `Prepend Checkpoint ➔ Re-anchors Active Window Base 🔄`.
  - **Memory Role** : Condenses aging history into concise checkpoints, re-anchoring the window base for perpetual conversation.

## Multi-Agent Architecture & Roles
* **👑 MAIN PLANNER** : Primary orchestrator directly interacting with the user, formulating top-level plans with an independent sliding window and prefix cache.
* **👥 SUBAGENT WORKER** : Parallel worker process dispatched via `invoke_subagent`, maintaining isolated context windows and independent API quotas.
* **⚙️ INTERNAL HARNESS** : Internal background runtime harness for lifecycle orchestration, event streaming, and process management.
* **🛡️ BLOCKED / DENIED** : Security boundary interceptor preventing unauthorized file reads or command execution with 0 GPU token cost.

## Step Types & Agent Lifecycle
* **👤 USER_INPUT** : Natural language prompt from human client, marking the start of a conversation turn. Staged locally (Total Context: Nil).
* **🤖 MODEL_RESPONSE** : Cloud LLM Chain-of-Thought reasoning and structured response with prefix cache acceleration.
* **🛠️ TOOL_CALL** : Tool invocation instruction emitted by model to local host environment.
* **💻 TOOL_RESULT / OUTPUT** : Execution feedback from local tools (stdout/stderr, file contents, diffs). 0 GPU tokens, staged for subsequent turn.
* **🦿 SYSTEM_MESSAGE** : Background notification from host harness (async task completions, timer signals, linter feedback) triggering reactive wakeups.
* **📜 SYSTEM_INIT** : Session boot and environment initialization injecting agent identity and repository rules.
* **⚙️ CHECKPOINT** : Truncation checkpoint created during sidecar context compaction.
* **⚠️ ERROR_MESSAGE** : System exceptions or runtime failures (network aborts, timeout errors).

## Billing Ground Truth (Track 1 Metrics)
* **Total Context Window** : Exact total tokens delivered to cloud LLM in current HTTP payload.
* **Step Delta** : Local token differential generated exclusively within this single step.
* **Prefix Cache Hit** : Reused prefix tokens stored directly in cloud GPU HBM (70%~75% cost discount).
* **New Billable Tokens** : Uncached cold tokens requiring full prefill compute on GPU.
* **Raw Log Accumulated** : Cumulative uncompressed tokens in append-only disk transcript.
* **TTL Cold Start** : GPU HBM cache eviction after ~5 min idle timeout, causing full cold-start recalculation.

## Cache Status Badges & Semantics
* **[CACHE HIT]** : Cache hit rate >= 80.0%. Optimal performance state leveraging GPU HBM.
* **[PARTIAL HIT]** : Hit rate between 0.1% and 79.9%. Prefix hit combined with large cold inbound tools/diffs.
* **[CACHE WRITE]** : Step 0 initialization (0.0% hit). System prompt and tools written to GPU KV Cache.
* **[TTL EXPIRED]** : Idle timeout (>5 min). GPU cache evicted; turn requires cold-start recalculation.
* **[CACHE MISS]** : Cache miss (0.0%). Broken prefix chain due to prompt mutation or model routing.

## Aggregate Metrics & Pricing Algorithms
* **Total Processed** : Cumulative tokens processed across all cloud turns (Σ TotalTokens).
* **Cache Hit Volume** : Cumulative GPU KV Cache hit volume (Σ CachedTokens).
* **Uncached Inbound** : Cumulative uncached inbound prefill tokens (Σ NewTokens).
* **Effective Tokens** : Discount-weighted effective tokens: Effective = Σ [Cached × (1 - Discount) + New].
* **Cached Saved %** : Percentage of tokens saved via prefix caching: Saved = Cached × Discount.
* **Multi-Model Discount Matrix** : Gemini 3.7 Flash: 75% discount; Gemini 2.5 Pro: 75% discount; Claude 3.7 Sonnet: 90% discount.
* **Google AI Pro Quota (5000 RPD)** : Daily quota percentage computed as Turns / 5000 × 100%.

## Context Mechanics & Physics
* **Context Compaction** : Dual-watermark compaction triggering recursive summarization at 95% High Watermark (~245k) down to 48% Low Watermark (~120k).
* **Reverse Sliding Window** : Reverse sliding window algorithm filling official Google token budget from newest to oldest.
* **Longest Common Prefix** : Exact character-level prefix matching algorithm identifying GPU KV Cache reusable boundaries.
