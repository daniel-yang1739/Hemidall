package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ToolSignature represents a parsed, clean non-JSON tool declaration
type ToolSignature struct {
	Name        string   `json:"name"`
	Signature   string   `json:"signature"`
	Description string   `json:"description"`
	Required    []string `json:"required"`
	RawSchema   string   `json:"raw_schema"`
}

// SkillInfo represents an installed agent skill
type SkillInfo struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // ACTIVE, READY
	Path        string `json:"path"`
	Description string `json:"description"`
	Guidelines  string `json:"guidelines"`
	RawMarkdown string `json:"raw_markdown"`
}

// AgentContextPayload is the comprehensive universal domain model for [2] Context view
type AgentContextPayload struct {
	AgentType    AgentType
	TargetModel  string
	TotalTokens  int
	ContextLimit int
	IsRawMode    bool

	// 1. SYSTEM & RULES
	IdentityPrompt  string
	ConstitutionDoc string
	RuntimeMetadata map[string]string

	// 2. TOOLS & SCHEMAS
	NativeTools []ToolSignature
	ActiveSkills []SkillInfo
	MCPServers  []string

	// 3. CONTEXT HIST
	CheckpointSummary   string
	CompactedStepsCount int
	ActiveHistoryTurns  []UnifiedAgentEvent

	// 4. ACTIVE INBOUND
	LatestPrompt     string
	StagedBuffers    string
	ProjectedHitRate float64
	ProjectedCached  int
	ProjectedNew     int
	ProjectedCostUSD float64
	ProjectedCostTWD float64
}

