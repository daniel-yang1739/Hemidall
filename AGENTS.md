# 📜 REPOSITORY CONSTITUTION

> **Status**: ACTIVE & IMMUTABLE (Highest Authority)
> **Scope**: All AI Agents, Subagents, and Pair-Programming Sessions

---

## 🌐 ARTICLE I: LANGUAGE POLICY

1. **Codebase & Engineering Artifacts (STRICTLY ENGLISH ONLY)**:
   - All source code, internal packages, structs, variables, function signatures, and tests in `agent-observer/` (and all non-docs directories) **MUST BE 100% IN ENGLISH**.
   - **Repository Root Rules & Configs (`AGENTS.md`, configs)**: Strictly English only. Zero Chinese allowed outside `docs/`.
   - **Code Comments**: English only. Zero non-English comments allowed in `.go`, `.py`, `Makefile`, or configuration files.
   - **Terminal & System Logs**: English only (`fmt.Printf`, `log.Println`, CLI banners, error messages).

2. **Documentation (`docs/` - HUMAN-FACING KNOWLEDGE HUB)**:
   - The `docs/` directory (`02_wiki/`, `ithome_draft/`, `ithome_ready/`, `reviews/`) is designated for human reading, study, and publication.
   - Traditional Chinese (繁體中文) is the primary language for `docs/` technical articles, analogies, and explanations.

---

## 💎 ARTICLE II: OBSIDIAN VAULT VERSION CONTROL (WHITELIST POLICY)

1. **Strict Whitelist for `.obsidian/`**:
   - By default, ALL internal Obsidian runtime files (`workspace.json`, `graph.json`, `cache/`, `themes/`, `plugins/`) are **STRICTLY IGNORED**.
   - **ONLY the following 4 explicit core configuration files are allowed to be tracked in Git**:
     - `docs/.obsidian/app.json` (Editor settings, full-width mode)
     - `docs/.obsidian/appearance.json` (Theme selector, font preferences)
     - `docs/.obsidian/core-plugins.json` (Enabled core plugins)
     - `docs/.obsidian/community-plugins.json` (Enabled community plugin registry)

---

## 🧠 ARTICLE III: WIKI DISTILLATION & KNOWLEDGE MANAGEMENT

1. **Wiki is the Destination, Not an Intermediate Station**:
   - `docs/02_wiki/` is the permanent, highest-quality, fully refined technical asset.
   - `docs/ithome_draft/` and `docs/ithome_ready/` are decoupled writing project workspaces.
2. **Digest & Delete Policy (`01_raw/`)**:
   - `docs/01_raw/` is a temporary intake pool. Once raw materials are 100% distilled and verified into `docs/02_wiki/`, the raw files must be deleted immediately.
3. **Mandatory Step 0 (Pre-Flight Feedbacks Inspection)**:
   - Before writing or distilling, agents must inspect `.agents/skills/wiki-distiller/feedbacks/` for all `🟢 [ACCEPTED]` standards and `🔴 [REJECTED]` boundaries.
4. **Zero-Persona Contamination in Wiki**:
   - Reviewer names belong strictly to `.agents/skills/wiki-distiller/roles/` and `docs/reviews/`. Wiki body text must remain 100% objective, authoritative, and persona-free.
5. **Mandatory 4-Dimension Diagram Walkthrough**:
   - Every Mermaid diagram must be followed by a walkthrough detailing Core View, Step-by-Step path, Color/Physical semantics, and Underlying engineering details.
6. **Incremental Micro-Batch Carpet Distillation (Strict Prohibition of Big-Bang Distillation)**:
   - Raw materials must be processed incrementally in single files or small coherent micro-batches (1 to 3 files at a time).
   - Each batch must complete the full lifecycle: thorough reading -> codebase truth verification -> wiki card drafting/updating -> cross-verification -> Digest & Delete -> MOC sync, before moving to the next batch or file.
   - Attempting to ingest or process all raw files in a single pass ("Big-Bang Distillation") is STRICTLY PROHIBITED, as it inevitably leads to hallucinated abstractions, loss of critical forensics/engineering details, and superficial coverage.

---

## 🧪 ARTICLE IV: TESTING & GROUND TRUTH CONSTITUTION

1. **Full Confusion Matrix Coverage (TP / TN / FP / FN Matrix)**:
   - All logic changes must cover both Positive (TP, TN) and Negative/Boundary (FP, FN) scenarios to prevent regression.
2. **High-Signal Logic & Data Flow Focus (Reject Trivial Tests)**:
   - Focus exclusively on: Core business logic, data flow integrity, UI state machine transitions, data conversion, protobuf parsing, and cross-module wiring.
   - Avoid low-value, over-detailed trivial checks. Tests must serve as high-signal alarms that fail whenever production logic deviates from the correct path.
3. **Expectations Are Inviolable (Business Expectations First, No Assertion Weakening or Code Pollution)**:
   - Real business expectations are immutable. Never modify test assertions to make a test pass against flawed logic.
   - Never compromise production codebase architecture for testing convenience (e.g., adding dummy function parameters or returns solely for test hooks).
4. **Restricted Control Flow in Unit Tests (Table-Driven Loops and Assertions Only)**:
   - Table-driven tests **MAY use `for` loops** to execute a declarative list or map of test cases, preferably with explicit `t.Run` subtests. Test cases must define their inputs and expected outputs directly.
   - `if` statements **MAY be used for assertions and failure reporting only**, such as comparing actual and expected values before calling `t.Errorf`, `t.Fatalf`, or an equivalent assertion helper.
   - Tests **MUST NOT use conditional logic to calculate, alter, or select expected results**, reproduce production branching, or make the asserted business expectation depend on the implementation result.
   - Other behavioral control flow in test execution bodies, including non-table-driven loops and `switch` statements, is prohibited. Extract setup mechanics into test helpers when necessary, while keeping each test case's expectation explicit and independently reviewable.

---

## 🧱 ARTICLE V: CONSTANT & CONFIGURATION HYGIENE (MAGIC NUMBER & SCOPING DISCIPLINE)

1. **Strict Prohibition of Raw Magic Literals**:
   - Numerical values, pricing rates, token limits, thresholds, and buffer sizes must never appear as anonymous bare literals in execution bodies. Every number must have an explicit, semantic name.
2. **Scoping & Single Source of Truth Hierarchy**:
   - **File-Local Constants**: If a constant is strictly and exclusively used within a single file (e.g., file-specific UI timer or buffer size), define it as a file-scoped constant in that specific file. Do not pollute the global namespace.
   - **Cross-File / Shared Constants**: If a constant is shared across multiple files/packages, or is foreseeable to be shared across adapters/core/ui (e.g., token units, exchange rates, quota thresholds, baseline tokens), it **MUST** be defined in `internal/core/constants.go` or `internal/core/model_specs.go`.
   - **Zero Duplicate Definitions**: A shared constant must have exactly one authoritative definition. Never duplicate constant literals across multiple files.
3. **Dynamic Model Specifications & User Settings Decoupling**:
   - Official vendor limits (1M/2M limits, pricing per million, caching discount rates) belong to `internal/core/model_specs.go` and are resolved dynamically by model name.
   - User preferences and host limits (custom window, RPD tier, exchange rate) belong to `internal/core/config.go` and are loaded from host settings files (`settings.json`, `opencode.json`).
