package antigravity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"heimdall/internal/core"
)

const (
	// maxScanBufferSize defines the maximum memory buffer (1MB) allocated for scanning long JSONL lines
	maxScanBufferSize = 1024 * 1024

	// bytesPerMegabyte is the conversion factor from raw bytes to megabytes
	bytesPerMegabyte = 1024.0 * 1024.0

	// sessionMetadataHeadLines is enough to locate bootstrap metadata without
	// parsing an entire transcript during session-switcher discovery.
	sessionMetadataHeadLines = 10

	// sessionMetadataTailBytes bounds list-view metadata work for large sessions.
	sessionMetadataTailBytes = 1_024 * 1_024
)

var (
	userRequestRegex = regexp.MustCompile(`(?s)<USER_REQUEST>(.*?)</USER_REQUEST>`)
	userObjRegex     = regexp.MustCompile(`(?m)^#\s*USER Objective:\s*(.+)$`)
	modelSelectRegex = regexp.MustCompile(`(?i)Model Selection[` + "`" + `'"]*\s+from\s+.*?\s+to\s+(.+?)(?:\.\s+No need|\.\s+If reporting|\n|$)`)
	dirPathRegex     = regexp.MustCompile(`"DirectoryPath"\s*:\s*"\\?"?([^"\\]+)`)
	absPathRegex     = regexp.MustCompile(`"AbsolutePath"\s*:\s*"\\?"?([^"\\]+)`)
	userWorkspacesRe = regexp.MustCompile(`(?s)<user_information>.*?workspaces.*?(/[^ \n\r\t]+)`)
	tagRe            = regexp.MustCompile(`<[^>]+>`)
)

type transcriptMetadataLine struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// CleanModelName returns an observed session-declaration model string only when
// the extracted value is plausibly a model identifier. It never supplies a
// fallback model name because that would mislabel unknown persisted usage.
func CleanModelName(raw string) string {
	raw = strings.TrimSpace(raw)
	low := strings.ToLower(raw)
	if len(raw) > 35 || strings.Contains(low, "comment") || strings.Contains(low, "need to") || strings.Contains(low, "user") {
		return ""
	}
	if raw == "" || strings.EqualFold(raw, "none") {
		return ""
	}
	return raw
}

// FormatShortPath formats a full path to its trailing 2-3 readable directory segments
func FormatShortPath(fullPath string) string {
	if fullPath == "" {
		return "workspace"
	}
	clean := filepath.Clean(fullPath)
	parts := strings.Split(clean, string(filepath.Separator))
	var valid []string
	for _, p := range parts {
		if p != "" {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return "workspace"
	}
	if len(valid) <= 2 {
		return strings.Join(valid, "/")
	}
	return strings.Join(valid[len(valid)-2:], "/")
}

// CleanPromptText cleans raw prompt strings by stripping XML tags and condensing whitespaces
func CleanPromptText(raw string) string {
	if raw == "" {
		return ""
	}
	if m := userRequestRegex.FindStringSubmatch(raw); len(m) > 1 {
		raw = m[1]
	} else if m := userObjRegex.FindStringSubmatch(raw); len(m) > 1 {
		raw = m[1]
	}
	// Strip any remaining XML/HTML tags
	raw = tagRe.ReplaceAllString(raw, " ")

	// Replace newlines and tabs with single space
	fields := strings.Fields(raw)
	return strings.Join(fields, " ")
}

// ExtractSessionMetadata reads only the bootstrap head and a bounded tail of a
// session transcript. It must stay cheap because session switcher discovery
// calls it for every locally available session.
func ExtractSessionMetadata(transcriptPath string) (initialGoal string, lastPrompt string, workspaceDir string, modelName string) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return "", "", "", ""
	}
	defer file.Close()

	var headLines []string

	scanner := bufio.NewScanner(file)
	// Buffer size up to 1MB per line for transcripts
	buf := make([]byte, maxScanBufferSize)
	scanner.Buffer(buf, maxScanBufferSize)

	for lineCount := 0; lineCount < sessionMetadataHeadLines && scanner.Scan(); lineCount++ {
		line := scanner.Text()
		headLines = append(headLines, line)
		var metadata transcriptMetadataLine
		if initialGoal == "" && json.Unmarshal([]byte(line), &metadata) == nil && metadata.Type == "USER_INPUT" {
			initialGoal = CleanPromptText(metadata.Content)
		}
	}

	lastPrompt = readLatestUserPrompt(file)
	if lastPrompt == "" {
		lastPrompt = initialGoal
	}

	// 2. Workspace & Model from Head Lines
	for _, l := range headLines {
		if modelName == "" {
			if m := modelSelectRegex.FindStringSubmatch(l); len(m) > 1 {
				modelName = CleanModelName(m[1])
			}
		}
		if workspaceDir == "" {
			if m := userWorkspacesRe.FindStringSubmatch(l); len(m) > 1 {
				workspaceDir = strings.TrimSpace(m[1])
			} else if m := dirPathRegex.FindStringSubmatch(l); len(m) > 1 {
				workspaceDir = strings.TrimSpace(m[1])
			} else if m := absPathRegex.FindStringSubmatch(l); len(m) > 1 {
				workspaceDir = filepath.Dir(strings.TrimSpace(m[1]))
			}
		}
	}

	if workspaceDir == "" {
		workspaceDir = "workspace"
	}

	return initialGoal, lastPrompt, workspaceDir, modelName
}

