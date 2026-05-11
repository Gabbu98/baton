package main

import (
	"baton/agents"
	"baton/utils"
	"errors"
	"fmt"
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
)

func launchAI(args []string) error {
	fmt.Printf("🚀 Baton: Launching %s...\n", args[0])
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

const (
	batonStart = "<!-- baton:context:start -->"
	batonEnd   = "<!-- baton:context:end -->"
)

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

func tryResume(chatName, aiCmd string, extraArgs []string) (bool, error) {
	id := utils.LoadResumeId(chatsDir, chatName, aiCmd)
	if id != "" {
		var resumeArgs []string
		switch aiCmd {
		case "claude", "gemini":
			resumeArgs = []string{aiCmd, "--resume", id}
		case "opencode":
			resumeArgs = []string{"opencode", "--session", id}
		}

		if len(resumeArgs) > 0 {
			fmt.Printf("🔁 Baton: Resuming %s session %s...\n", aiCmd, id[:8])
			if err := launchAI(append(resumeArgs, extraArgs...)); err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					return false, fmt.Errorf("binary not found: %s", aiCmd)
				}
				// Session is stale - clear and signal caller to fallback
				utils.ClearResumeId(chatsDir, chatName, aiCmd)
				fmt.Printf("⚠️  Baton: %s session not found, falling back...\n", aiCmd)
				return false, nil
			}
			return true, nil
		}
	}
	return false, nil
}

func injectMDContext(chatFile, aiCmd, cwd string) {
	if data, err := os.ReadFile(chatFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		mdFile := utils.MdFileFor(aiCmd, cwd)
		if err := writeMDContext(mdFile, string(data), contextPreamble(aiCmd)); err == nil {
			fmt.Printf("📝 Baton: Context injected into %s\n", filepath.Base(mdFile))
		}
	}
}

func runBaton(chatName, aiCmd string, extraArgs []string) {
	cwd, _ := os.Getwd()
	os.MkdirAll(chatsDir, 0755)

	chatFile := filepath.Join(chatsDir, chatName+".md")

	wasResumed, err := tryResume(chatName, aiCmd, extraArgs)
	if !wasResumed || err != nil {
		injectMDContext(chatFile, aiCmd, cwd)
		if err := launchAI(append([]string{aiCmd}, extraArgs...)); err != nil {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				fmt.Printf("❌ %v\n", err)
			}
			return
		}
	}

	extractAndSave(chatName, aiCmd, cwd)
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

func extractAndSave(chatName, aiCmd, cwd string) {
	var content string
	var sessionID string

	switch aiCmd {
	case "claude":
		claude := agents.NewClaudeStrategy(claudeDir)
		content = claude.ExtractContext(cwd)
		sessionID = claude.LatestSessionID(cwd)
	case "gemini":
		gemini := agents.NewGeminiStrategy(geminiDir)
		content = gemini.ExtractContext(cwd)
		sessionID = gemini.LatestSessionID(cwd)
	case "opencode":
		opencode := agents.NewOpenCodeStrategy(opencodeDir)
		content = opencode.ExtractContext(cwd)
		sessionID = opencode.LatestSessionID(cwd)
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

	if sessionID != "" {
		utils.SaveResumeId(chatsDir, chatName, aiCmd, sessionID)
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
