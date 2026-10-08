package core

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// SystemPromptSection is one top-level tagged section in persisted prompt order.
type SystemPromptSection struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
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
	SnapshotInputBoundaryStep  int
	SnapshotHasInputBoundary   bool
	SystemPrompt               string
	SystemPromptSections       []SystemPromptSection
	SkillsSection              string
	MCPSection                 string
	PersistedContextEntryCount int
	PersistedRecords           []PersistedContextRecord

	// 1. SYSTEM & RULES
	IdentityPrompt  string
	ConstitutionDoc string

	// 2. TOOLS & SCHEMAS
	NativeTools  []ToolSignature
	ActiveSkills []SkillInfo
	MCPServers   []string

	// 3. CONTEXT HIST
	CheckpointSummary   string
	CheckpointStepIndex int
	ActiveHistoryTurns  []UnifiedAgentEvent

	// 4. ACTIVE INBOUND
	LatestPrompt  string
	StagedBuffers string
}

// PersistedContextRecord is a decoded snapshot entry with authoritative proto semantics.
type PersistedContextRecord struct {
	Position              int
	ByteSize              int
	ObservedKind          uint64
	RoleName              string
	RoleDescription       string
	ObservedSequence      uint64
	PrimaryText           string
	HasPrivateContent     bool
	IsCompactedCheckpoint bool
}

// ContextBuildInput contains adapter-observed facts used to build a universal context payload.
type ContextBuildInput struct {
	History      []UnifiedAgentEvent
	SessionID    string
	TargetModel  string
	NativeTools  []ToolSignature
	ActiveSkills []SkillInfo
	MCPServers   []string
	Provenance   map[string]string
}

// GetNativeToolsDefinitions derives tool names and argument keys observed in session history.
// Antigravity does not persist the authoritative request schema in the JSONL transcript.
func GetNativeToolsDefinitions(history []UnifiedAgentEvent) []ToolSignature {
	observed := make(map[string]map[string]struct{})
	for _, event := range history {
		observeToolDefinitions(observed, event.ToolCalls)
	}
	return toolSignaturesFromObservedArguments(observed)
}

func observeToolDefinitions(observed map[string]map[string]struct{}, calls []ToolCallInfo) bool {
	changed := false
	for _, call := range calls {
		if call.ToolName == "" {
			continue
		}
		if observed[call.ToolName] == nil {
			observed[call.ToolName] = make(map[string]struct{})
			changed = true
		}
		for key := range call.Arguments {
			if _, exists := observed[call.ToolName][key]; !exists {
				observed[call.ToolName][key] = struct{}{}
				changed = true
			}
		}
	}
	return changed
}

