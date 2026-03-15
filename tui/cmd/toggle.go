package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
)

var toggleCmd = &cobra.Command{
	Use:   "toggle <task_id> <filepath>",
	Short: "Toggle task completion status",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()

		taskID := args[0]
		filePath := args[1]

		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			os.Exit(1)
		}

		// First we need the current status to toggle it.
		// Since sync.ToggleTask does a simple flip if it's 'done',
		// we lookup the task in DB to find its current status.
		var currentStatus string
		err = dbConn.DB.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to find task %s in database: %v\n", taskID, err)
			os.Exit(1)
		}

		err = sync.ToggleTask(taskID, absPath, currentStatus, dbConn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to toggle task: %v\n", err)
			os.Exit(1)
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			fmt.Println(`{"success": true}`)
		} else {
			fmt.Printf("Task %s toggled successfully.\n", taskID)
		}
	},
}

func init() {
	toggleCmd.Flags().Bool("json", false, "Output JSON result")
	rootCmd.AddCommand(toggleCmd)
}
