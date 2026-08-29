## Local 5-Dimension Context Anatomy (Track 2: Payload Analysis)
* **1. System Instruction** : Base system instructions, developer instructions, repository constitution (AGENTS.md), and safety limits. Anchored at the absolute prefix of GPU VRAM.
* **2. MCP Tools Schema** : JSON Schema definitions for function calling, detailing available tools, parameter types, descriptions, and required constraints.
* **3. Tool Results / Diff** : Real outputs from local tool executions (stdout/stderr, file reads, directory listings, git diffs). The primary contributor to context expansion.
* **4. Conversation Hist** : Preserved history of past user prompts, model responses, and tool invocation summaries within the active sliding window.
* **5. Active Turn / CoT** : Latest user prompt in the active turn + model Chain-of-Thought (CoT) reasoning + active tool call payload.

## Track 1 Panel Title Variations & Scenarios
* **☁️ TRACK 1: OFFICIAL CLOUD TELEMETRY** :
  - **Trigger Scenario** : Main Planner initiates cloud inference (StepTypeModelResponse), and Google official telemetry is written to SQLite generation metadata.
  - **Key Metrics** : Total Context Window (e.g. 160k), Step Delta (new inbound tokens), Prefix Cache Savings (hit rate & 75% cost discount), Turn Financial Cost ($ USD / NT$).
* **👥 TRACK 1: SUBAGENT CLOUD TELEMETRY** :
  - **Trigger Scenario** : Autonomous parallel subagent spawned via `invoke_subagent` executes its own cloud inference turn.
  - **Key Metrics** : Independent context window, independent cache hit rate, isolated billing and token quotas.
* **👤 TRACK 1: USER INTERACTION (CLIENT PROMPT)** :
  - **Trigger Scenario** : Human engineer submits a prompt (task command or clarification choice).
  - **Key Metrics** : Total Context is Nil (pre-inference staging), Step Delta (prompt token length), status marked as `Staged locally (Awaiting Next Cloud Inference Turn ⏳)`.
* **💻 TRACK 1: LOCAL EXECUTION STEP (OFFLINE)** :
  - **Trigger Scenario** : Local tool execution on host Mac (`run_command`, `view_file`, `replace_file_content`), generating subprocess outputs.
  - **Key Metrics** : Total Context is Nil (local machine subprocess, 0 GPU tokens billed), Step Delta (local BPE buffer), status marked as `Staged for Next Cloud Turn`.
* **🛡️ TRACK 1: PERMISSION BOUNDARY (BLOCKED)** :
  - **Trigger Scenario** : Security guardrail interception (access to sensitive settings, user rejection of dangerous shell commands).
  - **Key Metrics** : Total Context is Nil, 0 GPU Tokens Billed, `Blocked by System Permission Guard 🛡️`.
* **🦿 TRACK 1: INTERNAL HARNESS BACKGROUND TASK** :
  - **Trigger Scenario** : Host harness automated background event (asynchronous background task completion `<SYSTEM_MESSAGE>`, timer wake-up `schedule`, linter / IDE diagnostics).
  - **Key Metrics** : Total Context is Nil, local coordination event, packaged to wake up LLM on the subsequent cloud turn (Reactive Wakeup).
* **⚙️ TRACK 1: SYSTEM COMPACTION (CHECKPOINT)** :
  - **Trigger Scenario** : Context reaches 256k physical window limit; Antigravity triggers sidecar context GC, compressing 200k+ tokens into a ~10k checkpoint summary.
  - **Key Metrics** : Summary Size, Step Delta (summary payload), `Prepend Checkpoint ➔ Re-anchors Active Window Base 🔄`.

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