// GetNativeToolsDefinitions returns the 8 native harness tool definitions and schemas
func GetNativeToolsDefinitions() []ToolSignature {
	return []ToolSignature{
		{
			Name:        "write_to_file",
			Signature:   "write_to_file(TargetFile, CodeContent, Overwrite)",
			Description: "Create or overwrite file artifacts. Requires Summary & UserFacing meta.",
			Required:    []string{"TargetFile", "CodeContent", "Overwrite"},
			RawSchema: `{\n  "name": "write_to_file",\n  "description": "Use this tool to create new files...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["TargetFile", "CodeContent", "Overwrite"],\n    "properties": {\n      "TargetFile": { "type": "STRING", "description": "Absolute path" },\n      "CodeContent": { "type": "STRING", "description": "File body" },\n      "Overwrite": { "type": "BOOLEAN", "description": "Force overwrite" }\n    }\n  }\n}`,
		},
		{
			Name:        "replace_file_content",
			Signature:   "replace_file_content(TargetFile, Instruction, TargetContent, ReplacementContent, ...)",
			Description: "Precise contiguous code replacement. Strict whitespace matching.",
			Required:    []string{"TargetFile", "Instruction", "TargetContent", "ReplacementContent"},
			RawSchema: `{\n  "name": "replace_file_content",\n  "description": "Use this tool to edit an existing file...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["TargetFile", "Instruction", "TargetContent", "ReplacementContent"],\n    "properties": {\n      "TargetFile": { "type": "STRING" },\n      "Instruction": { "type": "STRING" },\n      "TargetContent": { "type": "STRING" },\n      "ReplacementContent": { "type": "STRING" }\n    }\n  }\n}`,
		},
		{
			Name:        "run_command",
			Signature:   "run_command(CommandLine: string, Cwd: string, WaitMsBeforeAsync: int)",
			Description: "Execute zsh/bash shell command. Cwd must stay within workspace.",
			Required:    []string{"CommandLine", "Cwd", "WaitMsBeforeAsync"},
			RawSchema: `{\n  "name": "run_command",\n  "description": "PROPOSE a command to run on behalf of user...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["CommandLine", "Cwd", "WaitMsBeforeAsync"],\n    "properties": {\n      "CommandLine": { "type": "STRING" },\n      "Cwd": { "type": "STRING" },\n      "WaitMsBeforeAsync": { "type": "INTEGER" }\n    }\n  }\n}`,
		},
		{
			Name:        "view_file",
			Signature:   "view_file(AbsolutePath: string, StartLine: int, EndLine: int)",
			Description: "Read text file lines (1-indexed) or binary images/media.",
			Required:    []string{"AbsolutePath"},
			RawSchema: `{\n  "name": "view_file",\n  "description": "View contents of file from local filesystem...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["AbsolutePath"],\n    "properties": {\n      "AbsolutePath": { "type": "STRING" },\n      "StartLine": { "type": "INTEGER" },\n      "EndLine": { "type": "INTEGER" }\n    }\n  }\n}`,
		},
		{
			Name:        "grep_search",
			Signature:   "grep_search(Query: string, SearchPath: string, CaseInsensitive: bool)",
			Description: "Fast ripgrep pattern & regex search across files and directories.",
			Required:    []string{"Query", "SearchPath"},
			RawSchema: `{\n  "name": "grep_search",\n  "description": "Use ripgrep to find exact pattern matches...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["Query", "SearchPath"],\n    "properties": {\n      "Query": { "type": "STRING" },\n      "SearchPath": { "type": "STRING" },\n      "CaseInsensitive": { "type": "BOOLEAN" }\n    }\n  }\n}`,
		},
		{
			Name:        "find_by_name",
			Signature:   "find_by_name(Pattern: string, SearchDirectory: string, Type: string)",
			Description: "Fast fd glob matching for file and directory discovery.",
			Required:    []string{"Pattern", "SearchDirectory"},
			RawSchema: `{\n  "name": "find_by_name",\n  "description": "Search for files within directory using fd...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["Pattern", "SearchDirectory"],\n    "properties": {\n      "Pattern": { "type": "STRING" },\n      "SearchDirectory": { "type": "STRING" },\n      "Type": { "type": "STRING" }\n    }\n  }\n}`,
		},
		{
			Name:        "list_dir",
			Signature:   "list_dir(DirectoryPath: string)",
			Description: "List immediate children and subdirectories.",
			Required:    []string{"DirectoryPath"},
			RawSchema: `{\n  "name": "list_dir",\n  "description": "List contents of directory...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["DirectoryPath"],\n    "properties": {\n      "DirectoryPath": { "type": "STRING" }\n    }\n  }\n}`,
		},
		{
			Name:        "ask_question",
			Signature:   "ask_question(questions: []QuestionObject)",
			Description: "Interactive multi-choice clarifying modal with radio/checkboxes.",
			Required:    []string{"questions"},
			RawSchema: `{\n  "name": "ask_question",\n  "description": "Ask user one or more multiple-choice questions...",\n  "parameters": {\n    "type": "OBJECT",\n    "required": ["questions"],\n    "properties": {\n      "questions": { "type": "ARRAY" }\n    }\n  }\n}`,
		},
	}
}

// GetActiveSkillsDefinitions returns the repertory of installed agent skills
func GetActiveSkillsDefinitions() []SkillInfo {
	skills := []SkillInfo{
		{
			Name:        "wiki-distiller",
			Status:      "ACTIVE",
			Path:        "/Users/daniel_y_yang/Documents/self/ithome2026/.agents/skills/wiki-distiller/SKILL.md",
			Description: "Andrej Karpathy LLM Wiki 知識提煉專家。雙輪深層對抗審查與抗體沉澱。",
			Guidelines:  "Double-round adversarial review loop, secretary antibody memory in feedbacks/, zero-persona in wiki body text.",
			RawMarkdown: "---\nname: wiki-distiller\ndescription: Andrej Karpathy LLM Wiki 知識提煉專家。\n---",
		},
		{
			Name:        "agy-customizations",
			Status:      "READY",
			Path:        "/Users/daniel_y_yang/.gemini/antigravity-cli/builtin/skills/agy-customizations/SKILL.md",
			Description: "Comprehensive guide and reference for the Antigravity Customization System.",
			Guidelines:  "Loading priorities, discovery mechanisms, rules, plugins, hooks, and MCP servers.",
			RawMarkdown: "---\nname: agy-customizations\ndescription: Antigravity Customization System.\n---",
		},
		{
			Name:        "antigravity-guide",
			Status:      "READY",
			Path:        "/Users/daniel_y_yang/.gemini/antigravity-cli/builtin/skills/antigravity_guide/SKILL.md",
			Description: "Comprehensive guide, sitemap & slash commands for Antigravity.",
			Guidelines:  "AGY CLI, Antigravity IDE, slash commands, keybindings, and SDK reference.",
			RawMarkdown: "---\nname: antigravity-guide\ndescription: Antigravity Comprehensive Guide.\n---",
		},
	}

	for i := range skills {
		if content, err := os.ReadFile(skills[i].Path); err == nil && len(content) > 0 {
			skills[i].RawMarkdown = string(content)
		}
	}

	return skills
}

