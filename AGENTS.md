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
4. **Zero Control Flow in Unit Tests (No Logic Statements in Test Bodies)**:
   - Unit tests **MUST NOT contain control flow logic (`for-loop`, `if-else`, `switch-case`)** inside test execution bodies.
   - Control flow inside tests creates code that itself requires testing. When testing multiple scenarios, decompose into distinct, flat, independent test functions or subtests with explicit, direct assertions.