func readLatestUserPrompt(file *os.File) string {
	info, err := file.Stat()
	if err != nil || info.Size() == 0 {
		return ""
	}
	start := info.Size() - sessionMetadataTailBytes
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return ""
	}
	tail, err := io.ReadAll(file)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(tail), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		var metadata transcriptMetadataLine
		if json.Unmarshal([]byte(lines[index]), &metadata) != nil || metadata.Type != "USER_INPUT" {
			continue
		}
		if prompt := CleanPromptText(metadata.Content); prompt != "" {
			return prompt
		}
	}
	return ""
}

// DiscoverAllSessions scans ~/.gemini/antigravity-cli/conversations and discovers all active/past sessions
func DiscoverAllSessions() ([]core.SessionInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return discoverAllSessionsAtHome(home)
}

// discoverAllSessionsAtHome creates a lightweight session-switcher catalog.
// It deliberately avoids opening every conversation database or counting every
// step, because those values are not needed before a user selects a session.
func discoverAllSessionsAtHome(home string) ([]core.SessionInfo, error) {
	installationPath := InstallationPath(home)
	dbDir := filepath.Join(installationPath, "conversations")

	entries, err := os.ReadDir(dbDir)
	if err != nil {
		return nil, err
	}

	var sessions []core.SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		if entry.Name() == "conversation_summaries.db" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		sid := strings.TrimSuffix(entry.Name(), ".db")
		fullDBPath := ConversationDatabasePath(home, sid)
		fullLogPath, compactLogPath := TranscriptPaths(home, sid)

		// Prefer compact transcript.jsonl for fast metadata extraction
		scanPath := compactLogPath
		if _, err := os.Stat(compactLogPath); err != nil {
			scanPath = fullLogPath
		}

		initialGoal, lastPrompt, workspaceDir, modelName := ExtractSessionMetadata(scanPath)

		sessions = append(sessions, core.SessionInfo{
			SessionID:    sid,
			WorkspaceDir: workspaceDir,
			ShortPath:    FormatShortPath(workspaceDir),
			InitialGoal:  initialGoal,
			LastPrompt:   lastPrompt,
			LastModified: info.ModTime(),
			SizeMB:       float64(info.Size()) / bytesPerMegabyte,
			ModelName:    modelName,
			DBPath:       fullDBPath,
			LogPath:      fullLogPath,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastModified.After(sessions[j].LastModified)
	})

	return sessions, nil
}

// GetLatestActiveSession resolves just the most recently modified conversation
// database. It intentionally avoids full catalog discovery, transcript scans,
// and per-session SQLite count queries on the application startup path.
func GetLatestActiveSession() (*core.SessionInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return discoverLatestSessionAtHome(home)
}

func discoverLatestSessionAtHome(home string) (*core.SessionInfo, error) {
	databaseDirectory := filepath.Join(InstallationPath(home), "conversations")
	entries, err := os.ReadDir(databaseDirectory)
	if err != nil {
		return nil, err
	}

	var latestEntry os.DirEntry
	var latestInfo os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") || entry.Name() == "conversation_summaries.db" {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		if latestInfo == nil || info.ModTime().After(latestInfo.ModTime()) {
			latestEntry = entry
			latestInfo = info
		}
	}
	if latestEntry == nil || latestInfo == nil {
		return nil, fmt.Errorf("no active or historical antigravity sessions found")
	}

	sessionID := strings.TrimSuffix(latestEntry.Name(), ".db")
	fullLogPath, compactLogPath := TranscriptPaths(home, sessionID)
	logPath := fullLogPath
	if _, statErr := os.Stat(logPath); statErr != nil {
		logPath = compactLogPath
	}
	initialGoal, lastPrompt, workspaceDir, modelName := ExtractSessionMetadata(logPath)
	return &core.SessionInfo{
		SessionID:    sessionID,
		WorkspaceDir: workspaceDir,
		ShortPath:    FormatShortPath(workspaceDir),
		InitialGoal:  initialGoal,
		LastPrompt:   lastPrompt,
		LastModified: latestInfo.ModTime(),
		SizeMB:       float64(latestInfo.Size()) / bytesPerMegabyte,
		ModelName:    modelName,
		DBPath:       filepath.Join(databaseDirectory, latestEntry.Name()),
		LogPath:      logPath,
	}, nil
}
