# 📜 REPOSITORY CONSTITUTION (專案開發與協同憲法)

> **Status**: ACTIVE & IMMUTABLE (最高約束力)  
> **Scope**: All AI Agents, Subagents, and Pair-Programming Sessions

---

## 🌐 ARTICLE I: LANGUAGE POLICY (語言憲法 - 核心鐵律)

1. **Codebase & Engineering Artifacts (STRICTLY ENGLISH ONLY)**:
   - All source code, internal packages, structs, variables, function signatures, and tests in `agent-observer/` (or any non-docs directory) **MUST BE 100% IN ENGLISH**.
   - **Code Comments**: English only. Zero Chinese comments allowed in `.go`, `.py`, `Makefile`, or config files.
   - **Terminal & System Logs**: English only (`fmt.Printf`, `log.Println`, CLI banners, error messages).
   
2. **Documentation (`docs/` - HUMAN-FACING KNOWLEDGE HUB)**:
   - The `docs/` directory (`02_wiki/`, `ithome_draft/`, `ithome_ready/`, `reviews/`) is designed for human reading, study, and publication.
   - Traditional Chinese (繁體中文) is the primary language for `docs/` technical articles, analogies, and explanations.

---

## 💎 ARTICLE II: OBSIDIAN VAULT VERSION CONTROL (白名單憲法)

1. **Strict Whitelist for `.obsidian/`**:
   - By default, ALL internal Obsidian runtime files (`workspace.json`, `graph.json`, `cache/`, `themes/`, `plugins/`) are **STRICTLY IGNORED**.
   - **ONLY the following 4 explicit core configuration files are allowed to be tracked in Git**:
     - `docs/.obsidian/app.json` (Editor settings, full-width mode)
     - `docs/.obsidian/appearance.json` (Theme selector, font preferences)
     - `docs/.obsidian/core-plugins.json` (Enabled core plugins)
     - `docs/.obsidian/community-plugins.json` (Enabled community plugin registry)

---

## 🧠 ARTICLE III: WIKI DISTILLATION & KM CONSTITUTION (知識庫憲法)

1. **Wiki is the Destination, Not an Intermediate Station**:
   - `docs/02_wiki/` is the permanent, highest-quality, fully refined technical asset.
   - `docs/ithome_draft/` and `docs/ithome_ready/` are decoupled writing project workspaces.
2. **Digest & Delete Policy (`01_raw/`)**:
   - `docs/01_raw/` is a temporary intake pool. Once raw materials are 100% distilled and verified into `docs/02_wiki/`, the raw files must be deleted immediately.
3. **Mandatory Step 0 (Pre-Flight Feedbacks Inspection)**:
   - Before writing or distilling, agents must inspect `.agents/skills/wiki-distiller/feedbacks/` for all `🟢 [ACCEPTED]` standards and `🔴 [REJECTED]` boundaries.
4. **Zero-Persona Contamination in Wiki**:
   - Reviewer names (e.g., Xiao-Ming, Old Chen) belong strictly to `.agents/skills/wiki-distiller/roles/` and `docs/reviews/`. Wiki body text must remain 100% objective, authoritative, and persona-free.
5. **Mandatory 4-Dimension Diagram Walkthrough**:
   - Every Mermaid diagram must be followed by a walkthrough detailing Core View, Step-by-Step path, Color/Physical semantics, and Underlying engineering details.
