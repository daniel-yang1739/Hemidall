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
	generatedLogsDirectory    = ".system_generated/logs"
	fullTranscriptFileName    = "transcript_full.jsonl"
	compactTranscriptFileName = "transcript.jsonl"
)

// DiscoverSessionSources resolves known Antigravity artifacts for one session.
// Missing optional artifacts are omitted; callers can expose that absence as evidence.
func DiscoverSessionSources(home, sessionID string) ([]agents.SourceRef, error) {
	root := filepath.Join(home, geminiDirectoryName, antigravityDirectoryName)
	fullTranscript := filepath.Join(root, brainDirectory, sessionID, generatedLogsDirectory, fullTranscriptFileName)
	compactTranscript := filepath.Join(root, brainDirectory, sessionID, generatedLogsDirectory, compactTranscriptFileName)
	databasePath := filepath.Join(root, conversationsDirectory, sessionID+".db")
	sources := make([]agents.SourceRef, 0, sourceCapacity)
	if pathExists(fullTranscript) {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindTranscript, Path: fullTranscript})
	} else if pathExists(compactTranscript) {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindTranscript, Path: compactTranscript})
	}
	if pathExists(databasePath) {
		sources = append(sources, agents.SourceRef{Kind: agents.SourceKindConversation, Path: databasePath})
	}
	return sources, nil
}

const sourceCapacity = 2

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
