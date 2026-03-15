package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
)

var addCmd = &cobra.Command{
	Use:   "add <description>",
	Short: "Add a new task to the inbox",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()
		cfg := getConfig()

		desc := strings.Join(args, " ")
		err := sync.AddTaskToInbox(desc, cfg.InboxFile, dbConn, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to add task: %v\n", err)
			os.Exit(1)
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			fmt.Println(`{"success": true}`)
		} else {
			fmt.Println("Task added successfully.")
		}
	},
}

func init() {
	addCmd.Flags().Bool("json", false, "Output JSON result")
	rootCmd.AddCommand(addCmd)
}
