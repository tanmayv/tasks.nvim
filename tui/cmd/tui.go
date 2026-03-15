package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/ui"
)

func runTUI(cmd *cobra.Command) {
	dbConn := getDB()
	cfg := getConfig()
	defer dbConn.Close()

	project, _ := cmd.Flags().GetString("project")
	statuses, _ := cmd.Flags().GetStringSlice("status")
	filter, _ := cmd.Flags().GetString("filter")

	model := ui.NewModel(dbConn, cfg.InboxFile, project, statuses, filter)

	p := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(p)
	model.StartDBWatcher(cfg.DBPath)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI program: %v\n", err)
		os.Exit(1)
	}
}
