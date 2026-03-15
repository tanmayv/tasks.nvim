package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tanmayv/nvim-task-manager/tui/ui"
)

func runTUI() {
	dbConn := getDB()
	cfg := getConfig()
	defer dbConn.Close()

	model := ui.NewModel(dbConn, cfg.InboxFile)

	p := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(p)
	model.StartDBWatcher(cfg.DBPath)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI program: %v\n", err)
		os.Exit(1)
	}
}
