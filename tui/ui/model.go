package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tanmayv/nvim-task-manager/tui/config"
	"github.com/tanmayv/nvim-task-manager/tui/db"
	"github.com/tanmayv/nvim-task-manager/tui/parser"
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
	reindex   key.Binding
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
		reindex: key.NewBinding(
			key.WithKeys("I"), // Capital I
			key.WithHelp("I", "reindex"),
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

	if task.NoteTitle != "" {
		metaParts = append(metaParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#98FB98")).Render("· "+task.NoteTitle))
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
		l.SetFilterText(filterText)
	}

	// Inject keys into list's help menu
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.toggle, keys.add, keys.delete, keys.edit, keys.openNotes, keys.reindex}
	}
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.toggle, keys.add, keys.delete, keys.edit, keys.openNotes, keys.reindex}
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

		completedOpts := db.GetTasksOpts{
			Project: m.filterProject,
		}
		completedTasks, err := m.dbConn.GetCompletedTodayTasks(completedOpts)
		if err == nil {
			tasks = append(tasks, completedTasks...)
		}

		cfg, _ := config.LoadConfig()
		var zkDir string
		if cfg != nil && len(cfg.Directories) > 0 {
			zkDir = cfg.Directories[0]
		}

		items := make([]list.Item, len(tasks))
		for i, t := range tasks {
			if t.AttachedNote != "" && zkDir != "" {
				t.NoteTitle = fetchNoteTitle(t.AttachedNote, zkDir)
			}
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

						// Extract active projects and tags from the active fuzzy search filter
						var filterProjectFromSearch string
						var tagsFromSearch []string
						filterVal := m.list.FilterValue()
						if strings.TrimSpace(filterVal) != "" {
							for _, token := range strings.Fields(filterVal) {
								if strings.HasPrefix(token, "@") {
									filterProjectFromSearch = strings.TrimPrefix(token, "@")
								} else if strings.HasPrefix(token, "#") {
									tagsFromSearch = append(tagsFromSearch, token)
								}
							}
						}

						for _, taskDesc := range m.inputModel.PendingTasks {
							finalDesc := taskDesc
							
							// Token-aware project and tag extraction check on the quick-add text
							hasProject := false
							existingTags := make(map[string]bool)
							for _, word := range strings.Fields(taskDesc) {
								if strings.HasPrefix(word, "@") {
									hasProject = true
								} else if strings.HasPrefix(word, "#") {
									existingTags[word] = true
								}
							}

							// Project inheritance
							if !hasProject {
								if m.filterProject != "" {
									finalDesc = finalDesc + " @" + m.filterProject
								} else if filterProjectFromSearch != "" {
									finalDesc = finalDesc + " @" + filterProjectFromSearch
								}
							}

							// Tag inheritance
							for _, tag := range tagsFromSearch {
								if !existingTags[tag] {
									finalDesc = finalDesc + " " + tag
								}
							}

							err := sync.AddTaskToInbox(finalDesc, m.inboxPath, m.dbConn, cfg)
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
				cfg, _ := config.LoadConfig()
				var zkDir string
				if cfg != nil && len(cfg.Directories) > 0 {
					zkDir = cfg.Directories[0]
				}

				// Part 3: Overloaded Note Navigation
				if selected.task.AttachedNote != "" {
					noteTarget := selected.task.AttachedNote
					if !strings.HasPrefix(noteTarget, "/") && zkDir != "" {
						resolved := resolveNotePath(noteTarget, zkDir)
						if resolved != "" {
							noteTarget = resolved
						}
					}

					cmd := exec.Command("nvim", noteTarget)
					return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
						if err != nil {
							m.err = err
						}
						return ReloadMsg{}
					})
				}

				// Fall back standardly to scanning the task description for zk links
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
					if zkDir != "" {
						zkCmd.Dir = zkDir
					}
					out, err := zkCmd.Output()
					
					if err != nil {
						m.err = err
						return m, nil
					}

					if len(out) == 0 {
						m.err = fmt.Errorf("no notes resolved for links: %v", links)
						return m, nil
					}

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
					} else {
						m.err = fmt.Errorf("no notes resolved for links: %v", links)
					}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+t"))):
			if m.list.FilterState() == list.Filtering {
				break
			}
			if selected, ok := m.list.SelectedItem().(item); ok {
				// Overload check: If note is already attached, directly open it!
				if selected.task.AttachedNote != "" {
					cfg, _ := config.LoadConfig()
					var zkDir string
					if cfg != nil && len(cfg.Directories) > 0 {
						zkDir = cfg.Directories[0]
					}

					noteTarget := selected.task.AttachedNote
					if !strings.HasPrefix(noteTarget, "/") && zkDir != "" {
						resolved := resolveNotePath(noteTarget, zkDir)
						if resolved != "" {
							noteTarget = resolved
						}
					}

					cmd := exec.Command("nvim", noteTarget)
					return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
						if err != nil {
							m.err = err
						}
						return ReloadMsg{}
					})
				}

				// Part 2: TUI Inline Note Creation Hotkey <C-t>
				out, err := exec.Command("nn", "--print-path").Output()
				if err != nil {
					m.err = fmt.Errorf("failed to execute nn: %w", err)
					return m, nil
				}
				notePath := strings.TrimSpace(string(out))
				if notePath == "" {
					m.err = fmt.Errorf("no path returned by nn")
					return m, nil
				}

				cmd := exec.Command("nvim", notePath)
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
					if err != nil {
						m.err = err
						return ReloadMsg{}
					}

					// Extract Note ID
					noteFile := strings.Split(notePath, "/")
					noteID := strings.TrimSuffix(noteFile[len(noteFile)-1], ".md")

					// 1. Update SQLite database attached_note field
					_, err = m.dbConn.DB.Exec(`UPDATE tasks SET attached_note = ?, updated_at = ? WHERE id = ?`, noteID, time.Now().Unix(), selected.task.ID)
					if err != nil {
						m.err = err
						return ReloadMsg{}
					}

					// 2. Read original markdown file lines
					filePath := selected.task.FilePath
					file, err := os.Open(filePath)
					if err != nil {
						m.err = err
						return ReloadMsg{}
					}
					var lines []string
					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						lines = append(lines, scanner.Text())
					}
					file.Close()

					// 3. Update the inline task note metadata note:ID
					lineModified := false
					for i, line := range lines {
						if strings.Contains(line, "id:"+selected.task.ID) {
							parsed := parser.ParseLine(line)
							if parsed != nil && parsed.ID == selected.task.ID {
								if parsed.Metadata == nil {
									parsed.Metadata = make(map[string]string)
								}
								parsed.Metadata["note"] = noteID
								lines[i] = parser.FormatLine(parsed)
								lineModified = true
								break
							}
						}
					}

					if lineModified {
						outF, err := os.Create(filePath)
						if err != nil {
							m.err = fmt.Errorf("failed to re-write task source file %s: %w", filePath, err)
							return ReloadMsg{}
						}
						for _, line := range lines {
							outF.WriteString(line + "\n")
						}
						outF.Close()
					} else {
						// If no line was modified (ID not found in file)
						m.err = fmt.Errorf("task ID %s not found inside source file %s (failed to append inline note)", selected.task.ID, filePath)
						return ReloadMsg{}
					}

					// 4. Sync buffer to reconcile systems
					cfg, _ := config.LoadConfig()
					_ = sync.SyncBuffer(filePath, m.dbConn, cfg)

					return ReloadMsg{}
				})
			}
		case key.Matches(msg, m.keys.reindex):
			if m.list.FilterState() == list.Filtering {
				break
			}
			return m, m.reindexTasks()
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case ReloadMsg:
		return m, m.loadTasks()
	case []list.Item:
		cmd := m.list.SetItems(msg)
		m.loaded = true
		return m, cmd

	case error:
		m.err = msg
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) reindexTasks() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		if err := m.dbConn.ClearTasks(); err != nil {
			return err
		}

		for _, dir := range cfg.Directories {
			_ = sync.IndexDirectory(dir, m.dbConn, cfg)
		}

		return ReloadMsg{}
	}
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

