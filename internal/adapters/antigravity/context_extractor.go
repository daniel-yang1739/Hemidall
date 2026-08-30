package antigravity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"heimdall/internal/core"
)

// ExtractAgentContextPayload loads Antigravity's persisted context snapshot when available.
// Transcript-derived data remains an explicitly labeled fallback for live or legacy sessions.
func ExtractAgentContextPayload(history []core.UnifiedAgentEvent, sessionID, targetModel string) core.AgentContextPayload {
	if targetModel == "" {
		targetModel = latestObservedModel(history)
	}
	cwd, _ := os.Getwd()
	payload := core.BuildContextPayloadFromHistory(core.ContextBuildInput{
		History:      history,
		SessionID:    sessionID,
		AgentType:    core.AgentTypeAntigravity,
		TargetModel:  targetModel,
		NativeTools:  observedTools(history),
		ActiveSkills: discoveredSkills(cwd),
		RuntimeMetadata: map[string]string{
			"OS":        runtime.GOOS,
			"Arch":      runtime.GOARCH,
			"Shell":     os.Getenv("SHELL"),
			"Cwd":       cwd,
			"SessionID": sessionID,
			"Model":     targetModel,
		},
		Provenance: map[string]string{
			"checkpoint":     "observed_from_antigravity_session_history",
			"active_turns":   "observed_from_antigravity_session_history",
			"latest_prompt":  "observed_from_antigravity_session_history",
			"staged_buffers": "inferred_from_local_steps_after_last_cloud_step",
			"tokens":         "latest_positive_history_token_breakdown; may be telemetry or Heimdall-derived",
			"runtime":        "observed_from_heimdall_process_environment",
			"tools":          "inferred_from_observed_tool_arguments",
			"skills":         "discovered_from_local_skill_directories",
		},
	})
	payload.SourceKind = core.ContextEvidenceTranscriptFallback
	snapshot, err := LoadContextSnapshot(sessionID)
	if err != nil {
		return payload
	}
	home, _ := os.UserHomeDir()
	payload.SourceKind = core.ContextEvidencePersistedSnapshot
	payload.SourcePath = filepath.Join(home, ".gemini", "antigravity-cli", "conversations", sessionID+".db")
	payload.SnapshotAvailable = true
	payload.SnapshotGenIndex = snapshot.GenIndex
	payload.SnapshotBytes = snapshot.BlobBytes
	payload.SystemPrompt = snapshot.SystemPrompt
	payload.IdentityPrompt = snapshot.Identity
	payload.ConstitutionDoc = snapshot.UserRules
	payload.SkillsSection = snapshot.SkillsSection
	payload.MCPSection = snapshot.MCPSection
	payload.PersistedContextEntryCount = snapshot.PersistedContextEntryCount
	payload.PersistedRecords = snapshot.PersistedRecords
	payload.NativeTools = snapshot.Tools
	payload.ActiveSkills = nil
	payload.MCPServers = nil
	payload.Provenance = map[string]string{
		"system_prompt":    "persisted_gen_metadata_field_1.1",
		"identity":         "parsed_from_system_prompt_identity_section",
		"user_rules":       "parsed_from_system_prompt_user_rules_section",
		"skills":           "persisted_system_prompt_skills_section",
		"tools":            "persisted_gen_metadata_field_1.8_repeated",
		"history_snapshot": "persisted_gen_metadata_field_1.2_occurrence_count",
		"mcp":              "not_attributable_without_official_protobuf_schema",
		"checkpoint":       "observed_from_antigravity_session_history",
		"active_turns":     "observed_from_antigravity_session_history",
		"latest_prompt":    "observed_from_antigravity_session_history",
		"staged_buffers":   "inferred_from_local_steps_after_last_cloud_step",
		"tokens":           "latest_positive_history_token_breakdown; may be telemetry or Heimdall-derived",
		"runtime":          "observed_from_heimdall_process_environment",
	}
	return payload
}

func observedTools(history []core.UnifiedAgentEvent) []core.ToolSignature {
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
	var names []string
	for name := range observed {
		names = append(names, name)
	}
	sort.Strings(names)

	tools := make([]core.ToolSignature, 0, len(names))
	for _, name := range names {
		var keys []string
		for key := range observed[name] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		raw, _ := json.MarshalIndent(map[string]interface{}{"name": name, "observedArgumentKeys": keys}, "", "  ")
		tools = append(tools, core.ToolSignature{
			Name:        name,
			Signature:   fmt.Sprintf("%s(%s)", name, strings.Join(keys, ", ")),
			Description: "Observed in the Antigravity transcript; authoritative schema unavailable.",
			RawSchema:   string(raw),
		})
	}
	return tools
}

func discoveredSkills(cwd string) []core.SkillInfo {
	home, _ := os.UserHomeDir()
	roots := []string{
		findNearestDirectory(cwd, filepath.Join(".agents", "skills")),
		filepath.Join(home, ".gemini", "antigravity-cli", "builtin", "skills"),
	}
	seen := make(map[string]struct{})
	var skills []core.SkillInfo
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			path := filepath.Join(root, entry.Name(), "SKILL.md")
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			name, description := parseSkillFrontmatter(string(content), entry.Name())
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			skills = append(skills, core.SkillInfo{
				Name: name, Status: "DISCOVERED", Path: path,
				Description: description, RawMarkdown: string(content),
			})
		}
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills
}

func findNearestDirectory(startDir, relativePath string) string {
	for dir := startDir; dir != ""; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, relativePath)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}

func parseSkillFrontmatter(markdown, fallbackName string) (string, string) {
	name := fallbackName
	description := ""
	for _, line := range strings.Split(markdown, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			name = strings.Trim(strings.TrimSpace(value), `"'`)
		case "description":
			description = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return name, description
}

func latestObservedModel(history []core.UnifiedAgentEvent) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Tokens.OfficialModel != "" {
			return history[i].Tokens.OfficialModel
		}
	}
	return ""
}
