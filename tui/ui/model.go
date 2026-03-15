package ui

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tanmayv/nvim-task-manager/tui/config"
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

// Key bindings
type keyMap struct {
	toggle    key.Binding
	add       key.Binding
	delete    key.Binding
	edit      key.Binding
	openNotes key.Binding
}

func newKeyMap() *keyMap {
	return &keyMap{
		toggle: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "mark done"),
		),
		add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		edit: key.NewBinding(
			key.WithKeys("e", "enter"),
			key.WithHelp("e/enter", "edit task"),
		),
		openNotes: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "open notes"),
		),
	}
}

// Custom 2-line delegate
type taskDelegate struct {
	keys *keyMap
}

func (d taskDelegate) Height() int                               { return 2 }
func (d taskDelegate) Spacing() int                              { return 1 }
func (d taskDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
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
	list          list.Model
	dbConn        *db.DB
	inboxPath     string
	loaded        bool
	err           error
	keys          *keyMap
	isInputView   bool
	inputModel    InputModel
	program       *tea.Program
	filterProject string
	filterStatus  []string
}

func NewModel(dbConn *db.DB, inboxPath string, project string, statuses []string, filterText string) *Model {
	keys := newKeyMap()
	d := taskDelegate{keys: keys}
	l := list.New([]list.Item{}, d, 0, 0)
	l.Title = "Open Tasks"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Background(lipgloss.Color("#4169E1")).Foreground(lipgloss.Color("#FFF")).Padding(0, 1)

	if filterText != "" {
		l.FilterInput.SetValue(filterText)
	}

	// Inject keys into list's help menu
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.toggle, keys.add, keys.delete, keys.edit, keys.openNotes}
	}
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.toggle, keys.add, keys.delete, keys.edit, keys.openNotes}
	}

	return &Model{
		list:          l,
		dbConn:        dbConn,
		inboxPath:     inboxPath,
		isInputView:   false,
		inputModel:    NewInputModel(dbConn),
		keys:          keys,
		filterProject: project,
		filterStatus:  statuses,
	}
}

func (m *Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		opts := db.GetTasksOpts{
			Statuses: m.filterStatus,
			Project:  m.filterProject,
		}
		if len(opts.Statuses) == 0 {
			opts.Statuses = []string{"todo", "in_progress"} // Default if somehow emptied
		}

		tasks, err := m.dbConn.GetTasks(opts)
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
			// Handle confirmation first before we pass keystrokes to the text input
			if m.inputModel.Confirming {
				switch msg.String() {
				case "y", "Y":
					m.isInputView = false
					m.inputModel.PendingTasks = nil
					m.inputModel.Confirming = false
					m.inputModel.textInput.SetValue("")
					return m, nil
				case "n", "N", "esc":
					m.inputModel.Confirming = false
					return m, nil
				case "ctrl+c":
					return m, tea.Quit
				}
				return m, nil // swallow everything else
			}

			// Record if we were completing before we pass the message down
			wasCompleting := m.inputModel.isCompleting && len(m.inputModel.suggestions) > 0

			// Handle Input View specific escapes BEFORE text input update
			switch msg.String() {
			case "esc", "ctrl+c":
				val := m.inputModel.textInput.Value()
				if len(m.inputModel.PendingTasks) > 0 || strings.TrimSpace(val) != "" {
					m.inputModel.Confirming = true
					return m, nil
				}
				if msg.String() == "ctrl+c" {
					return m, tea.Quit
				}
				m.isInputView = false
				m.inputModel.textInput.SetValue("")
				return m, nil
			case "q":
				val := m.inputModel.textInput.Value()
				if len(m.inputModel.PendingTasks) > 0 && val == "" {
					m.inputModel.Confirming = true
					return m, nil
				}
				// if val is not empty, let textinput handle typing 'q'
			case "enter":
				// If we are autocompleting, don't submit task, let InputModel handle it
				if wasCompleting {
					break
				}
				
				val := m.inputModel.textInput.Value()
				if strings.TrimSpace(val) != "" {
					m.inputModel.PendingTasks = append(m.inputModel.PendingTasks, strings.TrimSpace(val))
					m.inputModel.textInput.SetValue("")
					return m, nil
				} else {
					// Submitting all tasks
					if len(m.inputModel.PendingTasks) > 0 {
						cfg, _ := config.LoadConfig()
						for _, taskDesc := range m.inputModel.PendingTasks {
							err := sync.AddTaskToInbox(taskDesc, m.inboxPath, m.dbConn, cfg)
							if err != nil {
								m.err = err
							}
						}
					}
					m.isInputView = false
					m.inputModel.PendingTasks = nil
					m.inputModel.textInput.SetValue("")
					return m, m.loadTasks() // reload
				}
			}

			// Pass to input model
			var cmd tea.Cmd
			m.inputModel, cmd = m.inputModel.Update(msg)
			return m, cmd
		}

		// Handle List View
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q"))):
			// only quit if we aren't actively typing in the filter
			if !m.list.SettingFilter() {
				return m, tea.Quit
			}
		case key.Matches(msg, m.keys.toggle):
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
				return m, m.loadTasks()
			}
		case key.Matches(msg, m.keys.add):
			if m.list.FilterState() == list.Filtering {
				break
			}
			m.isInputView = true
			m.inputModel.textInput.Focus()
			return m, nil
		case key.Matches(msg, m.keys.delete):
			if m.list.FilterState() == list.Filtering {
				break
			}
			if selected, ok := m.list.SelectedItem().(item); ok {
				err := sync.DeleteTask(selected.task.ID, selected.task.FilePath, m.dbConn)
				if err != nil {
					m.err = err
					return m, nil
				}
				// Reload after toggle
				return m, m.loadTasks()
			}
		case key.Matches(msg, m.keys.edit):
			if m.list.FilterState() == list.Filtering {
				break
			}
			if selected, ok := m.list.SelectedItem().(item); ok {
				cmd := exec.Command("nvim", fmt.Sprintf("+%d", selected.task.LineNumber), selected.task.FilePath)
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
					if err != nil {
						m.err = err
					}
					return ReloadMsg{}
				})
			}
		case key.Matches(msg, m.keys.openNotes):
			if m.list.FilterState() == list.Filtering {
				break
			}
			if selected, ok := m.list.SelectedItem().(item); ok {
				// Parse zk links from task description: [[...]]
				var links []string
				desc := selected.task.Description
				for {
					start := strings.Index(desc, "[[")
					if start == -1 {
						break
					}
					end := strings.Index(desc[start:], "]]")
					if end == -1 {
						break
					}
					link := desc[start+2 : start+end]
					if link != "" {
						links = append(links, link)
					}
					desc = desc[start+end+2:]
				}

				if len(links) > 0 {
					// Ask zk to resolve the links into absolute paths
					args := []string{"list", "--quiet", "--format", "{{absPath}}"}
					args = append(args, links...)
					zkCmd := exec.Command("zk", args...)
					out, err := zkCmd.Output()
					
					if err == nil && len(out) > 0 {
						absPaths := strings.Split(strings.TrimSpace(string(out)), "\n")
						var validPaths []string
						for _, p := range absPaths {
							if !strings.HasPrefix(p, "zk: warning:") && p != "" {
								validPaths = append(validPaths, p)
							}
						}

						if len(validPaths) > 0 {
							nvimArgs := append([]string{"-O"}, validPaths...) // -O opens in vertical splits
							cmd := exec.Command("nvim", nvimArgs...)
							return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
								if err != nil {
									m.err = err
								}
								return ReloadMsg{}
							})
						}
					}
				}
			}
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
