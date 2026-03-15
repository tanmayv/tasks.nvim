package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tanmayv/nvim-task-manager/tui/db"
	"github.com/tanmayv/nvim-task-manager/tui/sync"
)

// UI styling
var (
	docStyle       = lipgloss.NewStyle().Margin(1, 2)
	statusStyleMap = map[string]lipgloss.Style{
		"todo":        lipgloss.NewStyle().Foreground(lipgloss.Color("#2E8B57")), // SeaGreen
		"done":        lipgloss.NewStyle().Foreground(lipgloss.Color("#7F7F7F")).Strikethrough(true),
		"in_progress": lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")), // Orange
		"cancelled":   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")), // Red
	}
	projectStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6495ED"))            // CornflowerBlue
	tagStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4"))            // HotPink
	urgentScoreStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true) // OrangeRed
	highScoreStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#DAA520"))            // Goldenrod
	normalScoreStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9"))            // DarkGray
)

// list item adapter
type item struct {
	task *db.Task
}

func (i item) Title() string       { return i.task.Description }
func (i item) Description() string { return "" }
func (i item) FilterValue() string {
	filterStr := i.task.Description
	if i.task.Project != "" {
		filterStr += " @" + i.task.Project
	}
	for _, tag := range i.task.Tags {
		filterStr += " #" + tag
	}
	return filterStr
}

// Custom 2-line delegate
type taskDelegate struct{}

func (d taskDelegate) Height() int                             { return 2 }
func (d taskDelegate) Spacing() int                            { return 1 }
func (d taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d taskDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	task := i.task

	// Format checkbox and title (Line 1)
	statusStr := "[ ]"
	if task.Status == "done" {
		statusStr = "[x]"
	} else if task.Status == "in_progress" {
		statusStr = "[/]"
	} else if task.Status == "cancelled" {
		statusStr = "[-]"
	}

	titleStyle, ok := statusStyleMap[task.Status]
	if !ok {
		titleStyle = statusStyleMap["todo"]
	}

	// Highlight selected item
	baseStyle := lipgloss.NewStyle()
	if index == m.Index() {
		baseStyle = baseStyle.Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#4169E1")) // RoyalBlue bg
		fmt.Fprintf(w, "> %s %s\n", statusStr, task.Description)
	} else {
		fmt.Fprintf(w, "  %s %s\n", titleStyle.Render(statusStr), titleStyle.Render(task.Description))
	}

	// Format metadata (Line 2)
	scoreStyle := normalScoreStyle
	if task.Score >= 200 {
		scoreStyle = urgentScoreStyle
	} else if task.Score >= 50 {
		scoreStyle = highScoreStyle
	}
	scoreStr := scoreStyle.Render(fmt.Sprintf("[%d]", task.Score))

	metaParts := []string{scoreStr}

	if task.Project != "" {
		metaParts = append(metaParts, projectStyle.Render("@"+task.Project))
	}

	var tagStrs []string
	for _, tag := range task.Tags {
		tagStrs = append(tagStrs, tagStyle.Render("#"+tag))
	}
	if len(tagStrs) > 0 {
		metaParts = append(metaParts, strings.Join(tagStrs, " "))
	}

	if task.DueDate != "" {
		dueStr := "due:" + task.DueDate
		if task.Score >= 200 { // Overdue logic triggers urgent score
			metaParts = append(metaParts, urgentScoreStyle.Render(dueStr))
		} else {
			metaParts = append(metaParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEFA")).Render(dueStr))
		}
	}

	metaLine := strings.Join(metaParts, " ")

	if index == m.Index() {
		fmt.Fprintf(w, "    %s", metaLine)
	} else {
		fmt.Fprintf(w, "    %s", metaLine)
	}
}

// App model
type Model struct {
	list        list.Model
	dbConn      *db.DB
	inboxPath   string
	loaded      bool
	err         error
	isInputView bool
	inputModel  InputModel
	program     *tea.Program
}

func NewModel(dbConn *db.DB, inboxPath string) *Model {
	d := taskDelegate{}
	l := list.New([]list.Item{}, d, 0, 0)
	l.Title = "Open Tasks"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Background(lipgloss.Color("#4169E1")).Foreground(lipgloss.Color("#FFF")).Padding(0, 1)

	return &Model{
		list:        l,
		dbConn:      dbConn,
		inboxPath:   inboxPath,
		isInputView: false,
		inputModel:  NewInputModel(),
	}
}

func (m *Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.dbConn.GetTasks(db.GetTasksOpts{
			Statuses: []string{"todo", "in_progress"},
		})
		if err != nil {
			return err
		}

		items := make([]list.Item, len(tasks))
		for i, t := range tasks {
			items[i] = item{task: t}
		}

		return items
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

func (m *Model) Init() tea.Cmd {
	return m.loadTasks()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isInputView {
			// Handle Input View
			switch msg.String() {
			case "esc":
				m.isInputView = false
				m.inputModel.textInput.SetValue("")
				return m, nil
			case "enter":
				val := m.inputModel.textInput.Value()
				if strings.TrimSpace(val) != "" {
					err := sync.AddTaskToInbox(val, m.inboxPath, m.dbConn)
					if err != nil {
						m.err = err
					}
				}
				m.isInputView = false
				m.inputModel.textInput.SetValue("")
				return m, m.loadTasks() // reload
			}

			// Pass to input model
			var cmd tea.Cmd
			m.inputModel.textInput, cmd = m.inputModel.textInput.Update(msg)
			return m, cmd
		}

		// Handle List View
		switch msg.String() {
		case "ctrl+c", "q":
			// only quit if we aren't actively typing in the filter
			if !m.list.SettingFilter() {
				return m, tea.Quit
			}
		case " ":
			// Toggle task status
			if m.list.FilterState() == list.Filtering {
				break
			}

			if selected, ok := m.list.SelectedItem().(item); ok {
				err := sync.ToggleTask(selected.task.ID, selected.task.FilePath, selected.task.Status, m.dbConn)
				if err != nil {
					m.err = err
					return m, nil
				}
				// Reload after toggle
				return m, m.loadTasks()
			}
		case "a":
			if m.list.FilterState() == list.Filtering {
				break
			}
			m.isInputView = true
			m.inputModel.textInput.Focus()
			return m, nil
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case ReloadMsg:
		return m, m.loadTasks()
	case []list.Item:
		m.list.SetItems(msg)
		m.loaded = true

	case error:
		m.err = msg
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress q to quit.", m.err)
	}

	if !m.loaded {
		return "\nLoading tasks...\n"
	}

	if m.isInputView {
		return docStyle.Render(m.inputModel.View())
	}

	return docStyle.Render(m.list.View())
}
