package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync <filepath>",
	Short: "Sync a specific markdown file to the database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()
		cfg := getConfig()

		filePath := args[0]
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			os.Exit(1)
		}

		err = sync.SyncBuffer(absPath, dbConn, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync buffer: %v\n", err)
			os.Exit(1)
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			fmt.Println(`{"success": true}`)
		} else {
			fmt.Printf("File %s synced successfully.\n", absPath)
		}
	},
}

var indexCmd = &cobra.Command{
	Use:   "index <dirpath>",
	Short: "Recursively index all markdown files in a directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()
		cfg := getConfig()

		dirPath := args[0]
		absPath, err := filepath.Abs(dirPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			os.Exit(1)
		}

		err = sync.IndexDirectory(absPath, dbConn, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to index directory: %v\n", err)
			os.Exit(1)
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			fmt.Println(`{"success": true}`)
		} else {
			fmt.Printf("Directory %s indexed successfully.\n", absPath)
		}
	},
}

func init() {
	syncCmd.Flags().Bool("json", false, "Output JSON result")
	indexCmd.Flags().Bool("json", false, "Output JSON result")

	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(indexCmd)
}
