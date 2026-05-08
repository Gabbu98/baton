package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type sessionMeta struct {
	ResumeId string `json:"resumeId"`
}

func markdownRespectiveJsonPath(chatsDir, chatName string) string {
	return filepath.Join(chatsDir, chatName+".json")
}

func LoadResumeId(chatsDir, chatName string) string {
	data, err := os.ReadFile(markdownRespectiveJsonPath(chatsDir, chatName))
	if err != nil {
		return ""
	}
	var m sessionMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	return m.ResumeId
}

func SaveResumeId(chatsDir, chatName, id string) {
	data, _ := json.Marshal(sessionMeta{ResumeId: id})
	os.WriteFile(markdownRespectiveJsonPath(chatsDir, chatName), data, 0644)
}

func ClearResumeId(chatsDir, chatName string) {
	os.Remove(markdownRespectiveJsonPath(chatsDir, chatName))
}

func MdFileFor(chatName, cwd string) string {
	return filepath.Join(cwd, strings.ToUpper(chatName)+".md")
}
