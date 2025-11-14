package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/you/lazyadmin/internal/auth"
	"github.com/you/lazyadmin/internal/clients"
	"github.com/you/lazyadmin/internal/config"
	"github.com/you/lazyadmin/internal/logging"
	"github.com/you/lazyadmin/internal/tasks"
)

type mode int

const (
	modeMain mode = iota
	modeLogs
	modeHelp
)

type filterType int

const (
	filterAll filterType = iota
	filterHTTP
	filterPostgres
)

type operationItem struct {
	op config.Operation
}

func (i operationItem) Title() string       { return i.op.Label }
func (i operationItem) Description() string { return fmt.Sprintf("%s (%s)", i.op.ID, i.op.Type) }
func (i operationItem) FilterValue() string { return i.op.Label }

type operationResultMsg struct {
	op     config.Operation
	output string
	errMsg string
}

type taskResultMsg struct {
	task   config.Task
	result *tasks.TaskResult
	summary string
}

type taskItem struct {
	task config.Task
}

func (i taskItem) Title() string       { return i.task.Label }
func (i taskItem) Description() string { return fmt.Sprintf("task:%s (risk:%s)", i.task.ID, i.task.RiskLevel) }
func (i taskItem) FilterValue() string { return i.task.Label }

type Model struct {
	cfg         *config.Config
	principal   *auth.Principal
	logger      *logging.AuditLogger
	httpClients map[string]*clients.HTTPClient
	pgClients   map[string]*clients.PostgresClient
	taskRunner  *tasks.Runner

	mode      mode
	filter    filterType
	viewTasks bool
	list      list.Model
	lastOp    *config.Operation
	lastOutput string
	lastError  string

	// Task fields
	lastTask       *config.Task
	lastTaskResult *tasks.TaskResult
	lastSummary    string

	logTable table.Model
	logRows  []table.Row
}

func NewModel(
	cfg *config.Config,
	principal *auth.Principal,
	logger *logging.AuditLogger,
	httpClients map[string]*clients.HTTPClient,
	pgClients map[string]*clients.PostgresClient,
	runner *tasks.Runner,
) Model {
	items := operationsToItems(cfg, principal, filterAll)

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = fmt.Sprintf(
		"lazyadmin – project=%s env=%s – user=%s roles=%v",
		cfg.Project,
		cfg.Env,
		principal.SSHUser,
		principal.ConfigUser.Roles,
	)

	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	// Log table (empty initially; populated on entering logs mode)
	columns := []table.Column{
		{Title: "Time", Width: 24},
		{Title: "User", Width: 10},
		{Title: "Op", Width: 20},
		{Title: "OK", Width: 3},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
	)

	return Model{
		cfg:         cfg,
		principal:   principal,
		logger:      logger,
		httpClients: ensureHTTPMap(httpClients),
		pgClients:   pgClients,
		taskRunner:  runner,
		mode:        modeMain,
		filter:      filterAll,
		viewTasks:   false,
		list:        l,
		logTable:    t,
	}
}

func tasksToItems(cfg *config.Config, principal *auth.Principal) []list.Item {
	items := []list.Item{}
	for _, t := range cfg.Tasks {
		if !principal.HasAnyRole(t.AllowedRoles) {
			continue
		}
		items = append(items, taskItem{task: t})
	}
	return items
}

func ensureHTTPMap(m map[string]*clients.HTTPClient) map[string]*clients.HTTPClient {
	if m == nil {
		return map[string]*clients.HTTPClient{}
	}
	return m
}

func operationsToItems(cfg *config.Config, principal *auth.Principal, f filterType) []list.Item {
	items := []list.Item{}
	for _, op := range cfg.Operations {
		if !principal.HasAnyRole(op.AllowedRoles) {
			continue
		}

		if f == filterHTTP && op.Type != "http" {
			continue
		}

		if f == filterPostgres && op.Type != "postgres" {
			continue
		}

		items = append(items, operationItem{op: op})
	}
	return items
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeMain:
		return m.updateMain(msg)
	case modeLogs:
		return m.updateLogs(msg)
	case modeHelp:
		return m.updateHelp(msg)
	default:
		return m, nil
	}
}

func (m Model) View() string {
	switch m.mode {
	case modeMain:
		return m.viewMain()
	case modeLogs:
		return m.viewLogs()
	case modeHelp:
		return m.viewHelp()
	default:
		return "unknown mode"
	}
}

// === MAIN MODE ===

