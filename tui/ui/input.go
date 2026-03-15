package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tanmayv/nvim-task-manager/tui/db"
)

type InputModel struct {
	textInput        textinput.Model
	dbConn           *db.DB
	projects         []string
	tags             []string
	suggestions      []string
	suggestionIndex  int
	activeSuggestion string
	isCompleting     bool
}

func NewInputModel(dbConn *db.DB) InputModel {
	ti := textinput.New()
	ti.Placeholder = "Add a new task (e.g. 'Fix bug | @work #urgent due:today')"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	projects := []string{}
	tags := []string{}

	if dbConn != nil {
		p, _ := dbConn.GetProjects()
		if p != nil {
			projects = p
		}
		t, _ := dbConn.GetTags()
		if t != nil {
			tags = t
		}
	}

	return InputModel{
		textInput:       ti,
		dbConn:          dbConn,
		projects:        projects,
		tags:            tags,
		suggestionIndex: 0,
	}
}

func (m *InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.isCompleting && len(m.suggestions) > 0 {
				m.suggestionIndex = (m.suggestionIndex + 1) % len(m.suggestions)
				m.activeSuggestion = m.suggestions[m.suggestionIndex]
				return *m, nil
			}
		case "shift+tab":
			if m.isCompleting && len(m.suggestions) > 0 {
				m.suggestionIndex--
				if m.suggestionIndex < 0 {
					m.suggestionIndex = len(m.suggestions) - 1
				}
				m.activeSuggestion = m.suggestions[m.suggestionIndex]
				return *m, nil
			}
		case "enter":
			if m.isCompleting && len(m.suggestions) > 0 {
				// Insert suggestion
				val := m.textInput.Value()
				cursor := m.textInput.Position()
				
				// Find start of current word
				start := cursor
				for start > 0 && val[start-1] != ' ' {
					start--
				}
				
				prefix := val[:start]
				suffix := val[cursor:]
				
				insert := ""
				currentWord := val[start:cursor]
				if strings.HasPrefix(currentWord, "@") {
					insert = "@" + m.activeSuggestion + " "
				} else if strings.HasPrefix(currentWord, "#") {
					insert = "#" + m.activeSuggestion + " "
				}
				
				m.textInput.SetValue(prefix + insert + suffix)
				m.textInput.SetCursor(len(prefix) + len(insert))
				
				m.isCompleting = false
				m.suggestions = nil
				return *m, nil
			}
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	m.updateSuggestions()

	return *m, cmd
}

func (m *InputModel) updateSuggestions() {
	val := m.textInput.Value()
	cursor := m.textInput.Position()

	if cursor == 0 {
		m.isCompleting = false
		m.suggestions = nil
		return
	}

	// Find current word
	start := cursor
	for start > 0 && val[start-1] != ' ' {
		start--
	}
	
	currentWord := val[start:cursor]
	
	m.suggestions = nil
	m.isCompleting = false
	
	if strings.HasPrefix(currentWord, "@") && len(currentWord) > 0 {
		m.isCompleting = true
		prefix := strings.ToLower(currentWord[1:])
		for _, p := range m.projects {
			if strings.HasPrefix(strings.ToLower(p), prefix) {
				m.suggestions = append(m.suggestions, p)
			}
		}
	} else if strings.HasPrefix(currentWord, "#") && len(currentWord) > 0 {
		m.isCompleting = true
		prefix := strings.ToLower(currentWord[1:])
		for _, t := range m.tags {
			if strings.HasPrefix(strings.ToLower(t), prefix) {
				m.suggestions = append(m.suggestions, t)
			}
		}
	}

	if len(m.suggestions) > 0 {
		if m.suggestionIndex >= len(m.suggestions) || m.suggestionIndex < 0 {
			m.suggestionIndex = 0
		}
		m.activeSuggestion = m.suggestions[m.suggestionIndex]
	} else {
		m.suggestionIndex = 0
		m.activeSuggestion = ""
	}
}

func (m InputModel) View() string {
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#4169E1")).Bold(true).Render("New Task"),
		"",
		m.textInput.View(),
	)

	if m.isCompleting && len(m.suggestions) > 0 {
		var suggestionViews []string
		for i, s := range m.suggestions {
			style := lipgloss.NewStyle().Padding(0, 1)
			if i == m.suggestionIndex {
				style = style.Background(lipgloss.Color("#4169E1")).Foreground(lipgloss.Color("#FFFFFF"))
			} else {
				style = style.Foreground(lipgloss.Color("#A9A9A9"))
			}
			suggestionViews = append(suggestionViews, style.Render(s))
		}
		
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			"",
			lipgloss.JoinHorizontal(lipgloss.Top, suggestionViews...),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("(tab to cycle, enter to select)"),
		)
	} else {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			"",
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("(esc to cancel, enter to save)"),
		)
	}

	return view
}
