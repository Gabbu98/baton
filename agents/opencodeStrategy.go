package agents

import (
	"baton/utils"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OpenCode stores sessions in a SQLite DB at ~/.local/share/opencode/opencode.db.
// Schema:
//
//	message(id, session_id, time_created, time_updated, data JSON{role, time, agent, model})
//	part(id, message_id, session_id, time_created, time_updated, data JSON{type, text})
type OpenCodeStrategy struct {
	directory string
}

func NewOpenCodeStrategy(dir string) *OpenCodeStrategy {
	return &OpenCodeStrategy{directory: dir}
}

func (opencode *OpenCodeStrategy) LatestSessionId(current_working_directory string) string {
	return ""
}

func (opencode *OpenCodeStrategy) ExtractContext(current_working_directory string) string {
	dbPath := opencode.findDB()
	if dbPath == "" {
		return ""
	}
	return opencode.queryMessages(dbPath)
}

func (opencode *OpenCodeStrategy) findDB() string {
	main := filepath.Join(opencode.directory, "opencode.db")
	if _, err := os.Stat(main); err == nil {
		return main
	}

	var dbs []string
	filepath.WalkDir(opencode.directory, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".db") {
			dbs = append(dbs, path)
		}
		return nil
	})
	if len(dbs) == 0 {
		return ""
	}
	utils.SortByModTime(dbs)
	return dbs[0]
}

func (opencode *OpenCodeStrategy) queryMessages(dbPath string) string {
	query := `
		SELECT m.data AS msg_data, p.data AS part_data
		FROM message m
		JOIN part p ON p.message_id = m.id
		ORDER BY p.time_created DESC
		LIMIT 12;
	`
	out, err := exec.Command("sqlite3", "-json", dbPath, query).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return ""
	}

	type row struct {
		MsgData  string `json:"msg_data"`
		PartData string `json:"part_data"`
	}
	var rows []row
	if err := json.Unmarshal(out, &rows); err != nil {
		return ""
	}

	// Reverse topencode chronological order
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	type messageJSON struct {
		Role string `json:"role"`
	}

	type partJSON struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	var sb strings.Builder
	for _, r := range rows {
		var msg messageJSON
		if err := json.Unmarshal([]byte(r.MsgData), &msg); err != nil {
			continue
		}

		var p partJSON
		if err := json.Unmarshal([]byte(r.PartData), &p); err != nil {
			continue
		}

		if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, strings.TrimSpace(p.Text)))
		}
	}
	return sb.String()
}
