package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	claudeDir  = envOr("BATON_CLAUDE_DIR", filepath.Join(home, ".claude"))
	geminiDir  = envOr("BATON_GEMINI_DIR", filepath.Join(home, ".gemini"))
	bridgeFile = envOr("BATON_BRIDGE_FILE", filepath.Join(home, ".config", "baton", "bridge.md"))
	vaultDir   = envOr("BATON_VAULT_DIR", filepath.Join(home, "Documents", "Baton_Vault"))
)

const (
	batonStart = "<!-- baton:context:start -->"
	batonEnd   = "<!-- baton:context:end -->"
)

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
	if len(os.Args) < 2 {
		fmt.Println("🪄 Baton Orchestrator\nUsage: baton [binary-name] [args...]")
		return
	}
	runBaton(os.Args[1], os.Args[2:])
}

func runBaton(aiCmd string, extraArgs []string) {
	cwd, _ := os.Getwd()

	// Inject previous bridge context into the AI's MD file automatically.
	if data, err := os.ReadFile(bridgeFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		mdFile := mdFileFor(aiCmd, cwd)
		if err := writeMDContext(mdFile, string(data)); err == nil {
			fmt.Printf("📝 Baton: Context injected into %s\n", filepath.Base(mdFile))
		}
	}

	fmt.Printf("🚀 Baton: Launching %s...\n", aiCmd)
	args := append([]string{aiCmd}, extraArgs...)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Error: Could not start '%s'. Ensure it is in your PATH.\n", aiCmd)
		return
	}

	extractAndSave(aiCmd, cwd)
}

func mdFileFor(aiCmd, cwd string) string {
	name := strings.ToUpper(aiCmd) + ".md"
	return filepath.Join(cwd, name)
}

// writeMDContext upserts the baton-fenced section at the top of the MD file.
func writeMDContext(mdFile, bridgeContent string) error {
	section := fmt.Sprintf("%s\n## Baton Context Bridge\n_Carried over: %s_\n\n%s\n%s\n",
		batonStart,
		time.Now().Format("2006-01-02 15:04"),
		strings.TrimSpace(bridgeContent),
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

func extractAndSave(aiCmd, cwd string) {
	var content string

	switch aiCmd {
	case "claude":
		content = extractClaudeContext(cwd)
	case "gemini":
		// Gemini CLI does not store parseable chat history; nothing to extract.
		fmt.Println("ℹ️  Baton: Gemini history not extractable; bridge unchanged.")
		return
	default:
		content = extractGenericContext(aiCmd)
	}

	if strings.TrimSpace(content) == "" {
		fmt.Println("⚠️  Baton: No history found to extract.")
		return
	}

	os.MkdirAll(vaultDir, 0755)
	os.MkdirAll(filepath.Dir(bridgeFile), 0755)

	os.WriteFile(bridgeFile, []byte(content), 0644)

	vaultPath := filepath.Join(vaultDir, "handoff_log.md")
	if f, err := os.OpenFile(vaultPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("\n## %s [%s]\n%s\n", time.Now().Format(time.RFC3339), aiCmd, content))
	}

	fmt.Println("✅ Baton passed! Bridge updated.")
	exec.Command("osascript", "-e", `display notification "Context saved." with title "Baton 🪄"`).Run()
}

// extractClaudeContext finds the most recently modified session JSONL for the given cwd.
func extractClaudeContext(cwd string) string {
	projectKey := cwdToProjectKey(cwd)
	projectDir := filepath.Join(claudeDir, "projects", projectKey)

	sessions := jsonlFiles(projectDir)
	if len(sessions) == 0 {
		// Fallback: most recent session across all projects
		sessions = jsonlFiles(filepath.Join(claudeDir, "projects"))
	}
	if len(sessions) == 0 {
		return ""
	}

	sortByModTime(sessions)
	return parseClaudeJSONL(sessions[0])
}

// cwdToProjectKey converts a filesystem path to Claude's project directory naming.
// Claude replaces every non-alphanumeric character (/, .) with "-".
func cwdToProjectKey(cwd string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, cwd)
}

func parseClaudeJSONL(path string) string {
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

		text := claudeContentText(e.Message.Content)
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

// claudeContentText extracts human-readable text from Claude's content field,
// which may be a plain string or an array of typed content blocks.
func claudeContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Plain string content
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	// Array of content blocks — only extract type:"text" blocks
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

// extractGenericContext is a fallback for unknown AI tools.
func extractGenericContext(aiCmd string) string {
	root := filepath.Join(home, "."+aiCmd)
	files := jsonlFiles(root)
	if len(files) == 0 {
		return ""
	}
	sortByModTime(files)
	return parseGenericChatFile(files[0])
}

func jsonlFiles(root string) []string {
	var files []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func sortByModTime(files []string) {
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		return fi.ModTime().After(fj.ModTime())
	})
}

func parseGenericChatFile(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	type entry struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
		Text    string      `json:"text"`
		Parts   []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}

	var result strings.Builder

	if strings.HasSuffix(path, ".jsonl") {
		var lines []string
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		start := len(lines) - 4
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			var e entry
			if json.Unmarshal([]byte(line), &e) == nil {
				if txt := genericText(e.Text, e.Content, e.Parts); txt != "" {
					result.WriteString(fmt.Sprintf("[%s]: %s\n\n", e.Role, txt))
				}
			}
		}
	} else {
		data, _ := io.ReadAll(file)
		var raw map[string]interface{}
		json.Unmarshal(data, &raw)
		msgs, _ := raw["messages"].([]interface{})
		if len(msgs) == 0 {
			msgs, _ = raw["history"].([]interface{})
		}
		start := len(msgs) - 3
		if start < 0 {
			start = 0
		}
		for _, m := range msgs[start:] {
			b, _ := json.Marshal(m)
			var e entry
			json.Unmarshal(b, &e)
			if txt := genericText(e.Text, e.Content, e.Parts); txt != "" {
				result.WriteString(fmt.Sprintf("[%s]: %s\n\n", e.Role, txt))
			}
		}
	}

	return result.String()
}

func genericText(text string, content interface{}, parts []struct {
	Text string `json:"text"`
}) string {
	if text != "" {
		return text
	}
	if s, ok := content.(string); ok && s != "" {
		return s
	}
	if len(parts) > 0 && parts[0].Text != "" {
		return parts[0].Text
	}
	return ""
}

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
