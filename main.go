package main

import (
	"baton/agents"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var home = os.Getenv("HOME")

var (
	claudeDir   = envOr("BATON_CLAUDE_DIR", filepath.Join(home, ".claude"))
	geminiDir   = envOr("BATON_GEMINI_DIR", filepath.Join(home, ".gemini"))
	opencodeDir = envOr("BATON_OPENCODE_DIR", filepath.Join(home, ".local", "share", "opencode"))
	bridgeFile  = envOr("BATON_BRIDGE_FILE", filepath.Join(home, ".config", "baton", "bridge.md"))
	vaultDir    = envOr("BATON_VAULT_DIR", filepath.Join(home, "Documents", "Baton_Vault"))
	chatsDir    = envOr("BATON_CHATS_DIR", filepath.Join(home, "Baton_Chats"))
	stateDir    = envOr("BATON_STATE_DIR", filepath.Join(home, ".config", "baton", "state"))
)

const (
	batonStart = "<!-- baton:context:start -->"
	batonEnd   = "<!-- baton:context:end -->"
)

type resumeState struct {
	ResumeID string `json:"resumeId"`
	SavedAt  string `json:"savedAt"`
}

func stateFilePath(chatName, tool string) string {
	return filepath.Join(stateDir, chatName+"_"+tool+".json")
}

func loadResumeID(chatName, tool string) string {
	data, err := os.ReadFile(stateFilePath(chatName, tool))
	if err != nil {
		return ""
	}
	var s resumeState
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}
	return s.ResumeID
}

func saveResumeID(chatName, tool, id string) {
	os.MkdirAll(stateDir, 0755)
	s := resumeState{ResumeID: id, SavedAt: time.Now().Format(time.RFC3339)}
	data, _ := json.Marshal(s)
	os.WriteFile(stateFilePath(chatName, tool), data, 0644)
}

func clearResumeID(chatName, tool string) {
	os.Remove(stateFilePath(chatName, tool))
}

func sessionExists(claudeDir, sessionID, cwd string) bool {
	projectKey := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, cwd)
	path := filepath.Join(claudeDir, "projects", projectKey, sessionID+".jsonl")
	_, err := os.Stat(path)
	return err == nil
}

// Claude JSONL entry — message.content is a raw JSON field (string or []ContentBlock)
type claudeEntry struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("🪄 Baton Orchestrator\nUsage: baton [binary-name] [args...]")
		return
	}

	chatName := os.Args[1]
	targetAI := os.Args[2]
	extraArgs := os.Args[3:]

	runBaton(chatName, targetAI, extraArgs)
}

// common
func runBaton(chatName, aiCmd string, extraArgs []string) {
	cwd, _ := os.Getwd()
	os.MkdirAll(chatsDir, 0755)

	chatFile := filepath.Join(chatsDir, chatName+".md")

	var args []string
	if aiCmd == "claude" {
		id := loadResumeID(chatName, "claude")
		if id != "" && !sessionExists(claudeDir, id, cwd) {
			fmt.Printf("⚠️  Baton: Stale session ID %s..., falling back to context injection.\n", id[:8])
			clearResumeID(chatName, "claude")
			id = ""
		}
		if id != "" {
			fmt.Printf("🔁 Baton: Resuming Claude session %s...\n", id[:8])
			args = append([]string{"claude", "--resume", id}, extraArgs...)
		} else {
			if data, err := os.ReadFile(chatFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
				mdFile := mdFileFor(aiCmd, cwd)
				if err := writeMDContext(mdFile, string(data), contextPreamble(aiCmd)); err == nil {
					fmt.Printf("📝 Baton: Context injected into %s\n", filepath.Base(mdFile))
				}
			}
			args = append([]string{"claude"}, extraArgs...)
		}
	} else {
		if data, err := os.ReadFile(chatFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			mdFile := mdFileFor(aiCmd, cwd)
			if err := writeMDContext(mdFile, string(data), contextPreamble(aiCmd)); err == nil {
				fmt.Printf("📝 Baton: Context injected into %s\n", filepath.Base(mdFile))
			}
		}
		args = append([]string{aiCmd}, extraArgs...)
	}

	fmt.Printf("🚀 Baton: Launching %s...\n", aiCmd)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Error: Could not start '%s'. Ensure it is in your PATH.\n", aiCmd)
		return
	}

	extractAndSave(chatName, aiCmd, cwd)
}