func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-7) // space for status + details
	case operationResultMsg:
		m.lastOp = &msg.op
		m.lastOutput = msg.output
		m.lastError = msg.errMsg
		return m, nil
	case taskResultMsg:
		m.lastTask = &msg.task
		m.lastTaskResult = msg.result
		m.lastSummary = msg.summary
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.viewTasks {
				if it, ok := m.list.SelectedItem().(taskItem); ok {
					return m, m.runTask(it.task)
				}
			} else {
				if it, ok := m.list.SelectedItem().(operationItem); ok {
					return m, m.runOperation(it.op)
				}
			}
		case "t":
			m.viewTasks = !m.viewTasks
			if m.viewTasks {
				m.list.SetItems(tasksToItems(m.cfg, m.principal))
			} else {
				m.list.SetItems(operationsToItems(m.cfg, m.principal, m.filter))
			}
		case "a":
			if !m.viewTasks {
				m.filter = filterAll
				m.list.SetItems(operationsToItems(m.cfg, m.principal, m.filter))
			}
		case "h":
			if !m.viewTasks {
				m.filter = filterHTTP
				m.list.SetItems(operationsToItems(m.cfg, m.principal, m.filter))
			}
		case "p":
			if !m.viewTasks {
				m.filter = filterPostgres
				m.list.SetItems(operationsToItems(m.cfg, m.principal, m.filter))
			}
		case "l":
			m.mode = modeLogs
			return m.withLoadedLogs(), nil
		case "?":
			m.mode = modeHelp
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) viewMain() string {
	viewLabel := "Operations"
	if m.viewTasks {
		viewLabel = "Tasks"
	}

	filterLabel := "N/A"
	if !m.viewTasks {
		filterLabel = "All"
		switch m.filter {
		case filterHTTP:
			filterLabel = "HTTP"
		case filterPostgres:
			filterLabel = "Postgres"
		}
	}

	status := fmt.Sprintf(
		"[View: %s] [Filter: %s]  [t:toggle view] [a/h/p:filter ops] [enter:run] [l:logs] [?:help] [q:quit]",
		viewLabel,
		filterLabel,
	)

	s := m.list.View() + "\n"
	s += status + "\n"

	s += "\nDetails:\n"
	if m.viewTasks {
		if m.lastTask != nil {
			s += fmt.Sprintf("  Last task: %s (risk:%s)\n", m.lastTask.ID, m.lastTask.RiskLevel)
			if m.lastTaskResult != nil {
				s += fmt.Sprintf("  Success: %v\n", m.lastTaskResult.Success)
			}
			if m.lastSummary != "" {
				s += "  Summary:\n"
				for _, line := range splitLines(m.lastSummary) {
					s += "    " + line + "\n"
				}
			}
		} else {
			s += "  No tasks run yet.\n"
		}
	} else {
		if m.lastOp != nil {
			s += fmt.Sprintf("  Last op: %s (%s)\n", m.lastOp.ID, m.lastOp.Type)
			if m.lastError != "" {
				s += fmt.Sprintf("  Error: %s\n", m.lastError)
			} else if m.lastOutput != "" {
				s += fmt.Sprintf("  Output: %s\n", m.lastOutput)
			}
		} else {
			s += "  No operations run yet.\n"
		}
	}

	return s
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// === LOGS MODE ===

func (m Model) withLoadedLogs() Model {
	rows, err := logging.ReadRecent(m.logger, 50)
	if err != nil {
		m.lastError = fmt.Sprintf("read logs: %v", err)
		rows = nil
	}

	tRows := []table.Row{}
	for _, r := range rows {
		ok := "✗"
		if r.Success {
			ok = "✓"
		}

		tRows = append(tRows, table.Row{
			r.OccurredAt.Format("2006-01-02 15:04:05"),
			r.UserID,
			r.OperationID,
			ok,
		})
	}

	m.logRows = tRows
	m.logTable.SetRows(tRows)
	return m
}

func (m Model) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.logTable.SetWidth(msg.Width)
		m.logTable.SetHeight(msg.Height - 4)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.mode = modeMain
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.logTable, cmd = m.logTable.Update(msg)
	return m, cmd
}

func (m Model) viewLogs() string {
	s := "Recent audit log entries (q/esc to return):\n\n"
	s += m.logTable.View()
	return s
}

// === HELP MODE ===

func (m Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		default:
			m.mode = modeMain
			return m, nil
		}
	}

	return m, nil
}

func (m Model) viewHelp() string {
	return `
lazyadmin keybindings:

  Main mode:

    ↑/↓ or j/k   Move selection

    enter        Run selected operation

    a            Filter: all operations

    h            Filter: HTTP operations only

    p            Filter: Postgres operations only

    l            View recent audit logs

    ?            Show this help

    q / ctrl+c   Quit

  Logs mode:

    q / esc      Return to main

(Press any key to return)

`
}

func (m Model) runOperation(op config.Operation) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var out string
		var err error

		switch op.Type {
		case "http":
			client, ok := m.httpClients[op.Target]
			if !ok {
				err = fmt.Errorf("no http resource named %q", op.Target)
			} else {
				out, err = client.Request(ctx, op.Method, op.Path)
			}
		case "postgres":
			client, ok := m.pgClients[op.Target]
			if !ok {
				err = fmt.Errorf("no postgres resource named %q", op.Target)
			} else {
				out, err = client.RunScalarQuery(ctx, op.Query)
			}
		default:
			err = fmt.Errorf("unsupported op type: %s", op.Type)
		}

		entry := logging.AuditEntry{
			Time:        time.Now(),
			UserID:      m.principal.ConfigUser.ID,
			SSHUser:     m.principal.SSHUser,
			OperationID: op.ID,
			Success:     err == nil,
		}
		if err != nil {
			entry.Error = err.Error()
		}

		_ = m.logger.Log(ctx, entry)

		if err != nil {
			return operationResultMsg{op: op, errMsg: err.Error()}
		}
		return operationResultMsg{op: op, output: out}
	}
}

func (m Model) runTask(task config.Task) tea.Cmd {
	return func() tea.Msg {
		if m.taskRunner == nil {
			return taskResultMsg{
				task:    task,
				result:  nil,
				summary: "task runner not configured",
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		tr := m.taskRunner.Run(ctx, m.principal.ConfigUser.ID, m.principal.SSHUser, task)

		summary, err := tasks.RenderSummary(task, tr)
		if err != nil {
			summary = fmt.Sprintf("error rendering summary: %v", err)
		}

		return taskResultMsg{
			task:    task,
			result:  &tr,
			summary: summary,
		}
	}
}
