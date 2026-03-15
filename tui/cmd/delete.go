package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <task_id> <filepath>",
	Short: "Delete a task from markdown and database",
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

		err = sync.DeleteTask(taskID, absPath, dbConn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete task: %v\n", err)
			os.Exit(1)
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			fmt.Println(`{"success": true}`)
		} else {
			fmt.Printf("Task %s deleted successfully.\n", taskID)
		}
	},
}

func init() {
	deleteCmd.Flags().Bool("json", false, "Output JSON result")
	rootCmd.AddCommand(deleteCmd)
}