// common
func mdFileFor(chatName, cwd string) string {
	return filepath.Join(cwd, strings.ToUpper(chatName)+".md")
}

// contextPreamble returns AI-specific instructions to frame the injected history.
func contextPreamble(aiCmd string) string {
	switch aiCmd {
	case "opencode":
		return "> **IMPORTANT:** The section below is prior conversation history carried over by Baton.\n" +
			"> Treat it as established context. Do not search the web for information already present here.\n" +
			"> Reference it directly when answering questions in this session.\n"
	default:
		return ""
	}
}

// common
// writeMDContext upserts the baton-fenced section at the top of the MD file.
func writeMDContext(mdFile, bridgeContent, preamble string) error {
	body := strings.TrimSpace(bridgeContent)
	if preamble != "" {
		body = preamble + "\n" + body
	}
	section := fmt.Sprintf("%s\n## Baton Context Bridge\n_Carried over: %s_\n\n%s\n%s\n",
		batonStart,
		time.Now().Format("2006-01-02 15:04"),
		body,
		batonEnd)

	existing := ""
	if raw, err := os.ReadFile(mdFile); err == nil {
		existing = removeBatonSection(string(raw))
	}

	var final string
	if strings.TrimSpace(existing) != "" {
		final = section + "\n" + existing
	} else {
		final = section
	}

	return os.WriteFile(mdFile, []byte(final), 0644)
}

// common
func removeBatonSection(content string) string {
	start := strings.Index(content, batonStart)
	if start == -1 {
		return content
	}
	end := strings.Index(content, batonEnd)
	if end == -1 {
		return content[:start]
	}
	after := content[end+len(batonEnd):]
	return strings.TrimLeft(after, "\n")
}

// strategy class
func extractAndSave(chatName, aiCmd, cwd string) {
	var content string

	switch aiCmd {
	case "claude":
		claude := agents.NewClaudeStrategy(claudeDir)
		content = claude.ExtractContext(cwd)
		if id := claude.LatestSessionID(cwd); id != "" {
			saveResumeID(chatName, "claude", id)
		}
	case "gemini":
		gemini := agents.NewGeminiStrategy(geminiDir)
		content = gemini.ExtractContext(cwd)
	case "opencode":
		opencode := agents.NewOpenCodeStrategy(opencodeDir)
		content = opencode.ExtractContext(cwd)
	default:
		generic := agents.NewGenericStrategy(home)
		content = generic.ExtractContext(aiCmd)
	}

	if strings.TrimSpace(content) == "" {
		fmt.Println("⚠️  Baton: No history found to extract.")
		return
	}

	chatFile := filepath.Join(chatsDir, chatName+".md")

	if f, err := os.OpenFile(chatFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		defer f.Close()

		info, _ := f.Stat()
		if info.Size() == 0 {
			f.WriteString(fmt.Sprintf("# Chat: %s\nCreated: %s\n\n", chatName, time.Now().Format("2006-01-02")))
		}

		f.WriteString(fmt.Sprintf("\n--- \n### Session: %s [%s]\n%s", time.Now().Format("15:04:05"), aiCmd, content))
	}

	os.MkdirAll(filepath.Dir(bridgeFile), 0755)
	os.WriteFile(bridgeFile, []byte(content), 0644)

	vaultPath := filepath.Join(vaultDir, "handoff_log.md")
	if vf, err := os.OpenFile(vaultPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		defer vf.Close()
		vf.WriteString(fmt.Sprintf("\n## %s [%s: %s]\n%s\n", time.Now().Format(time.RFC3339), chatName, aiCmd, content))
	}

	fmt.Println("✅ Baton passed! Bridge updated.")
	exec.Command("osascript", "-e", `display notification "Context saved." with title "Baton 🪄"`).Run()
}

// deprecated
func copyToClipboard(content string) {
	cmd := exec.Command("pbcopy")
	in, _ := cmd.StdinPipe()
	go func() {
		defer in.Close()
		io.WriteString(in, content)
	}()
	cmd.Run()
	fmt.Println("📋 Context copied to clipboard.")
}
