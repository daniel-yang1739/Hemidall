package antigravity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"heimdall/internal/agent_adapters"
	"heimdall/internal/core"
)

const (
	conversationDatabaseSuffix = ".db"
	summaryDatabaseName        = "conversation_summaries.db"
	bytesPerMegabyte           = 1_048_576.0
	catalogMetadataLineLimit   = 10
	catalogScannerBufferBytes  = 1_048_576
	defaultWorkspaceName       = "workspace"
)

type catalogTranscriptLine struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// DiscoverAllSessions returns a lightweight session catalog without opening
// every database. It is safe to call before a session is selected.
func DiscoverAllSessions(home string) ([]core.SessionInfo, error) {
	databaseDirectory := filepath.Join(home, geminiDirectoryName, antigravityDirectoryName, conversationsDirectory)
	entries, err := os.ReadDir(databaseDirectory)
	if err != nil {
		return nil, err
	}
	sessions := make([]core.SessionInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), conversationDatabaseSuffix) || entry.Name() == summaryDatabaseName {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		sessionID := strings.TrimSuffix(entry.Name(), conversationDatabaseSuffix)
		sources, sourceErr := DiscoverSessionSources(home, sessionID)
		if sourceErr != nil {
			continue
		}
		transcriptPath := sourcePath(sources, agents.SourceKindTranscript)
		initialGoal := readInitialGoal(transcriptPath)
		sessions = append(sessions, core.SessionInfo{
			SessionID:    sessionID,
			WorkspaceDir: defaultWorkspaceName,
			ShortPath:    FormatShortPath(defaultWorkspaceName),
			InitialGoal:  initialGoal,
			LastPrompt:   initialGoal,
			LastModified: info.ModTime(),
			SizeMB:       float64(info.Size()) / bytesPerMegabyte,
			DBPath:       sourcePath(sources, agents.SourceKindConversation),
			LogPath:      transcriptPath,
		})
	}
	sort.Slice(sessions, func(left, right int) bool {
		return sessions[left].LastModified.After(sessions[right].LastModified)
	})
	return sessions, nil
}

// DiscoverLatestSession resolves the most recently modified Antigravity session.
func DiscoverLatestSession(home string) (*core.SessionInfo, error) {
	sessions, err := DiscoverAllSessions(home)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no Antigravity sessions found")
	}
	return &sessions[0], nil
}

func sourcePath(sources []agents.SourceRef, kind agents.SourceKind) string {
	for _, source := range sources {
		if source.Kind == kind {
			return source.Path
		}
	}
	return ""
}

// FormatShortPath reduces a workspace location to a compact human-readable label.
func FormatShortPath(fullPath string) string {
	if fullPath == "" {
		return defaultWorkspaceName
	}
	cleanPath := filepath.Clean(fullPath)
	parts := strings.Split(cleanPath, string(filepath.Separator))
	nonEmptyParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	if len(nonEmptyParts) == 0 {
		return defaultWorkspaceName
	}
	if len(nonEmptyParts) <= 2 {
		return strings.Join(nonEmptyParts, "/")
	}
	return strings.Join(nonEmptyParts[len(nonEmptyParts)-2:], "/")
}

func readInitialGoal(transcriptPath string) string {
	if transcriptPath == "" {
		return ""
	}
	file, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, catalogScannerBufferBytes)
	scanner.Buffer(buffer, catalogScannerBufferBytes)
	for lineCount := 0; lineCount < catalogMetadataLineLimit && scanner.Scan(); lineCount++ {
		var line catalogTranscriptLine
		if json.Unmarshal(scanner.Bytes(), &line) != nil || line.Type != string(core.StepTypeUserInput) {
			continue
		}
		return strings.Join(strings.Fields(line.Content), " ")
	}
	return ""
}
