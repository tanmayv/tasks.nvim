package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tanmayv/nvim-task-manager/tui/db"
)

func TestToggleTask(t *testing.T) {
	// Setup temp file and DB
	tmpDir, err := os.MkdirTemp("", "taskmanager-sync-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	d, err := db.Connect(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	// Initialize tables (normally done by nvim plugin or migration)
	d.DB.Exec(`
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			description TEXT,
			status TEXT,
			project TEXT,
			priority TEXT,
			due_date TEXT,
			start_date TEXT,
			file_path TEXT,
			line_number INTEGER,
			created_at INTEGER,
			updated_at INTEGER
		);
		CREATE TABLE task_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
			key TEXT,
			value TEXT
		);
		CREATE TABLE task_tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
			tag_name TEXT
		);
	`)

	inboxPath := filepath.Join(tmpDir, "inbox.md")
	os.WriteFile(inboxPath, []byte("- [ ] Fix bug | @work id:t:123\n"), 0644)

	task := &db.Task{
		ID:          "t:123",
		Description: "Fix bug",
		Status:      "todo",
		FilePath:    inboxPath,
		LineNumber:  1,
	}
	d.InsertTask(task)

	// Test toggling to done
	err = ToggleTask("t:123", inboxPath, "todo", d)
	if err != nil {
		t.Fatalf("Failed to toggle task: %v", err)
	}

	content, _ := os.ReadFile(inboxPath)
	contentStr := string(content)

	if !strings.Contains(contentStr, "[x]") {
		t.Errorf("File did not update to done state, got: %s", contentStr)
	}

	today := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(contentStr, "done:"+today) {
		t.Errorf("File did not append done:date metadata, got: %s", contentStr)
	}

	// Verify DB is updated
	tasks, _ := d.GetTasks(db.GetTasksOpts{Statuses: []string{"done"}})
	if len(tasks) != 1 {
		t.Errorf("DB did not reflect toggled task correctly")
	}
}
