package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tanmayv/nvim-task-manager/tui/db"
)

type ZkNote struct {
	Title        string
	FilenameStem string
	Display      string
}

type Suggestion struct {
	Display string
	Insert  string
}

type InputModel struct {
	textInput        textinput.Model
	dbConn           *db.DB
	projects         []string
	tags             []string
	zkNotes          []ZkNote
	suggestions      []Suggestion
	suggestionIndex  int
	activeSuggestion Suggestion
	isCompleting     bool
	PendingTasks     []string
	Confirming       bool
}

func fetchZkNotes() []ZkNote {
	cmd := exec.Command("zk", "list", "--quiet", "--format", "{{title}}\t{{filenameStem}}")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var notes []ZkNote
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "zk: warning:") || line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) == 2 {
			title := strings.TrimSpace(parts[0])
			stem := strings.TrimSpace(parts[1])
			display := title
			if display == "" {
				display = stem
			}
			notes = append(notes, ZkNote{Title: title, FilenameStem: stem, Display: display})
		}
	}
	return notes
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

	zkNotes := fetchZkNotes()

	return InputModel{
		textInput:       ti,
		dbConn:          dbConn,
		projects:        projects,
		tags:            tags,
		zkNotes:         zkNotes,
		suggestionIndex: 0,
	}
}

func (m *InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "ctrl+n":
			if m.isCompleting && len(m.suggestions) > 0 {
				m.suggestionIndex = (m.suggestionIndex + 1) % len(m.suggestions)
				m.activeSuggestion = m.suggestions[m.suggestionIndex]
				return *m, nil
			}
		case "shift+tab", "ctrl+p":
			if m.isCompleting && len(m.suggestions) > 0 {
				m.suggestionIndex--
				if m.suggestionIndex < 0 {
					m.suggestionIndex = len(m.suggestions) - 1
				}
				m.activeSuggestion = m.suggestions[m.suggestionIndex]
				return *m, nil
			}
		case "enter", "ctrl+y":
			if m.isCompleting && len(m.suggestions) > 0 {
				val := m.textInput.Value()
				cursor := m.textInput.Position()
				
				// Find start of current word
				start := cursor
				for start > 0 && val[start-1] != ' ' {
					start--
				}
				
				prefix := val[:start]
				suffix := val[cursor:]
				
				insert := m.activeSuggestion.Insert + " "
				
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
				m.suggestions = append(m.suggestions, Suggestion{Display: "@" + p, Insert: "@" + p})
			}
		}
	} else if strings.HasPrefix(currentWord, "#") && len(currentWord) > 0 {
		m.isCompleting = true
		prefix := strings.ToLower(currentWord[1:])
		for _, t := range m.tags {
			if strings.HasPrefix(strings.ToLower(t), prefix) {
				m.suggestions = append(m.suggestions, Suggestion{Display: "#" + t, Insert: "#" + t})
			}
		}
	} else if strings.HasPrefix(currentWord, "[[") && len(currentWord) > 1 {
		m.isCompleting = true
		prefix := strings.ToLower(currentWord[2:])
		for _, n := range m.zkNotes {
			if strings.HasPrefix(strings.ToLower(n.Display), prefix) || strings.HasPrefix(strings.ToLower(n.FilenameStem), prefix) {
				m.suggestions = append(m.suggestions, Suggestion{Display: n.Display, Insert: "[[" + n.FilenameStem + "]]"})
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
		m.activeSuggestion = Suggestion{}
	}
}

func (m InputModel) View() string {
	if m.Confirming {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("Discard pending tasks? (y/n)"),
		)
	}

	var pendingViews []string
	if len(m.PendingTasks) > 0 {
		pendingViews = append(pendingViews, lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9")).Render("Pending tasks:"))
		for i, t := range m.PendingTasks {
			displayTask := t
			if len(displayTask) > 55 {
				displayTask = displayTask[:52] + "..."
			}
			pendingViews = append(pendingViews, lipgloss.NewStyle().Foreground(lipgloss.Color("#6495ED")).Render(fmt.Sprintf(" %d. %s", i+1, displayTask)))
		}
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#4169E1")).Bold(true).Render("New Task"),
		"",
		m.textInput.View(),
	)

	if len(pendingViews) > 0 {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			"",
			strings.Join(pendingViews, "\n"),
		)
	}

	if m.isCompleting && len(m.suggestions) > 0 {
		var suggestionViews []string
		for i, s := range m.suggestions {
			style := lipgloss.NewStyle().Padding(0, 1)
			if i == m.suggestionIndex {
				style = style.Background(lipgloss.Color("#4169E1")).Foreground(lipgloss.Color("#FFFFFF"))
			} else {
				style = style.Foreground(lipgloss.Color("#A9A9A9"))
			}
			suggestionViews = append(suggestionViews, style.Render(s.Display))
		}
		
		displayCount := 5
		if len(suggestionViews) > displayCount {
			suggestionViews = suggestionViews[:displayCount]
			suggestionViews = append(suggestionViews, lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("..."))
		}

		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			"",
			lipgloss.JoinHorizontal(lipgloss.Top, suggestionViews...),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("(tab/c-n/c-p to cycle, enter/c-y to select)"),
		)
	} else {
		helpText := "(enter to queue, enter on empty to save, esc/q to cancel)"
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			"",
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(helpText),
		)
	}

	return view
}