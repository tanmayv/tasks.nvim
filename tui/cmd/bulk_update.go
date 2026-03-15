package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/parser"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
)

type BulkOrigin struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	InitialLine string `json:"initial_line"`
}

var bulkUpdateCmd = &cobra.Command{
	Use:   "bulk-update",
	Short: "Apply a bulk update from a Neovim scratch buffer",
	Run: func(cmd *cobra.Command, args []string) {
		editedFile, _ := cmd.Flags().GetString("edited-file")
		originsFile, _ := cmd.Flags().GetString("origins")

		if editedFile == "" || originsFile == "" {
			fmt.Fprintln(os.Stderr, "Both --edited-file and --origins are required")
			os.Exit(1)
		}

		// 1. Load Origins
		originsData, err := os.ReadFile(originsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read origins: %v\n", err)
			os.Exit(1)
		}

		var originsMap map[string]BulkOrigin
		if err := json.Unmarshal(originsData, &originsMap); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse origins JSON: %v\n", err)
			os.Exit(1)
		}

		// 2. Load Edited File
		editedData, err := os.ReadFile(editedFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read edited file: %v\n", err)
			os.Exit(1)
		}

		dbConn := getDB()
		defer dbConn.Close()
		cfg := getConfig()

		// 3. Parse Current State
		lines := strings.Split(string(editedData), "\n")
		currentTasks := make(map[string]struct {
			Line string
			Task *parser.Task
		})

		for _, line := range lines {
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
				continue
			}
			task := parser.ParseLine(line)
			if task != nil {
				if task.ID != "" {
					currentTasks[task.ID] = struct {
						Line string
						Task *parser.Task
					}{line, task}
				} else {
					// It's a brand new task added in the scratch buffer
					parts := parser.FormatDescription(task)
					desc := strings.Join(parts, " ")
					sync.AddTaskToInbox(desc, cfg.InboxFile, dbConn)
				}
			}
		}

		// 4. Determine Changes per file
		type FileChange struct {
			Deletes []BulkOrigin
			Updates []struct {
				Origin  BulkOrigin
				NewTask *parser.Task
			}
		}
		fileChanges := make(map[string]*FileChange)

		for origID, origin := range originsMap {
			if _, exists := fileChanges[origin.FilePath]; !exists {
				fileChanges[origin.FilePath] = &FileChange{}
			}

			current, exists := currentTasks[origID]
			if !exists {
				// Task was deleted from editor buffer
				fileChanges[origin.FilePath].Deletes = append(fileChanges[origin.FilePath].Deletes, origin)
			} else if origin.InitialLine != current.Line {
				// Task was modified
				fileChanges[origin.FilePath].Updates = append(fileChanges[origin.FilePath].Updates, struct {
					Origin  BulkOrigin
					NewTask *parser.Task
				}{origin, current.Task})
			}
		}

		// 5. Apply Changes to files
		for filePath, changes := range fileChanges {
			if len(changes.Deletes) == 0 && len(changes.Updates) == 0 {
				continue
			}

			targetData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Skipping %s: %v\n", filePath, err)
				continue
			}
			targetLines := strings.Split(string(targetData), "\n")

			// Process updates
			for _, update := range changes.Updates {
				for i, tLine := range targetLines {
					tTask := parser.ParseLine(tLine)
					if tTask != nil && tTask.ID == update.Origin.ID {
						update.NewTask.Prefix = tTask.Prefix // Preserve original indentation
						targetLines[i] = parser.FormatLine(update.NewTask)
						break
					}
				}
			}

			// Process deletes
			var finalLines []string
			for _, tLine := range targetLines {
				keep := true
				tTask := parser.ParseLine(tLine)
				if tTask != nil {
					for _, del := range changes.Deletes {
						if tTask.ID == del.ID {
							keep = false
							break
						}
					}
				}
				if keep {
					finalLines = append(finalLines, tLine)
				}
			}

			// Save file
			// Note: dropping the last empty string from split if it exists
			outputStr := strings.Join(finalLines, "\n")
			err = os.WriteFile(filePath, []byte(outputStr), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", filePath, err)
				continue
			}

			// Resync the file with the DB
			sync.SyncBuffer(filePath, dbConn)
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			fmt.Println(`{"success": true}`)
		} else {
			fmt.Println("Bulk update applied successfully.")
		}
	},
}

func init() {
	bulkUpdateCmd.Flags().String("edited-file", "", "Path to the edited scratch buffer file")
	bulkUpdateCmd.Flags().String("origins", "", "Path to the JSON mapping of origins")
	bulkUpdateCmd.Flags().Bool("json", false, "Output JSON result")

	rootCmd.AddCommand(bulkUpdateCmd)
}
