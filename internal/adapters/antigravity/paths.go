package antigravity

import "path/filepath"

const (
	antigravityDirectoryName = "antigravity-cli"
	geminiDirectoryName      = ".gemini"
)

// InstallationPath returns the root of one user's Antigravity installation.
func InstallationPath(home string) string {
	return filepath.Join(home, geminiDirectoryName, antigravityDirectoryName)
}

// ConversationDatabasePath returns the SQLite database path for one session.
func ConversationDatabasePath(home, sessionID string) string {
	return filepath.Join(InstallationPath(home), "conversations", sessionID+".db")
}

// TranscriptPaths returns the full and compact transcript paths for one session.
func TranscriptPaths(home, sessionID string) (fullPath, compactPath string) {
	logsPath := filepath.Join(InstallationPath(home), "brain", sessionID, ".system_generated", "logs")
	return filepath.Join(logsPath, "transcript_full.jsonl"), filepath.Join(logsPath, "transcript.jsonl")
}

// BuiltinSkillsPath returns Antigravity's bundled skills directory.
func BuiltinSkillsPath(home string) string {
	return filepath.Join(InstallationPath(home), "builtin", "skills")
}

// SettingsPath returns Antigravity's primary settings file path.
func SettingsPath(home string) string {
	return filepath.Join(InstallationPath(home), "settings.json")
}
