package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type InputModel struct {
	textInput textinput.Model
}

func NewInputModel() InputModel {
	ti := textinput.New()
	ti.Placeholder = "Add a new task (e.g. 'Fix bug | @work #urgent due:today')"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	return InputModel{
		textInput: ti,
	}
}

func (m InputModel) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#4169E1")).Bold(true).Render("New Task"),
		"",
		m.textInput.View(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("(esc to cancel, enter to save)"),
	)
}
