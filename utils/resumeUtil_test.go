package utils

import (
	"os"
	"testing"
)

func TestSaveAndLoadResumeId(t *testing.T) {
	dir := t.TempDir()

	SaveResumeId(dir, "chat1", "claude", "abc123")

	id := LoadResumeId(dir, "chat1", "claude")
	if id != "abc123" {
		t.Fatalf("expected abc123, got %s", id)
	}
}

func TestSavePreserveExistingAgents(t *testing.T) {
	dir := t.TempDir()

	SaveResumeId(dir, "chat1", "claude", "123")
	SaveResumeId(dir, "chat1", "gemini", "456")

	if LoadResumeId(dir, "chat1", "claude") != "123" {
		t.Fatalf("claude does not exist anymore")
	}

	if LoadResumeId(dir, "chat1", "gemini") != "456" {
		t.Fatalf("gemini does not exist anymore")
	}
}

func TestClearResumeId(t *testing.T) {
	tempDir := t.TempDir()
	chatName := "clear-test"
	jsonPath := markdownRespectiveJsonPath(tempDir, chatName)

	SaveResumeId(tempDir, chatName, "claude", "c1")
	SaveResumeId(tempDir, chatName, "gemini", "g1")

	ClearResumeId(tempDir, chatName, "claude")
	if got := LoadResumeId(tempDir, chatName, "claude"); got != "" {
		t.Errorf("expected empty string after clearing agent, got %q", got)
	}
	if got := LoadResumeId(tempDir, chatName, "gemini"); got != "g1" {
		t.Errorf("expected other agent to remain, got %q", got)
	}

	ClearResumeId(tempDir, chatName, "gemini")
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("expected metadata file to be deleted when empty, but it still exists")
	}
}

func TestClearResumeId_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	// Should not panic or error
	ClearResumeId(tempDir, "none", "claude")
}
