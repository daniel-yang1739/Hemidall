package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverSkillDefinitions reads SKILL.md files from explicitly supplied roots.
// Callers choose the roots so core does not assume an agent-specific installation path.
func DiscoverSkillDefinitions(roots ...string) []SkillInfo {
	seen := make(map[string]struct{})
	var skills []SkillInfo
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
			name, description := parseSkillDefinition(string(content), entry.Name())
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			skills = append(skills, SkillInfo{
				Name:        name,
				Status:      "DISCOVERED",
				Path:        path,
				Description: description,
				RawMarkdown: string(content),
			})
		}
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills
}

// FindNearestDirectory resolves a relative directory by walking from startDir to its filesystem root.
func FindNearestDirectory(startDir, relativePath string) string {
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

func parseSkillDefinition(markdown, fallbackName string) (string, string) {
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
