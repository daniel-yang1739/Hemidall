package antigravity

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	_ "modernc.org/sqlite"

	"heimdall/internal/core"
)

var (
	userRequestRegex = regexp.MustCompile(`(?s)<USER_REQUEST>(.*?)</USER_REQUEST>`)
	userObjRegex     = regexp.MustCompile(`(?m)^#\s*USER Objective:\s*(.+)$`)
	modelSelectRegex = regexp.MustCompile(`(?i)Model Selection[` + "`" + `'"]*\s+from\s+.*?\s+to\s+(.+?)(?:\.\s+No need|\.\s+If reporting|\n|$)`)
	dirPathRegex     = regexp.MustCompile(`"DirectoryPath"\s*:\s*"\\?"?([^"\\]+)`)
	absPathRegex     = regexp.MustCompile(`"AbsolutePath"\s*:\s*"\\?"?([^"\\]+)`)
	userWorkspacesRe = regexp.MustCompile(`(?s)<user_information>.*?workspaces.*?(/[^ \n\r\t]+)`)
)

// CleanModelName validates and normalizes raw model string to a clean, authoritative model name
func CleanModelName(raw string) string {
	raw = strings.TrimSpace(raw)
	low := strings.ToLower(raw)
	if len(raw) > 35 || strings.Contains(low, "comment") || strings.Contains(low, "need to") || strings.Contains(low, "user") {
		return "Gemini 3.7 Flash"
	}
	if strings.Contains(low, "flash") {
		if strings.Contains(low, "high") {
			return "Gemini 3.7 Flash (High)"
		}
		return "Gemini 3.7 Flash"
	}
	if strings.Contains(low, "pro") {
		return "Gemini 2.5 Pro"
	}
	if strings.Contains(low, "claude") {
		if strings.Contains(low, "sonnet") {
			return "Claude 3.7 Sonnet"
		}
		return "Claude 3.5 Sonnet"
	}
	if raw == "" || strings.EqualFold(raw, "none") {
		return "Gemini 3.7 Flash"
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
	tagRe := regexp.MustCompile(`<[^>]+>`)
	raw = tagRe.ReplaceAllString(raw, " ")

	// Replace newlines and tabs with single space
	fields := strings.Fields(raw)
	return strings.Join(fields, " ")
}

// ExtractSessionMetadata performs lightweight head and tail scanning of a session transcript
func ExtractSessionMetadata(transcriptPath string) (initialGoal string, lastPrompt string, workspaceDir string, modelName string) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return "", "", "", ""
	}
	defer file.Close()

	var headLines []string
	var allUserInputs []string

	scanner := bufio.NewScanner(file)
	// Buffer size up to 1MB per line for transcripts
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if lineCount < 10 {
			headLines = append(headLines, line)
		}
		lineCount++

		var rawMap map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rawMap); err == nil {
			stepType, _ := rawMap["type"].(string)
			if stepType == "USER_INPUT" {
				if content, ok := rawMap["content"].(string); ok && content != "" {
					cleaned := CleanPromptText(content)
					if cleaned != "" {
						allUserInputs = append(allUserInputs, cleaned)
					}
				}
			}
		}
	}

	// 1. Initial Goal
	if len(allUserInputs) > 0 {
		initialGoal = allUserInputs[0]
		lastPrompt = allUserInputs[len(allUserInputs)-1]
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

	if modelName == "" {
		modelName = "Gemini 3.7 Flash"
	}
	if workspaceDir == "" {
		workspaceDir = "workspace"
	}

	return initialGoal, lastPrompt, workspaceDir, modelName
}

