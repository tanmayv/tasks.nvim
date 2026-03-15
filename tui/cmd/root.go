package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/config"
	"github.com/tanmayv/nvim-task-manager/tui/db"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
)

var rootCmd = &cobra.Command{
	Use:   "task",
	Short: "A CLI/TUI task manager companion to nvim-task-manager",
	Run: func(cmd *cobra.Command, args []string) {
		runTUI(cmd)
	},
}

func init() {
	rootCmd.Flags().StringP("filter", "f", "", "Pre-fill the TUI fuzzy search filter")
	rootCmd.Flags().StringP("project", "p", "", "Filter tasks by project")
	rootCmd.Flags().StringSliceP("status", "s", []string{"todo", "in_progress"}, "Filter tasks by status (todo, in_progress, done, cancelled)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getDB() *db.DB {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	needsIndexing := false
	if _, err := os.Stat(cfg.DBPath); os.IsNotExist(err) {
		needsIndexing = true
	}

	dbConn, err := db.Connect(cfg.DBPath)
	if err != nil {
		fmt.Printf("Error connecting to database at %s: %v\n", cfg.DBPath, err)
		os.Exit(1)
	}

	if needsIndexing {
		for _, dir := range cfg.Directories {
			err := sync.IndexDirectory(dir, dbConn, cfg)
			if err != nil {
				fmt.Printf("Warning: failed to index directory %s: %v\n", dir, err)
			}
		}
	}

	return dbConn
}

func getConfig() *config.Config {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}
