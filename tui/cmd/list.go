package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/db"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()

		status, _ := cmd.Flags().GetStringSlice("status")
		project, _ := cmd.Flags().GetString("project")
		asJSON, _ := cmd.Flags().GetBool("json")

		opts := db.GetTasksOpts{}
		if len(status) > 0 {
			opts.Statuses = status
		}
		if project != "" {
			opts.Project = project
		}

		tasks, err := dbConn.GetTasks(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching tasks: %v\n", err)
			os.Exit(1)
		}

		if asJSON {
			output, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(output))
		} else {
			for _, t := range tasks {
				fmt.Printf("[%s] %s (Score: %d)\n", t.Status, t.Description, t.Score)
			}
		}
	},
}

func init() {
	listCmd.Flags().StringSliceP("status", "s", []string{}, "Filter by status (e.g. todo, in_progress, done, cancelled)")
	listCmd.Flags().StringP("project", "p", "", "Filter by project")
	listCmd.Flags().Bool("json", false, "Output as JSON")

	rootCmd.AddCommand(listCmd)
}
