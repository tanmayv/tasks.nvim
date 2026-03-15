package sync

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tanmayv/nvim-task-manager/tui/db"
	"github.com/tanmayv/nvim-task-manager/tui/parser"
)

func AddTaskToInbox(desc, inboxPath string, d *db.DB) error {
	id := parser.GenerateID()
	now := time.Now().UTC()

	task := &Task{
		ID:          id,
		Description: desc,
		Status:      "todo",
		FilePath:    inboxPath,
		LineNumber:  0, // Will update when writing
		CreatedAt:   now.Unix(),
		UpdatedAt:   now.Unix(),
		Tags:        []string{},
		Metadata:    make(map[string]string),
	}

	// Make sure inbox directory exists
	dir := filepath.Dir(inboxPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(inboxPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Parse out metadata from user input (like @project, #tag, due:today)
	parsedTask := parser.ParseLine("- [ ] " + desc)
	if parsedTask != nil {
		// Replace defaults with parsed
		task.Description = parsedTask.Description
		task.Project = parsedTask.Project
		task.Tags = parsedTask.Tags
		task.Priority = parsedTask.Priority
		task.DueDate = parsedTask.DueDate
		task.StartDate = parsedTask.StartDate
		task.Metadata = parsedTask.Metadata
	}

	// Create formatted line to append
	taskLine := fmt.Sprintf("- [ ] %s | id:%s", desc, id)

	if parsedTask != nil {
		parsedTask.ID = id
		taskLine = parser.FormatLine(parsedTask)
	}

	lineCount, _ := countLines(inboxPath)
	task.LineNumber = lineCount + 1

	if _, err := f.WriteString(taskLine + "\n"); err != nil {
		return err
	}

	// Insert to DB
	return d.InsertTask(task)
}

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// ToggleTask Status changes a task in the markdown file and DB
func ToggleTask(taskID, filePath string, currentStatus string, d *db.DB) error {
	newStatus := "done"
	if currentStatus == "done" {
		newStatus = "todo"
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file %s: %w", filePath, err)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()
	if err := scanner.Err(); err != nil {
		return err
	}

	var metadata map[string]string
	lineModified := false

	for i, line := range lines {
		if strings.Contains(line, "id:"+taskID) {
			parsed := parser.ParseLine(line)
			if parsed != nil && parsed.ID == taskID {
				parsed.Status = newStatus

				if newStatus == "done" {
					if parsed.Metadata == nil {
						parsed.Metadata = make(map[string]string)
					}
					parsed.Metadata["done"] = time.Now().UTC().Format("2006-01-02")
				} else {
					delete(parsed.Metadata, "done")
				}

				metadata = parsed.Metadata
				lines[i] = parser.FormatLine(parsed)
				lineModified = true
				break
			}
		}
	}

	if !lineModified {
		return fmt.Errorf("task ID %s not found in file %s", taskID, filePath)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	for _, line := range lines {
		out.WriteString(line + "\n")
	}
	out.Close()

	return d.UpdateTaskStatus(taskID, newStatus, metadata)
}

// IndexDirectory scans a directory recursively for .md files and upserts tasks into the DB
func IndexDirectory(dirPath string, d *db.DB) error {
	// Find all .md files
	var mdFiles []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	// For each file, parse it and sync the tasks
	for _, file := range mdFiles {
		err := SyncBuffer(file, d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error syncing file %s: %v\n", file, err)
		}
	}

	return nil
}

// SyncBuffer reads a file, parses all tasks, and upserts them. It mimics `sync_buffer` from lua.
func SyncBuffer(filePath string, d *db.DB) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	var changes []struct {
		lineNumber int
		text       string
	}
	currentIDs := make(map[string]bool)

	for i, line := range lines {
		task := parser.ParseLine(line)
		if task != nil {
			// Provide ID if it doesn't have one
			if task.ID == "" {
				task.ID = parser.GenerateID()
			}

			// Format line to standardize position
			newLine := parser.FormatLine(task)
			if line != newLine {
				changes = append(changes, struct {
					lineNumber int
					text       string
				}{lineNumber: i + 1, text: newLine})
				lines[i] = newLine
			}

			currentIDs[task.ID] = true

			// Convert to DB task
			now := time.Now().Unix()
			dbTask := &Task{
				ID:          task.ID,
				Description: task.Description,
				Status:      task.Status,
				Project:     task.Project,
				Priority:    task.Priority,
				DueDate:     task.DueDate,
				StartDate:   task.StartDate,
				FilePath:    filePath,
				LineNumber:  i + 1,
				CreatedAt:   now,
				UpdatedAt:   now,
				Tags:        task.Tags,
				Metadata:    task.Metadata,
			}

			err = d.UpsertTask(dbTask)
			if err != nil {
				return fmt.Errorf("failed to upsert task %s: %w", task.ID, err)
			}
		}
	}

	// Apply changes back to buffer if any tasks needed IDs generated or formatted
	if len(changes) > 0 {
		out, err := os.Create(filePath)
		if err != nil {
			return err
		}
		for _, line := range lines {
			out.WriteString(line + "\n")
		}
		out.Close()
	}

	// Clean up missing tasks from this file in the DB
	err = d.DeleteMissingTasksInFile(filePath, currentIDs)
	if err != nil {
		return err
	}

	return nil
}

type Task = db.Task
