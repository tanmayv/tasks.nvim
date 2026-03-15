package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tanmayv/nvim-task-manager/tui/config"
	"github.com/tanmayv/nvim-task-manager/tui/db"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
	"github.com/tanmayv/nvim-task-manager/tui/ui"
)

func main() {
	indexDir := flag.String("index", "", "Directory to scan for markdown files and index tasks into SQLite")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	dbConn, err := db.Connect(cfg.DBPath)
	if err != nil {
		fmt.Printf("Error connecting to database at %s: %v\n", cfg.DBPath, err)
		os.Exit(1)
	}
	defer dbConn.Close()

	if *indexDir != "" {
		absPath, err := filepath.Abs(*indexDir)
		if err != nil {
			fmt.Printf("Error resolving path: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Indexing directory: %s\n", absPath)
		err = sync.IndexDirectory(absPath, dbConn)
		if err != nil {
			fmt.Printf("Failed to index directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Indexing complete.")
		return
	}

	model := ui.NewModel(dbConn, cfg.InboxFile)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
