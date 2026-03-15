package db

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

type Task struct {
	ID          string
	Description string
	Status      string
	Project     string
	Priority    string
	DueDate     string
	StartDate   string
	FilePath    string
	LineNumber  int
	CreatedAt   int64
	UpdatedAt   int64
	Tags        []string
	Metadata    map[string]string
	Score       int
}

type DB struct {
	DB *sql.DB
}

func Connect(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	return &DB{DB: db}, nil
}

func (d *DB) Close() error {
	return d.DB.Close()
}

type GetTasksOpts struct {
	Statuses []string
	Project  string
}

func (d *DB) GetTasks(opts GetTasksOpts) ([]*Task, error) {
	query := `SELECT id, description, status, project, priority, due_date, start_date, file_path, line_number, created_at, updated_at FROM tasks WHERE 1=1`
	var args []interface{}
	argCount := 1

	if len(opts.Statuses) > 0 {
		query += " AND status IN ("
		for i, status := range opts.Statuses {
			query += fmt.Sprintf("$%d", argCount)
			args = append(args, status)
			argCount++
			if i < len(opts.Statuses)-1 {
				query += ", "
			}
		}
		query += ")"
	}

	if opts.Project != "" {
		query += fmt.Sprintf(" AND project = $%d", argCount)
		args = append(args, opts.Project)
		argCount++
	}

	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{
			Tags:     []string{},
			Metadata: make(map[string]string),
		}
		var proj, prio, due, start sql.NullString
		if err := rows.Scan(
			&task.ID, &task.Description, &task.Status, &proj, &prio, &due, &start,
			&task.FilePath, &task.LineNumber, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, err
		}
		task.Project = proj.String
		task.Priority = prio.String
		task.DueDate = due.String
		task.StartDate = start.String

		tasks = append(tasks, task)
	}

	// Fetch tags and metadata for each task
	for _, task := range tasks {
		tagRows, err := d.DB.Query(`SELECT tag_name FROM task_tags WHERE task_id = ?`, task.ID)
		if err == nil {
			for tagRows.Next() {
				var tag string
				if tagRows.Scan(&tag) == nil {
					task.Tags = append(task.Tags, tag)
				}
			}
			tagRows.Close()
		}

		metaRows, err := d.DB.Query(`SELECT key, value FROM task_metadata WHERE task_id = ?`, task.ID)
		if err == nil {
			for metaRows.Next() {
				var key, value string
				if metaRows.Scan(&key, &value) == nil {
					task.Metadata[key] = value
				}
			}
			metaRows.Close()
		}

		d.calculateScore(task)
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Score != tasks[j].Score {
			return tasks[i].Score > tasks[j].Score
		}
		return tasks[i].Description < tasks[j].Description
	})

	return tasks, nil
}

func (d *DB) calculateScore(task *Task) {
	score := 0
	now := time.Now().UTC()
	todayStr := now.Format("2006-01-02")

	if task.CreatedAt > 0 {
		createdTime := time.Unix(task.CreatedAt, 0).UTC()
		ageDays := int(now.Sub(createdTime).Hours() / 24)
		if ageDays > 0 {
			score += ageDays
		}
	}

	if task.Priority == "high" {
		score += 50
	} else if task.Priority == "medium" {
		score += 20
	}

	for _, tag := range task.Tags {
		if tag == "urgent" {
			score += 100
		} else {
			score += 5
		}
	}

	if task.StartDate != "" && task.StartDate > todayStr {
		score -= 1000
	}

	if task.DueDate != "" {
		if task.DueDate < todayStr {
			dueTime, err := time.Parse("2006-01-02", task.DueDate)
			if err == nil {
				daysOverdue := int(now.Sub(dueTime).Hours() / 24)
				score += 200 + (daysOverdue * 10)
			} else {
				score += 200
			}
		} else if task.DueDate == todayStr {
			score += 150
		} else {
			dueTime, err := time.Parse("2006-01-02", task.DueDate)
			if err == nil {
				daysUntil := int(dueTime.Sub(now).Hours() / 24)
				if daysUntil <= 14 {
					futureScore := 100 - (daysUntil * 7)
					score += int(math.Max(0, float64(futureScore)))
				}
			}
		}
	}

	task.Score = score
}

func (d *DB) UpdateTaskStatus(taskID, status string, metadata map[string]string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().Unix(), taskID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM task_metadata WHERE task_id = ?`, taskID)
	if err != nil {
		return err
	}

	for k, v := range metadata {
		_, err = tx.Exec(`INSERT INTO task_metadata (task_id, key, value) VALUES (?, ?, ?)`, taskID, k, v)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) InsertTask(task *Task) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()

	_, err = tx.Exec(`
		INSERT INTO tasks (id, description, status, project, priority, due_date, start_date, file_path, line_number, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.Description, task.Status, task.Project, task.Priority, task.DueDate, task.StartDate, task.FilePath, task.LineNumber, now, now)

	if err != nil {
		return err
	}

	for _, tag := range task.Tags {
		_, err = tx.Exec(`INSERT INTO task_tags (task_id, tag_name) VALUES (?, ?)`, task.ID, tag)
		if err != nil {
			return err
		}
	}

	for k, v := range task.Metadata {
		_, err = tx.Exec(`INSERT INTO task_metadata (task_id, key, value) VALUES (?, ?, ?)`, task.ID, k, v)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) UpsertTask(task *Task) error {
	var count int
	err := d.DB.QueryRow(`SELECT count(*) FROM tasks WHERE id = ?`, task.ID).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Update
		tx, err := d.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
			UPDATE tasks 
			SET description = ?, status = ?, project = ?, priority = ?, due_date = ?, start_date = ?, file_path = ?, line_number = ?, updated_at = ?
			WHERE id = ?
		`, task.Description, task.Status, task.Project, task.Priority, task.DueDate, task.StartDate, task.FilePath, task.LineNumber, time.Now().Unix(), task.ID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`DELETE FROM task_metadata WHERE task_id = ?`, task.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`DELETE FROM task_tags WHERE task_id = ?`, task.ID)
		if err != nil {
			return err
		}

		for _, tag := range task.Tags {
			_, err = tx.Exec(`INSERT INTO task_tags (task_id, tag_name) VALUES (?, ?)`, task.ID, tag)
			if err != nil {
				return err
			}
		}

		for k, v := range task.Metadata {
			_, err = tx.Exec(`INSERT INTO task_metadata (task_id, key, value) VALUES (?, ?, ?)`, task.ID, k, v)
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	} else {
		return d.InsertTask(task)
	}
}

func (d *DB) DeleteMissingTasksInFile(filePath string, currentIDs map[string]bool) error {
	rows, err := d.DB.Query(`SELECT id FROM tasks WHERE file_path = ?`, filePath)
	if err != nil {
		return err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			if !currentIDs[id] {
				toDelete = append(toDelete, id)
			}
		}
	}
	rows.Close()

	if len(toDelete) > 0 {
		tx, err := d.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, id := range toDelete {
			_, err = tx.Exec(`DELETE FROM tasks WHERE id = ?`, id)
			if err != nil {
				return err
			}
		}
		return tx.Commit()
	}

	return nil
}
