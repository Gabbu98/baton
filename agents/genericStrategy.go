package agents

import (
	"baton/utils"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type GenericStrategy struct {
	directory string
}

func NewGenericStrategy(dir string) *GenericStrategy {
	return &GenericStrategy{directory: dir}
}

func (g *GenericStrategy) ExtractContext(current_working_directory string) string {
	root := filepath.Join(g.directory, "."+current_working_directory)
	files := utils.JsonFiles(root)
	if len(files) == 0 {
		return ""
	}
	utils.SortByModTime(files)
	return g.parseGenericChatFile(files[0])
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

func (g *GenericStrategy) parseGenericChatFile(path string) string {
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
