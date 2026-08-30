package antigravity

import (
	"path/filepath"
	"testing"
)

func TestSessionPathsUseOneAntigravityInstallationRoot(t *testing.T) {
	home := filepath.Join("home", "reader")
	sessionID := "session-42"
	fullPath, compactPath := TranscriptPaths(home, sessionID)

	if InstallationPath(home) != filepath.Join(home, ".gemini", "antigravity-cli") {
		t.Errorf("unexpected installation path: %q", InstallationPath(home))
	}
	if ConversationDatabasePath(home, sessionID) != filepath.Join(home, ".gemini", "antigravity-cli", "conversations", "session-42.db") {
		t.Errorf("unexpected database path: %q", ConversationDatabasePath(home, sessionID))
	}
	if fullPath != filepath.Join(home, ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs", "transcript_full.jsonl") {
		t.Errorf("unexpected full transcript path: %q", fullPath)
	}
	if compactPath != filepath.Join(home, ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl") {
		t.Errorf("unexpected compact transcript path: %q", compactPath)
	}
}
