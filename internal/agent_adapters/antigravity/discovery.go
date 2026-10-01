package antigravity

import (
	"os"
	"path/filepath"

	"heimdall/internal/agent_adapters"
)

const (
	geminiDirectoryName       = ".gemini"
	antigravityDirectoryName  = "antigravity-cli"
	conversationsDirectory    = "conversations"
	brainDirectory            = "brain"
	systemGeneratedDirectory  = ".system_generated"
	logsDirectoryName         = "logs"
	stepsDirectoryName        = "steps"
	fullTranscriptFileName    = "transcript_full.jsonl"
	compactTranscriptFileName = "transcript.jsonl"
)

// DiscoverSessionSources resolves known Antigravity artifacts for one session.
// Missing optional artifacts are omitted; callers can expose that absence as evidence.
func DiscoverSessionSources(home, sessionID string) ([]agents.SourceRef, error) {
	root := filepath.Join(home, geminiDirectoryName, antigravityDirectoryName)
	generatedPath := filepath.Join(root, brainDirectory, sessionID, systemGeneratedDirectory)
	fullTranscript := filepath.Join(generatedPath, logsDirectoryName, fullTranscriptFileName)
	compactTranscript := filepath.Join(generatedPath, logsDirectoryName, compactTranscriptFileName)
	databasePath := filepath.Join(root, conversationsDirectory, sessionID+".db")
	stepOutputsPath := filepath.Join(generatedPath, stepsDirectoryName)
	sources := make([]agents.SourceRef, 0, sourceCapacity)
	if pathExists(fullTranscript) {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindTranscript, Path: fullTranscript})
	} else if pathExists(compactTranscript) {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindTranscript, Path: compactTranscript})
	}
	if pathExists(databasePath) {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})
	}
	if pathExists(stepOutputsPath) {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindArtifacts, Path: stepOutputsPath})
	}
	return sources, nil
}

const sourceCapacity = 3

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func discoverStepOutputsFromTranscript(transcriptPath string) (agents.SourceRef, bool) {
	logsPath := filepath.Dir(transcriptPath)
	systemGeneratedPath := filepath.Dir(logsPath)
	if filepath.Base(logsPath) != logsDirectoryName || filepath.Base(systemGeneratedPath) != systemGeneratedDirectory {
		return agents.SourceRef{}, false
	}
	stepOutputsPath := filepath.Join(systemGeneratedPath, stepsDirectoryName)
	if !pathExists(stepOutputsPath) {
		return agents.SourceRef{}, false
	}
	return agents.SourceRef{Kind: agents.SourceKindArtifacts, Path: stepOutputsPath}, true
}
