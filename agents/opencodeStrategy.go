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

// OpenCode stores sessions in a SQLite DB at ~/.local/share/opencode/db.
// Schema: message(id, session_id, role, parts JSON, time INTEGER)
// parts: [{"type":"text","text":"..."}]
type OpenCodeStrategy struct {
	directory string
}

func NewOpenCodeStrategy(dir string) *OpenCodeStrategy {
	return &OpenCodeStrategy{directory: dir}
}

func (o *OpenCodeStrategy) ExtractContext(cwd string) string {
	dbPath := o.findDB()
	if dbPath == "" {
		return ""
	}
	return o.queryMessages(dbPath)
}

func (o *OpenCodeStrategy) findDB() string {
	main := filepath.Join(o.directory, "db")
	if _, err := os.Stat(main); err == nil {
		return main
	}

	var dbs []string
	filepath.WalkDir(o.directory, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Base(path) == "db" {
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

func (o *OpenCodeStrategy) queryMessages(dbPath string) string {
	// Fetch last 12 rows in reverse-chron, then we reverse for display.
	query := `SELECT role, parts FROM message ORDER BY time DESC LIMIT 12;`
	out, err := exec.Command("sqlite3", "-json", dbPath, query).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return ""
	}

	type row struct {
		Role  string `json:"role"`
		Parts string `json:"parts"`
	}
	var rows []row
	if err := json.Unmarshal(out, &rows); err != nil {
		return ""
	}

	// Reverse to chronological order
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	type part struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	var sb strings.Builder
	for _, r := range rows {
		var parts []part
		if err := json.Unmarshal([]byte(r.Parts), &parts); err != nil {
			continue
		}
		for _, p := range parts {
			if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
				sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", r.Role, strings.TrimSpace(p.Text)))
				break
			}
		}
	}
	return sb.String()
}
