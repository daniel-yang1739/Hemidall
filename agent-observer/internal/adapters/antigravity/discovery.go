package antigravity

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"agent-observer/internal/core"
)

// SessionInfo encapsulates discovered session metadata
type SessionInfo struct {
	SessionID    string
	StepCount    int
	LastModified time.Time
	SizeMB       float64
	DBPath       string
	LogPath      string
	Title        string
}

// DiscoverAllSessions scans ~/.gemini/antigravity-cli/conversations and discovers all active/past sessions
func DiscoverAllSessions() ([]SessionInfo, error) {
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

	var sessions []SessionInfo
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

		count := 0
		db, err := sql.Open("sqlite", "file:"+fullDBPath+"?mode=ro")
		if err == nil {
			_ = db.QueryRow("SELECT count(*) FROM steps").Scan(&count)
			_ = db.Close()
		}

		sessions = append(sessions, SessionInfo{
			SessionID:    sid,
			StepCount:    count,
			LastModified: info.ModTime(),
			SizeMB:       float64(info.Size()) / (1024 * 1024),
			DBPath:       fullDBPath,
			LogPath:      fullLogPath,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastModified.After(sessions[j].LastModified)
	})

	return sessions, nil
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

	return events, nil
}
