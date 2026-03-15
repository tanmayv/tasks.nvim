package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*DB, string) {
	tmpDir, err := os.MkdirTemp("", "taskmanager-test")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
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
	if err != nil {
		t.Fatal(err)
	}

	return &DB{DB: db}, dbPath
}

func TestGetTasks(t *testing.T) {
	d, dbPath := setupTestDB(t)
	defer os.RemoveAll(filepath.Dir(dbPath))
	defer d.Close()

	now := time.Now().Unix()

	err := d.InsertTask(&Task{
		ID:          "t:1",
		Description: "Learn Go",
		Status:      "todo",
		Project:     "work",
		Tags:        []string{"urgent"},
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = d.InsertTask(&Task{
		ID:          "t:2",
		Description: "Buy milk",
		Status:      "done",
		Project:     "home",
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}

	opts := GetTasksOpts{
		Statuses: []string{"todo"},
		Project:  "work",
	}

	tasks, err := d.GetTasks(opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	if tasks[0].ID != "t:1" {
		t.Errorf("Expected task t:1, got %s", tasks[0].ID)
	}
	if len(tasks[0].Tags) != 1 || tasks[0].Tags[0] != "urgent" {
		t.Errorf("Expected 1 tag 'urgent', got %v", tasks[0].Tags)
	}
}

func TestCalculateScore(t *testing.T) {
	d := &DB{}
	today := time.Now().UTC().Format("2006-01-02")

	// Urgent tag: +100
	t1 := &Task{Tags: []string{"urgent"}}
	d.calculateScore(t1)
	if t1.Score != 100 {
		t.Errorf("Expected urgent score 100, got %d", t1.Score)
	}

	// Due today: +150
	t2 := &Task{DueDate: today}
	d.calculateScore(t2)
	if t2.Score != 150 {
		t.Errorf("Expected due today score 150, got %d", t2.Score)
	}
}
