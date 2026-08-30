package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// ContextEvidenceKind identifies what Heimdall actually observed. It deliberately does
// not claim that a reconstructed value is a network request.
type ContextEvidenceKind string

const (
	ContextEvidencePersistedSnapshot  ContextEvidenceKind = "PERSISTED_CONTEXT_SNAPSHOT"
	ContextEvidenceTranscriptFallback ContextEvidenceKind = "TRANSCRIPT_DERIVED_FALLBACK"
)

// AgentContextPayload is the evidence-oriented domain model for the context view.
type AgentContextPayload struct {
	AgentType                  AgentType
	TargetModel                string
	TotalTokens                int
	ContextLimit               int
	IsRawMode                  bool
	Provenance                 map[string]string
	SourceKind                 ContextEvidenceKind
	SourcePath                 string
	SnapshotGenIndex           int
	SnapshotBytes              int
	SnapshotAvailable          bool
	SystemPrompt               string
	SkillsSection              string
	MCPSection                 string
	PersistedContextEntryCount int
	PersistedRecords           []PersistedContextRecord

	// 1. SYSTEM & RULES
	IdentityPrompt  string
	ConstitutionDoc string
	RuntimeMetadata map[string]string

	// 2. TOOLS & SCHEMAS
	NativeTools  []ToolSignature
	ActiveSkills []SkillInfo
	MCPServers   []string

	// 3. CONTEXT HIST
	CheckpointSummary   string
	CheckpointStepIndex int
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

// PersistedContextRecord is a safely decoded, schema-agnostic snapshot entry.
// ObservedKind and ObservedSequence are wire observations, not official field names.
type PersistedContextRecord struct {
	Position              int
	ByteSize              int
	ObservedKind          uint64
	ObservedSequence      uint64
	PrimaryText           string
	HasPrivateContent     bool
	IsCompactedCheckpoint bool
}

// ContextBuildInput contains adapter-observed facts used to build a universal context payload.
type ContextBuildInput struct {
	History         []UnifiedAgentEvent
	SessionID       string
	AgentType       AgentType
	TargetModel     string
	NativeTools     []ToolSignature
	ActiveSkills    []SkillInfo
	MCPServers      []string
	RuntimeMetadata map[string]string
	Provenance      map[string]string
}

// MeasureDynamicBaselineTokens estimates the locally discoverable baseline.
// Unavailable harness instructions and schemas are represented by fallback estimates.
func MeasureDynamicBaselineTokens() (systemTokens int, toolsDefTokens int) {
	cwd, _ := os.Getwd()
	constitution := readNearestFile(cwd, "AGENTS.md")
	runtimeDescription := strings.Join([]string{runtime.GOOS, runtime.GOARCH, os.Getenv("SHELL")}, " ")

	d1 := CountTokens(constitution) + CountTokens(runtimeDescription)
	if d1 <= 0 {
		d1 = DefaultFallbackSystemTokens
	}

	// Calculate Tools JSON Schema tokens
	skills := DiscoverSkillDefinitions(FindNearestDirectory(cwd, filepath.Join(".agents", "skills")))
	toolsText := ""
	for _, s := range skills {
		toolsText += s.RawMarkdown + "\n"
	}
	d2 := CountTokens(toolsText)
	if d2 <= 0 {
		d2 = DefaultFallbackToolsDefTokens
	}

	return d1, d2
}

// GetNativeToolsDefinitions derives tool names and argument keys observed in session history.
// Antigravity does not persist the authoritative request schema in the JSONL transcript.
func GetNativeToolsDefinitions(history []UnifiedAgentEvent) []ToolSignature {
	observed := make(map[string]map[string]struct{})
	for _, event := range history {
		for _, call := range event.ToolCalls {
			if call.ToolName == "" {
				continue
			}
			if observed[call.ToolName] == nil {
				observed[call.ToolName] = make(map[string]struct{})
			}
			for key := range call.Arguments {
				observed[call.ToolName][key] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(observed))
	for name := range observed {
		names = append(names, name)
	}
	sort.Strings(names)

	tools := make([]ToolSignature, 0, len(names))
	for _, name := range names {
		keys := make([]string, 0, len(observed[name]))
		for key := range observed[name] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		rawSchema, _ := marshalJSONNoEscape(map[string]interface{}{
			"name":                 name,
			"observedArgumentKeys": keys,
		})
		tools = append(tools, ToolSignature{
			Name:        name,
			Signature:   fmt.Sprintf("%s(%s)", name, strings.Join(keys, ", ")),
			Description: "Observed in the session transcript; authoritative schema unavailable.",
			RawSchema:   rawSchema,
		})
	}
	return tools
}

// BuildContextPayloadFromHistory assembles adapter-observed facts into the universal context model.
func BuildContextPayloadFromHistory(input ContextBuildInput) AgentContextPayload {
	history := input.History
	spec := ResolveModelSpec(input.TargetModel, nil)

	payload := AgentContextPayload{
		AgentType:       input.AgentType,
		TargetModel:     input.TargetModel,
		ContextLimit:    spec.DefaultAgentWindow,
		Provenance:      input.Provenance,
		RuntimeMetadata: input.RuntimeMetadata,
		NativeTools:     input.NativeTools,
		ActiveSkills:    input.ActiveSkills,
		MCPServers:      input.MCPServers,
	}

	checkpointHistoryIndex := -1
	// Scan only system bootstrap events for system fields to prevent code-view contamination.
	for i, e := range history {
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
			payload.CheckpointStepIndex = e.StepIndex
			checkpointHistoryIndex = i
		}
		if e.Type == StepTypeUserInput && e.RawContent != "" {
			payload.LatestPrompt = e.RawContent
		}
	}

	if len(history) > 0 {
		payload.ActiveHistoryTurns = history[checkpointHistoryIndex+1:]
	}
	payload.StagedBuffers = extractStagedBuffers(history)
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Tokens.TotalTokens <= 0 {
			continue
		}
		payload.TotalTokens = history[i].Tokens.TotalTokens
		if history[i].Tokens.ContextLimit > 0 {
			payload.ContextLimit = history[i].Tokens.ContextLimit
		}
		break
	}

	return payload
}

func readNearestFile(startDir, name string) string {
	for dir := startDir; dir != ""; dir = filepath.Dir(dir) {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return strings.TrimSpace(string(content))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}

func extractStagedBuffers(history []UnifiedAgentEvent) string {
	lastCloudIndex := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].IsCloudStep() {
			lastCloudIndex = i
			break
		}
	}
	var staged []string
	for _, event := range history[lastCloudIndex+1:] {
		if event.IsLocalStep() && event.RawContent != "" {
			staged = append(staged, fmt.Sprintf("Step #%d %s: %s", event.StepIndex, event.Type, event.RawContent))
		}
	}
	return strings.Join(staged, "\n")
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

// SerializeContextEvidence renders only the data Heimdall decoded or observed, with
// provenance. It is not an official HTTP request or a GenerateContentRequest body.
func SerializeContextEvidence(payload AgentContextPayload) (string, error) {
	return marshalJSONNoEscape(map[string]interface{}{
		"evidence": map[string]interface{}{
			"kind":               payload.SourceKind,
			"source_path":        payload.SourcePath,
			"gen_metadata_index": payload.SnapshotGenIndex,
			"snapshot_bytes":     payload.SnapshotBytes,
			"warning":            "This is Heimdall's evidence view, not a captured HTTP request or an official GenerateContentRequest body.",
		},
		"provenance":                         payload.Provenance,
		"persisted_system_prompt_text":       payload.SystemPrompt,
		"persisted_tool_entries":             payload.NativeTools,
		"persisted_field_2_occurrence_count": payload.PersistedContextEntryCount,
		"heimdall_runtime_metadata":          payload.RuntimeMetadata,
		"latest_available_context_tokens":    payload.TotalTokens,
		"latest_available_context_limit":     payload.ContextLimit,
		"transcript_observations": map[string]interface{}{
			"checkpoint_summary":    payload.CheckpointSummary,
			"checkpoint_step_index": payload.CheckpointStepIndex,
			"active_events":         payload.ActiveHistoryTurns,
			"latest_prompt":         payload.LatestPrompt,
			"staged_local_content":  payload.StagedBuffers,
		},
	})
}

// Subcategory identifiers in Context Tree
const (
	SubcatAll = iota // 🌐 FULL OUTBOUND PAYLOAD (ALL)
	SubcatIdentity
	SubcatAgentsMD
	SubcatRuntime
	SubcatTools
	SubcatSkills
	SubcatMCP
	SubcatAnchor
	SubcatCurrentHistory
	SubcatTurns
	SubcatPrompt
	SubcatBuffers
)

const SubcatLast = SubcatBuffers

// SerializeSubcategoryRaw serializes ONLY the selected subcategory part into its raw JSON wire representation
func SerializeSubcategoryRaw(payload AgentContextPayload, subcatIndex int) (string, error) {
	var rawObj interface{}

	switch subcatIndex {
	case SubcatAll:
		return SerializeContextEvidence(payload)

	case SubcatIdentity:
		rawObj = map[string]interface{}{
			"source":     payload.Provenance["identity"],
			"parsed_tag": "identity",
			"text":       payload.IdentityPrompt,
		}

	case SubcatAgentsMD:
		rawObj = map[string]interface{}{
			"source":     payload.Provenance["user_rules"],
			"parsed_tag": "user_rules",
			"text":       payload.ConstitutionDoc,
		}

	case SubcatRuntime:
		metaLines := make([]string, 0, len(payload.RuntimeMetadata))
		for k, v := range payload.RuntimeMetadata {
			metaLines = append(metaLines, fmt.Sprintf("%s: %s", k, v))
		}
		rawObj = map[string]interface{}{
			"source":                    payload.Provenance["runtime"],
			"heimdall_process_metadata": metaLines,
		}

	case SubcatTools:
		rawObj = map[string]interface{}{
			"source":               payload.Provenance["tools"],
			"decoded_tool_entries": payload.NativeTools,
		}

	case SubcatSkills:
		var skillLines []string
		for _, s := range payload.ActiveSkills {
			skillLines = append(skillLines, fmt.Sprintf("- %s (%s): %s", s.Name, s.Path, s.Description))
		}
		rawObj = map[string]interface{}{
			"source":                 payload.Provenance["skills"],
			"parsed_tag":             "skills",
			"text":                   payload.SkillsSection,
			"filesystem_discoveries": skillLines,
		}

	case SubcatMCP:
		rawObj = map[string]interface{}{
			"status":                  "UNKNOWN",
			"reason":                  "The reverse-engineered persisted schema does not attribute tools to MCP servers.",
			"mcp_related_system_text": payload.MCPSection,
		}

	case SubcatAnchor:
		rawObj = map[string]interface{}{
			"source":                payload.Provenance["checkpoint"],
			"checkpoint_step_index": payload.CheckpointStepIndex,
			"checkpoint_summary":    payload.CheckpointSummary,
		}

	case SubcatTurns:
		rawObj = map[string]interface{}{
			"source": payload.Provenance["active_turns"],
			"events": payload.ActiveHistoryTurns,
		}

	case SubcatCurrentHistory:
		rawObj = map[string]interface{}{
			"source":        "persisted_gen_metadata_field_1.2_repeated",
			"active_events": payload.PersistedRecords,
		}

	case SubcatPrompt:
		rawObj = map[string]interface{}{
			"source":        payload.Provenance["latest_prompt"],
			"latest_prompt": payload.LatestPrompt,
		}

	case SubcatBuffers:
		rawObj = map[string]interface{}{
			"source":                payload.Provenance["staged_buffers"],
			"observedStagedContent": payload.StagedBuffers,
		}

	default:
		return SerializeContextEvidence(payload)
	}

	return marshalJSONNoEscape(rawObj)
}

// SerializePersistedActiveEventRaw serializes either all persisted active events
// or one selected event without falling back to the full context payload.
func SerializePersistedActiveEventRaw(payload AgentContextPayload, position int) (string, error) {
	if position == 0 {
		return marshalJSONNoEscape(map[string]interface{}{
			"source":        "persisted_gen_metadata_field_1.2_repeated",
			"active_events": payload.PersistedRecords,
		})
	}
	for _, record := range payload.PersistedRecords {
		if record.Position == position {
			return marshalJSONNoEscape(map[string]interface{}{
				"source":       "persisted_gen_metadata_field_1.2_repeated",
				"active_event": record,
			})
		}
	}
	return marshalJSONNoEscape(map[string]interface{}{
		"source":       "persisted_gen_metadata_field_1.2_repeated",
		"active_event": nil,
	})
}