func fetchNoteTitle(attachedNote string, zkDir string) string {
	if attachedNote == "" || zkDir == "" {
		return ""
	}

	// Resolve note path standardly (first check dots/ directory)
	notePath := filepath.Join(zkDir, "dots", attachedNote+".md")
	file, err := os.Open(notePath)
	if err != nil {
		// Fallback to direct path inside notebook
		notePath = filepath.Join(zkDir, attachedNote+".md")
		file, err = os.Open(notePath)
		if err != nil {
			return attachedNote
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	frontmatterDelims := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			frontmatterDelims++
			if frontmatterDelims == 1 {
				inFrontmatter = true
				continue
			} else if frontmatterDelims == 2 {
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			continue
		}

		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}

	return attachedNote
}

func resolveNotePath(attachedNote string, zkDir string) string {
	if attachedNote == "" || zkDir == "" {
		return ""
	}

	noteFile := attachedNote
	if !strings.HasSuffix(noteFile, ".md") {
		noteFile += ".md"
	}

	// Fast path 1: dots/ directory
	p1 := filepath.Join(zkDir, "dots", noteFile)
	if _, err := os.Stat(p1); err == nil {
		return p1
	}

	// Fast path 2: notebook root folder
	p2 := filepath.Join(zkDir, noteFile)
	if _, err := os.Stat(p2); err == nil {
		return p2
	}

	// Fallback: recursive Walk to find match
	var foundPath string
	_ = filepath.Walk(zkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Base(path) == noteFile {
			foundPath = path
			return fmt.Errorf("abort_walk")
		}
		return nil
	})

	if foundPath != "" {
		return foundPath
	}

	return ""
}