func toolSignaturesFromObservedArguments(observed map[string]map[string]struct{}) []ToolSignature {
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
	payload := AgentContextPayload{
		TargetModel:  input.TargetModel,
		Provenance:   input.Provenance,
		NativeTools:  input.NativeTools,
		ActiveSkills: input.ActiveSkills,
		MCPServers:   input.MCPServers,
	}

	checkpointHistoryIndex := -1
	// Scan only system bootstrap events for system fields to prevent code-view contamination.
	for i, e := range history {
		if (e.Type == StepTypeSystemInit || e.StepIndex == 0) && strings.Contains(e.RawContent, "<identity>") {
			payload.IdentityPrompt = extractTranscriptTaggedSection(e.RawContent, "identity")
		}
		if (e.Type == StepTypeSystemInit || e.StepIndex == 0) && strings.Contains(e.RawContent, "<user_rules>") {
			payload.ConstitutionDoc = extractTranscriptTaggedSection(e.RawContent, "user_rules")
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
		if !history[i].Usage.HasObservedContextTokens {
			continue
		}
		payload.TotalTokens = history[i].Usage.ObservedContextTokens
		if history[i].Usage.HasContextLimit {
			payload.ContextLimit = history[i].Usage.ContextLimit
		}
		break
	}

	return payload
}

// BuildContextPayloadFromSession overlays the latest parsed persisted snapshot
// onto transcript-derived facts. It consumes only the core session model, never
// an adapter path or parser implementation.
func BuildContextPayloadFromSession(session Session, input ContextBuildInput) AgentContextPayload {
	payload := BuildContextPayloadFromHistory(input)
	payload.SourceKind = ContextEvidenceTranscriptFallback
	if len(session.ContextSnapshots) == 0 {
		return payload
	}
	snapshot := session.ContextSnapshots[len(session.ContextSnapshots)-1]
	payload.SourceKind = ContextEvidencePersistedSnapshot
	payload.SourcePath = snapshot.Source.Path
	payload.SnapshotAvailable = true
	payload.SnapshotGenIndex = snapshot.GenerationIndex
	payload.SnapshotBytes = snapshot.ByteSize
	payload.SnapshotInputBoundaryStep = snapshot.InputBoundaryStepIndex
	payload.SnapshotHasInputBoundary = snapshot.HasInputBoundary
	payload.SystemPrompt = snapshot.SystemPrompt
	payload.SystemPromptSections = ParseTopLevelSystemPromptSections(snapshot.SystemPrompt)
	payload.IdentityPrompt = snapshot.IdentityPrompt
	payload.ConstitutionDoc = snapshot.ConstitutionDoc
	payload.SkillsSection = snapshot.SkillsSection
	payload.MCPSection = snapshot.MCPSection
	payload.PersistedContextEntryCount = len(snapshot.PersistedContextRecords)
	payload.PersistedRecords = append([]PersistedContextRecord(nil), snapshot.PersistedContextRecords...)
	payload.NativeTools = append([]ToolSignature(nil), snapshot.NativeTools...)
	payload.ActiveSkills = nil
	payload.MCPServers = nil
	return payload
}

// ParseTopLevelSystemPromptSections extracts balanced outer tags without
// flattening nested markup inside each section.
func ParseTopLevelSystemPromptSections(text string) []SystemPromptSection {
	type openTag struct {
		name         string
		contentStart int
	}
	sections := make([]SystemPromptSection, 0)
	stack := make([]openTag, 0)
	for cursor := 0; cursor < len(text); {
		relativeStart := strings.IndexByte(text[cursor:], '<')
		if relativeStart < 0 {
			break
		}
		start := cursor + relativeStart
		relativeEnd := strings.IndexByte(text[start:], '>')
		if relativeEnd < 0 {
			break
		}
		end := start + relativeEnd
		name, closing, selfClosing := parsePromptTagToken(text[start+1 : end])
		cursor = end + 1
		if name == "" {
			continue
		}
		if closing {
			if len(stack) == 0 || stack[len(stack)-1].name != name {
				continue
			}
			opened := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				sections = append(sections, SystemPromptSection{Tag: name, Content: strings.TrimSpace(text[opened.contentStart:start])})
			}
			continue
		}
		if selfClosing {
			if len(stack) == 0 {
				sections = append(sections, SystemPromptSection{Tag: name})
			}
			continue
		}
		if !strings.Contains(text[cursor:], "</"+name+">") {
			continue
		}
		stack = append(stack, openTag{name: name, contentStart: end + 1})
	}
	return sections
}

func parsePromptTagToken(token string) (string, bool, bool) {
	token = strings.TrimSpace(token)
	if token == "" || strings.HasPrefix(token, "!") || strings.HasPrefix(token, "?") {
		return "", false, false
	}
	closing := strings.HasPrefix(token, "/")
	if closing {
		token = strings.TrimSpace(strings.TrimPrefix(token, "/"))
	}
	selfClosing := strings.HasSuffix(token, "/")
	if selfClosing {
		token = strings.TrimSpace(strings.TrimSuffix(token, "/"))
	}
	nameEnd := 0
	for nameEnd < len(token) && isPromptTagNameByte(token[nameEnd]) {
		nameEnd++
	}
	if nameEnd == 0 {
		return "", false, false
	}
	return token[:nameEnd], closing, selfClosing
}

func isPromptTagNameByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_' || value == '-'
}

