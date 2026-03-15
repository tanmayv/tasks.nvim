package sync

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tanmayv/nvim-task-manager/tui/db"
	"github.com/tanmayv/nvim-task-manager/tui/parser"
)

// DeleteTask removes a task from the markdown file and the database
func DeleteTask(taskID, filePath string, d *db.DB) error {
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

	lineModified := false
	var finalLines []string

	// Find and remove the line with the task ID
	for _, line := range lines {
		if strings.Contains(line, "id:"+taskID) {
			parsed := parser.ParseLine(line)
			if parsed != nil && parsed.ID == taskID {
				lineModified = true
				continue // Skip appending this line (deleting it)
			}
		}
		finalLines = append(finalLines, line)
	}

	if !lineModified {
		return fmt.Errorf("task ID %s not found in file %s", taskID, filePath)
	}

	// Overwrite file with the line removed
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	// Note: joining with newline, dropping the last empty string if it exists
	outputStr := strings.Join(finalLines, "\n")
	if len(finalLines) > 0 {
		out.WriteString(outputStr + "\n")
	}
	out.Close()

	// Delete from DB
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM tasks WHERE id = ?`, taskID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
