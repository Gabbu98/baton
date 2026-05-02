package agents

import (
	"baton/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GeminiStrategy struct {
	directory string
}

// finds most recently modified session JSON
func (gemini *GeminiStrategy) ExtractContext(common_working_directory string) string {
	projectName := filepath.Base(common_working_directory)
	projectDir := filepath.Join(gemini.directory, "tmp", projectName, "chats")

	entries, err := os.ReadDir(projectDir)
	var sessionFiles []string
	if err != nil {
		// Fallback: search for any session in ~/.gemini/tmp/*/chats/

		var geminiTempPath string = filepath.Join(gemini.directory, "tmp")
		filepath.WalkDir(geminiTempPath, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".json") && strings.Contains(path, "/chats/") {
				sessionFiles = append(sessionFiles, path)
			}
			return nil
		})

		if len(sessionFiles) == 0 {
			return ""
		}

		utils.SortByModTime(sessionFiles)
		return parseJson(sessionFiles[0])
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			sessionFiles = append(sessionFiles, filepath.Join(projectDir, e.Name()))
		}
	}

	if len(sessionFiles) == 0 {
		return ""
	}

	utils.SortByModTime(sessionFiles)
	return parseJson(sessionFiles[0])
}

func (gemini *GeminiStrategy) parseJson(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var session struct {
		Messages []struct {
			Type    string      `json:"type"`
			Content interface{} `json:"content"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(data, &session); err != nil {
		return ""
	}

	type msg struct{ role, text string }
	var messages []msg

	for _, m := range session.Messages {
		text := ""
		switch v := m.Content.(type) {
		case string:
			text = v
		case []interface{}:
			for _, item := range v {
				if obj, ok := item.(map[string]interface{}); ok {
					if t, ok := obj["text"].(string); ok {
						text += t
					}
				}
			}
		}

		if text != "" {
			role := m.Type
			if role == "gemini" {
				role = "assistant"
			}
			messages = append(messages, msg{role, text})
		}
	}

	// keep last 6 meaningful turns
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
