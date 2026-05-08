package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// maps agent names ("claude", "gemini") to their last session id
type sessionMeta map[string]string

func markdownRespectiveJsonPath(chatsDir, chatName string) string {
	return filepath.Join(chatsDir, chatName+".json")
}

func LoadResumeId(chatsDir, chatName, agent string) string {
	data, err := os.ReadFile(markdownRespectiveJsonPath(chatsDir, chatName))
	if err != nil {
		return ""
	}
	var m sessionMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	return m[agent]
}

func SaveResumeId(chatsDir, chatName, agent, id string) {
	path := markdownRespectiveJsonPath(chatsDir, chatName)
	m := make(sessionMeta)

	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &m)
	}

	m[agent] = id
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(path, data, 0644)
}

func ClearResumeId(chatsDir, chatName, agent string) {
	path := markdownRespectiveJsonPath(chatsDir, chatName)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	m := make(sessionMeta)
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}

	if _, ok := m[agent]; ok {
		delete(m, agent)
		if len(m) == 0 {
			_ = os.Remove(path)
		} else {
			newData, err := json.MarshalIndent(m, "", "  ")
			if err != nil {
				return
			}
			_ = os.WriteFile(path, newData, 0644)
		}
	}
}

func MdFileFor(chatName, cwd string) string {
	return filepath.Join(cwd, strings.ToUpper(chatName)+".md")
}