// ExtractAgentContextPayload extracts the structured context domain model from session history
func ExtractAgentContextPayload(history []UnifiedAgentEvent, sessionID, targetModel string) AgentContextPayload {
	if targetModel == "" {
		targetModel = "Gemini 3.7 Flash"
	}

	payload := AgentContextPayload{
		AgentType:    AgentTypeAntigravity,
		TargetModel:  targetModel,
		ContextLimit: 256000,
		TotalTokens:  185200,
		RuntimeMetadata: map[string]string{
			"OS":         "macOS Darwin 24.5.0",
			"Shell":      "zsh (/bin/zsh)",
			"Cwd":        "/Users/daniel_y_yang/Documents/self/ithome2026",
			"SessionID":  sessionID,
			"Model":      targetModel,
			"Harness":    "Google Antigravity Harness v2.0",
		},
		NativeTools:  GetNativeToolsDefinitions(),
		ActiveSkills: GetActiveSkillsDefinitions(),
		MCPServers:   []string{},
		IdentityPrompt: `You are Antigravity, a powerful agentic AI coding assistant designed by the Google DeepMind team.
Pair-programming with USER to solve coding tasks, investigate architectures, write tests, and build software.`,
		ConstitutionDoc: `# 📜 REPOSITORY CONSTITUTION (專案開發與協同憲法)
ARTICLE I: LANGUAGE POLICY (Code EN, Docs TC)
ARTICLE II: OBSIDIAN VAULT WHITELIST
ARTICLE III: WIKI DISTILLATION & KM CONSTITUTION`,
		CheckpointSummary: `[TRUNCATION_CHECKPOINT_BASE]
Conversation memory compacted at step #3000. 245,000 Historical Tokens -> 12,500 Summary (94.9% Compaction).
Memory re-anchored for sliding window prefill.`,
		CompactedStepsCount: 5800,
		LatestPrompt:        "/plan 把所有測試加進去我們再來補新功能。記得重點是期待不能變，不能因為我們要讓測試過，改測試讓他符合錯誤的邏輯。",
		StagedBuffers:       "Step #6220 edit_file -> 110 Tokens Buffer (440 bytes diff)\nDispatched by Cloud Step #6219 -> Staged for Next Cloud Inference Turn",
		ProjectedHitRate:    88.4,
		ProjectedCached:     163720,
		ProjectedNew:        21480,
		ProjectedCostUSD:    0.0089,
		ProjectedCostTWD:    0.28,
	}

	// Attempt to load genuine AGENTS.md directly from filesystem
	for _, agentsPath := range []string{"../AGENTS.md", "AGENTS.md", "/Users/daniel_y_yang/Documents/self/ithome2026/AGENTS.md"} {
		if content, err := os.ReadFile(agentsPath); err == nil && len(content) > 0 {
			payload.ConstitutionDoc = strings.TrimSpace(string(content))
			break
		}
	}

	// Scan history only on step 0 or StepTypeSystemInit to prevent code-view contamination
	for _, e := range history {
		if (e.Type == StepTypeSystemInit || e.StepIndex == 0) && strings.Contains(e.RawContent, "<identity>") {
			if start := strings.Index(e.RawContent, "<identity>"); start >= 0 {
				end := strings.Index(e.RawContent, "</identity>")
				if end > start {
					payload.IdentityPrompt = strings.TrimSpace(e.RawContent[start+10 : end])
				}
			}
		}
		if (e.Type == StepTypeSystemInit || e.StepIndex == 0) && strings.Contains(e.RawContent, "<user_rules>") {
			if start := strings.Index(e.RawContent, "<user_rules>"); start >= 0 {
				end := strings.Index(e.RawContent, "</user_rules>")
				if end > start {
					payload.ConstitutionDoc = strings.TrimSpace(e.RawContent[start+12 : end])
				}
			}
		}
		if e.Type == StepTypeCheckpoint || strings.Contains(e.RawContent, "<CONTEXT_SUMMARY>") {
			payload.CheckpointSummary = e.RawContent
			payload.CompactedStepsCount = e.StepIndex
		}
		if e.Type == StepTypeUserInput && e.RawContent != "" {
			payload.LatestPrompt = e.RawContent
		}
	}

	if len(history) > 0 {
		latest := history[len(history)-1]
		if latest.Tokens.TotalTokens > 0 {
			payload.TotalTokens = latest.Tokens.TotalTokens
		}
		// Extract last 5 turns
		if len(history) > 5 {
			payload.ActiveHistoryTurns = history[len(history)-5:]
		} else {
			payload.ActiveHistoryTurns = history
		}
	}

	return payload
}

