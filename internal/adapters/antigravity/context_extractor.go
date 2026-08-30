package antigravity

import (
	"os"
	"path/filepath"
	"runtime"

	"heimdall/internal/core"
)

// ExtractAgentContextPayload loads Antigravity's persisted context snapshot when available.
// Transcript-derived data remains an explicitly labeled fallback for live or legacy sessions.
func ExtractAgentContextPayload(history []core.UnifiedAgentEvent, sessionID, targetModel string) core.AgentContextPayload {
	if targetModel == "" {
		targetModel = latestObservedModel(history)
	}
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	payload := core.BuildContextPayloadFromHistory(core.ContextBuildInput{
		History:     history,
		SessionID:   sessionID,
		AgentType:   core.AgentTypeAntigravity,
		TargetModel: targetModel,
		NativeTools: core.GetNativeToolsDefinitions(history),
		ActiveSkills: core.DiscoverSkillDefinitions(
			core.FindNearestDirectory(cwd, filepath.Join(".agents", "skills")),
			BuiltinSkillsPath(home),
		),
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
	payload.SourceKind = core.ContextEvidencePersistedSnapshot
	payload.SourcePath = snapshot.SourcePath
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

func latestObservedModel(history []core.UnifiedAgentEvent) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Tokens.OfficialModel != "" {
			return history[i].Tokens.OfficialModel
		}
	}
	return ""
}