// DiscoverAllSessions scans ~/.gemini/antigravity-cli/conversations and discovers all active/past sessions
func DiscoverAllSessions() ([]core.SessionInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	geminiDir := filepath.Join(home, ".gemini", "antigravity-cli")
	dbDir := filepath.Join(geminiDir, "conversations")
	brainDir := filepath.Join(geminiDir, "brain")

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
		fullDBPath := filepath.Join(dbDir, entry.Name())
		fullLogPath := filepath.Join(brainDir, sid, ".system_generated", "logs", "transcript_full.jsonl")
		compactLogPath := filepath.Join(brainDir, sid, ".system_generated", "logs", "transcript.jsonl")

		// Prefer compact transcript.jsonl for fast metadata extraction
		scanPath := compactLogPath
		if _, err := os.Stat(compactLogPath); err != nil {
			scanPath = fullLogPath
		}

		count := 0
		db, err := sql.Open("sqlite", "file:"+fullDBPath+"?mode=ro")
		if err == nil {
			_ = db.QueryRow("SELECT count(*) FROM steps").Scan(&count)
			_ = db.Close()
		}

		initialGoal, lastPrompt, workspaceDir, modelName := ExtractSessionMetadata(scanPath)

		sessions = append(sessions, core.SessionInfo{
			AgentType:    core.AgentTypeAntigravity,
			SessionID:    sid,
			WorkspaceDir: workspaceDir,
			ShortPath:    FormatShortPath(workspaceDir),
			InitialGoal:  initialGoal,
			LastPrompt:   lastPrompt,
			StepCount:    count,
			LastModified: info.ModTime(),
			SizeMB:       float64(info.Size()) / (1024 * 1024),
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

// GetLatestActiveSession returns the most recently modified Antigravity session
func GetLatestActiveSession() (*core.SessionInfo, error) {
	sessions, err := DiscoverAllSessions()
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no active or historical antigravity sessions found")
	}
	return &sessions[0], nil
}

// LoadSessionHistory loads all historical events for a specific session ID
func LoadSessionHistory(sessionID string, analyzer *core.PayloadAnalyzer) ([]core.UnifiedAgentEvent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	geminiDir := filepath.Join(home, ".gemini", "antigravity-cli")
	dbPath := filepath.Join(geminiDir, "conversations", sessionID+".db")
	logPath := filepath.Join(geminiDir, "brain", sessionID, ".system_generated", "logs", "transcript_full.jsonl")

	sqliteReader := NewSQLiteTelemetryReader(dbPath)
	_ = sqliteReader.PollLatest()

	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session log %s: %w", logPath, err)
	}
	defer file.Close()

	var events []core.UnifiedAgentEvent
	reader := bufio.NewReader(file)

	watcher := NewWatcher(logPath, sessionID, analyzer, dbPath)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					ev, parseErr := watcher.parseLine(line)
					if parseErr == nil {
						analyzer.AnalyzeStep(&ev)
						events = append(events, ev)
					}
				}
				break
			}
			return events, err
		}
		ev, parseErr := watcher.parseLine(line)
		if parseErr == nil {
			analyzer.AnalyzeStep(&ev)
			events = append(events, ev)
		}
	}

	events = MergeMissingSQLiteSteps(events, dbPath, sessionID)
	core.BackfillPackagedIn(events)

	return events, nil
}

// MergeMissingSQLiteSteps reads SQLite steps table and inserts any internal/subagent steps missing from transcript
func MergeMissingSQLiteSteps(events []core.UnifiedAgentEvent, dbPath string, sessionID string) []core.UnifiedAgentEvent {
	if dbPath == "" {
		return events
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_journal=WAL")
	if err != nil {
		return events
	}
	defer db.Close()

	existingIndices := make(map[int]bool, len(events))
	for _, e := range events {
		existingIndices[e.StepIndex] = true
	}

	rows, err := db.Query("SELECT idx, step_type, status, metadata FROM steps ORDER BY idx ASC")
	if err != nil {
		return events
	}
	defer rows.Close()

	toolRegex := regexp.MustCompile(`(write_to_file|view_file|run_command|list_directory|read_url_content|ask_question)`)
	summaryRegex := regexp.MustCompile(`"toolSummary"\s*:\s*"([^"]+)"`)

	var merged []core.UnifiedAgentEvent
	merged = append(merged, events...)

	uuidRegex := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

	for rows.Next() {
		var idx, stepType, status int
		var metadata []byte
		if err := rows.Scan(&idx, &stepType, &status, &metadata); err != nil {
			continue
		}
		if existingIndices[idx] {
			continue
		}

		toolName := "internal"
		summary := fmt.Sprintf("Internal background execution #%d", idx)
		isSub := false
		agentRole := "INTERNAL"

		if len(metadata) > 0 {
			metaStr := string(metadata)
			uuids := uuidRegex.FindAllString(metaStr, -1)
			for _, u := range uuids {
				if u != sessionID {
					isSub = true
					agentRole = "SUBAGENT"
					break
				}
			}
			if m := toolRegex.FindStringSubmatch(metaStr); len(m) > 1 {
				toolName = m[1]
			}
			if m := summaryRegex.FindStringSubmatch(metaStr); len(m) > 1 {
				summary = m[1]
			} else if toolName != "internal" {
				summary = fmt.Sprintf("Execute %s", toolName)
			}
		}

		statusLabel := "DONE"
		stepScope := core.ScopeLocalExecution
		if isSub {
			stepScope = core.ScopeSubagent
		}
		stType := core.StepTypeRunCommand
		if status == 7 {
			statusLabel = "BLOCKED"
			summary = fmt.Sprintf("[BLOCKED] %s", summary)
		}
		if toolName == "view_file" {
			stType = core.StepTypeViewFile
		} else if toolName == "write_to_file" {
			stType = core.StepTypeCodeAction
		} else if toolName == "list_directory" {
			stType = core.StepTypeListDirectory
		}

		internalEvent := core.UnifiedAgentEvent{
			SessionID:  sessionID,
			StepIndex:  idx,
			Source:     "SYSTEM",
			Type:       stType,
			Status:     statusLabel,
			Scope:      stepScope,
			AgentRole:  agentRole,
			IsSubagent: isSub,
			Summary:    summary,
			Tokens: core.TokenBreakdown{
				TotalTokens:      0,
				CachedTokens:     0,
				NewTokens:        0,
				IsOfficialData:   false,
				ToolResultTokens: 0,
			},
		}
		merged = append(merged, internalEvent)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].StepIndex < merged[j].StepIndex
	})

	return merged
}