func marshalJSONNoEscape(v interface{}) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// SerializeToWirePayload serializes the AgentContextPayload into Gemini GenerateContentRequest JSON format
func SerializeToWirePayload(payload AgentContextPayload) (string, error) {
	// Build Gemini API Request Structure
	var funcDecls []map[string]interface{}
	for _, tool := range payload.NativeTools {
		funcDecls = append(funcDecls, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters": map[string]interface{}{
				"type":     "OBJECT",
				"required": tool.Required,
			},
		})
	}

	var contents []map[string]interface{}
	// Checkpoint or Turn 0
	if payload.CheckpointSummary != "" {
		contents = append(contents, map[string]interface{}{
			"role": "user",
			"parts": []map[string]string{
				{"text": payload.CheckpointSummary},
			},
		})
	}

	// Latest Inbound Turn
	contents = append(contents, map[string]interface{}{
		"role": "user",
		"parts": []map[string]string{
			{"text": fmt.Sprintf("<USER_REQUEST>%s</USER_REQUEST>", payload.LatestPrompt)},
		},
	})

	wireObj := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": fmt.Sprintf("<identity>%s</identity>\n<user_rules>%s</user_rules>", payload.IdentityPrompt, payload.ConstitutionDoc)},
			},
		},
		"tools": []map[string]interface{}{
			{
				"functionDeclarations": funcDecls,
			},
		},
		"contents": contents,
	}

	if len(payload.NativeTools) == 0 {
		wireObj["tools"] = []interface{}{}
	}

	return marshalJSONNoEscape(wireObj)
}

// Subcategory identifiers in Context Tree
const (
	SubcatAll       = 0 // 🌐 FULL OUTBOUND PAYLOAD (ALL)
	SubcatIdentity  = 1
	SubcatAgentsMD  = 2
	SubcatRuntime   = 3
	SubcatTools     = 4
	SubcatSkills    = 5
	SubcatMCP       = 6
	SubcatAnchor    = 7
	SubcatCompacted = 8
	SubcatTurns     = 9
	SubcatPrompt    = 10
	SubcatBuffers   = 11
)

