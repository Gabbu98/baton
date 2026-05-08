package agents

import (
	"baton/utils"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Claude JSONL entry — message.content is a raw JSON field (string or []ContentBlock)
type claudeEntry struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type ClaudeStrategy struct {
	directory string
}

func NewClaudeStrategy(dir string) *ClaudeStrategy {
	return &ClaudeStrategy{directory: dir}
}

func (claude *ClaudeStrategy) LatestSessionID(current_working_directory string) string {
	projectKey := claude.cwdToProjectKey(current_working_directory)
	projectDir := filepath.Join(claude.directory, "projects", projectKey)

	sessions := utils.JsonFiles(projectDir)
	if len(sessions) == 0 {
		sessions = utils.JsonFiles(filepath.Join(claude.directory, "projects"))
	}
	if len(sessions) == 0 {
		return ""
	}

	utils.SortByModTime(sessions)
	base := filepath.Base(sessions[0])
	return strings.TrimSuffix(base, ".jsonl")
}

func (claude *ClaudeStrategy) ExtractContext(current_working_directory string) string {
	projectKey := claude.cwdToProjectKey(current_working_directory)
	projectDir := filepath.Join(claude.directory, "projects", projectKey)

	sessions := utils.JsonFiles(projectDir)
	if len(sessions) == 0 {
		// Fallback
		sessions = utils.JsonFiles(filepath.Join(claude.directory, "projects"))
	}

	if len(sessions) == 0 {
		return ""
	}

	utils.SortByModTime(sessions)
	return claude.parseJson(sessions[0])
}

func (claude *ClaudeStrategy) cwdToProjectKey(current_working_directory string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, current_working_directory)
}

func (claude *ClaudeStrategy) parseJson(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	type msg struct{ role, text string }
	var messages []msg

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var e claudeEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Type != "user" && e.Type != "assistant" {
			continue
		}

		role := e.Message.Role
		if role == "" {
			role = e.Type
		}

		text := contextText(e.Message.Content)
		if text == "" {
			continue
		}
		// Skip internal Claude CLI meta-messages
		if strings.Contains(text, "<local-command-caveat>") ||
			strings.Contains(text, "<command-name>") ||
			strings.Contains(text, "<local-command-stdout>") ||
			strings.Contains(text, "<local-command-stderr>") {
			continue
		}

		messages = append(messages, msg{role, text})
	}

	// Keep last 6 meaningful turns
	start := len(messages) - 6
	if start < 0 {
		start = 0
	}

	var sb strings.Builder
	for _, m := range messages[start:] {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", m.role, m.text))
	}
	return sb.String()
}

// contentText extracts human-readable text from Claude's content field,
// which may be a plain string or an array of typed content blocks.
func contextText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// plain string content
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	// Array of content blocks - only extract type:"text" blocks
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
				parts = append(parts, strings.TrimSpace(b.Text))
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}