func extractTranscriptTaggedSection(text, tag string) string {
	openingTag := "<" + tag + ">"
	closingTag := "</" + tag + ">"
	start := strings.Index(text, openingTag)
	if start < 0 {
		return ""
	}
	start += len(openingTag)
	end := strings.Index(text[start:], closingTag)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(text[start : start+end])
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
			"kind":                          payload.SourceKind,
			"source_path":                   payload.SourcePath,
			"gen_metadata_index":            payload.SnapshotGenIndex,
			"snapshot_bytes":                payload.SnapshotBytes,
			"input_boundary_step_index":     payload.SnapshotInputBoundaryStep,
			"has_input_boundary_step_index": payload.SnapshotHasInputBoundary,
			"warning":                       "This is Heimdall's evidence view, not a captured HTTP request or an official GenerateContentRequest body.",
		},
		"provenance": payload.Provenance,
		"context_snapshot": map[string]interface{}{
			"available":                 payload.SnapshotAvailable,
			"system_prompt_text":        payload.SystemPrompt,
			"tool_entries":              payload.NativeTools,
			"field_2_occurrence_count":  payload.PersistedContextEntryCount,
			"persisted_context_records": payload.PersistedRecords,
		},
		"latest_persisted_usage_observation": map[string]interface{}{
			"context_tokens": payload.TotalTokens,
			"context_limit":  payload.ContextLimit,
		},
		"transcript_observations": map[string]interface{}{
			"checkpoint_summary":    payload.CheckpointSummary,
			"checkpoint_step_index": payload.CheckpointStepIndex,
			"active_events":         payload.ActiveHistoryTurns,
			"latest_prompt":         payload.LatestPrompt,
			"staged_local_content":  payload.StagedBuffers,
		},
	})
}

// Subcategory identifiers in the persisted model-input tree.
const (
	SubcatSystemPrompt = iota
	SubcatTools
	SubcatAnchor
	SubcatCurrentHistory
)

const SubcatLast = SubcatCurrentHistory

// SerializeSubcategoryRaw serializes only the selected persisted model-input content.
func SerializeSubcategoryRaw(payload AgentContextPayload, subcatIndex int) (string, error) {
	var rawObj interface{}

	switch subcatIndex {
	case SubcatSystemPrompt:
		rawObj = map[string]interface{}{
			"text": payload.SystemPrompt,
		}

	case SubcatTools:
		rawObj = map[string]interface{}{
			"decoded_tool_entries": payload.NativeTools,
		}

	case SubcatAnchor:
		var checkpoint *PersistedContextRecord
		for index := range payload.PersistedRecords {
			if payload.PersistedRecords[index].IsCompactedCheckpoint {
				checkpoint = &payload.PersistedRecords[index]
				break
			}
		}
		rawObj = map[string]interface{}{
			"checkpoint": checkpoint,
		}

	case SubcatCurrentHistory:
		rawObj = map[string]interface{}{
			"active_events": persistedActiveRecords(payload.PersistedRecords),
		}

	default:
		return "", fmt.Errorf("unknown model-input subcategory: %d", subcatIndex)
	}

	return marshalJSONNoEscape(rawObj)
}

// SerializeSystemPromptSectionRaw serializes one top-level prompt section.
func SerializeSystemPromptSectionRaw(payload AgentContextPayload, index int) (string, error) {
	if index < 0 || index >= len(payload.SystemPromptSections) {
		return "", fmt.Errorf("unknown system-prompt section: %d", index)
	}
	return marshalJSONNoEscape(payload.SystemPromptSections[index])
}

// SerializePersistedActiveEventRaw serializes either all persisted active events
// or one selected event without falling back to the full context payload.
func SerializePersistedActiveEventRaw(payload AgentContextPayload, position int) (string, error) {
	activeRecords := persistedActiveRecords(payload.PersistedRecords)
	if position == 0 {
		return marshalJSONNoEscape(map[string]interface{}{
			"active_events": activeRecords,
		})
	}
	for _, record := range activeRecords {
		if record.Position == position {
			return marshalJSONNoEscape(map[string]interface{}{
				"active_event": record,
			})
		}
	}
	return marshalJSONNoEscape(map[string]interface{}{
		"active_event": nil,
	})
}

func persistedActiveRecords(records []PersistedContextRecord) []PersistedContextRecord {
	if len(records) > 0 && records[0].IsCompactedCheckpoint {
		return records[1:]
	}
	return records
}