// SerializeSubcategoryRaw serializes ONLY the selected subcategory part into its raw JSON wire representation
func SerializeSubcategoryRaw(payload AgentContextPayload, subcatIndex int) (string, error) {
	var rawObj interface{}

	switch subcatIndex {
	case SubcatAll:
		return SerializeToWirePayload(payload)

	case SubcatIdentity:
		rawObj = map[string]interface{}{
			"role": "system",
			"name": "identity",
			"parts": []map[string]string{
				{"text": fmt.Sprintf("<identity>\n%s\n</identity>", payload.IdentityPrompt)},
			},
		}

	case SubcatAgentsMD:
		rawObj = map[string]interface{}{
			"role": "system",
			"name": "user_rules",
			"parts": []map[string]string{
				{"text": fmt.Sprintf("<user_rules>\n%s\n</user_rules>", payload.ConstitutionDoc)},
			},
		}

	case SubcatRuntime:
		metaLines := make([]string, 0, len(payload.RuntimeMetadata))
		for k, v := range payload.RuntimeMetadata {
			metaLines = append(metaLines, fmt.Sprintf("%s: %s", k, v))
		}
		rawObj = map[string]interface{}{
			"role": "system",
			"name": "user_information",
			"parts": []map[string]string{
				{"text": fmt.Sprintf("<user_information>\n%s\n</user_information>", strings.Join(metaLines, "\n"))},
			},
		}

	case SubcatTools:
		var funcDecls []map[string]interface{}
		for _, tool := range payload.NativeTools {
			funcDecls = append(funcDecls, map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters": map[string]interface{}{
					"type":     "OBJECT",
					"required": tool.Required,
				},
			})
		}
		rawObj = map[string]interface{}{
			"functionDeclarations": funcDecls,
		}

	case SubcatSkills:
		var skillLines []string
		for _, s := range payload.ActiveSkills {
			skillLines = append(skillLines, fmt.Sprintf("- %s (%s): %s", s.Name, s.Path, s.Description))
		}
		rawObj = map[string]interface{}{
			"role": "system",
			"name": "skills",
			"parts": []map[string]string{
				{"text": fmt.Sprintf("<skills>\nAvailable skills:\n%s\n</skills>", strings.Join(skillLines, "\n"))},
			},
		}

	case SubcatMCP:
		rawObj = map[string]interface{}{
			"mcpServers": payload.MCPServers,
			"count":      len(payload.MCPServers),
			"status":     "No external MCP servers configured",
		}

	case SubcatAnchor:
		rawObj = map[string]interface{}{
			"role": "user",
			"parts": []map[string]string{
				{"text": payload.CheckpointSummary},
			},
		}

	case SubcatCompacted:
		rawObj = map[string]interface{}{
			"checkpointType":           "TRUNCATION_CHECKPOINT",
			"compactedStepsCount":      payload.CompactedStepsCount,
			"originalHistoricalTokens": 245000,
			"compactedSummaryTokens":   12500,
			"compactionRatio":          0.949,
			"anchorRole":               "user",
		}

	case SubcatTurns:
		var turns []map[string]interface{}
		for _, e := range payload.ActiveHistoryTurns {
			role := "user"
			if e.IsCloudStep() {
				role = "model"
			}
			turns = append(turns, map[string]interface{}{
				"stepIndex": e.StepIndex,
				"role":      role,
				"parts": []map[string]string{
					{"text": e.RawContent},
				},
			})
		}
		if len(turns) == 0 {
			turns = append(turns, map[string]interface{}{
				"stepIndex": 6450,
				"role":      "user",
				"parts": []map[string]string{
					{"text": "<USER_REQUEST>/plan ...</USER_REQUEST>"},
				},
			})
		}
		rawObj = turns

	case SubcatPrompt:
		rawObj = map[string]interface{}{
			"role": "user",
			"parts": []map[string]string{
				{"text": fmt.Sprintf("<USER_REQUEST>\n%s\n</USER_REQUEST>", payload.LatestPrompt)},
			},
		}

	case SubcatBuffers:
		rawObj = map[string]interface{}{
			"role": "user",
			"parts": []map[string]interface{}{
				{
					"functionResponse": map[string]interface{}{
						"name": "edit_file",
						"response": map[string]interface{}{
							"status":       "staged",
							"stagedTokens": 110,
							"bufferBytes":  440,
							"raw":          payload.StagedBuffers,
						},
					},
				},
			},
		}

	default:
		return SerializeToWirePayload(payload)
	}

	return marshalJSONNoEscape(rawObj)
}
