package ui

import (
	"context"
	"crypto/rand"
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
	"github.com/you/lazyadmin/internal/users"
)

type mode int

const (
	modeMain mode = iota
	modeLogs
	modeHelp
	modeUsers
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

type userItem struct {
	user *users.User
}

func (i userItem) Title() string {
	return i.user.ID
}

func (i userItem) Description() string {
	return fmt.Sprintf("SSH: %v | Roles: %v", i.user.SSHUsers, i.user.Roles)
}

func (i userItem) FilterValue() string {
	return i.user.ID
}

type userRegistrationMsg struct {
	userID string
	err    error
}

type userListMsg struct {
	users []*users.User
	err   error
}

type Model struct {
	cfg         *config.Config
	principal   *auth.Principal
	logger      *logging.AuditLogger
	userStore   *users.Store
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

	// User management fields
	userList        []*users.User
	registeringUser bool
	registerStatus  string

	logTable table.Model
	logRows  []table.Row
}

func NewModel(
	cfg *config.Config,
	principal *auth.Principal,
	logger *logging.AuditLogger,
	userStore *users.Store,
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
		userStore:   userStore,
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
	case modeUsers:
		return m.updateUsers(msg)
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
	case modeUsers:
		return m.viewUsers()
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
		case "u":
			if m.principal.IsAdmin() {
				m.mode = modeUsers
				return m.withLoadedUsers(), nil
			}
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
		"[View: %s] [Filter: %s]  [t:toggle view] [a/h/p:filter ops] [enter:run] [l:logs]%s [?:help] [q:quit]",
		viewLabel,
		filterLabel,
		func() string {
			if m.principal.IsAdmin() {
				return " [u:users]"
			}
			return ""
		}(),
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
	help := `
lazyadmin keybindings:

  Main mode:

    ↑/↓ or j/k   Move selection

    enter        Run selected operation

    a            Filter: all operations

    h            Filter: HTTP operations only

    p            Filter: Postgres operations only

    l            View recent audit logs`
	if m.principal.IsAdmin() {
		help += `
    u            Manage users (admin only)`
	}
	help += `
    ?            Show this help

    q / ctrl+c   Quit

  Logs mode:

    q / esc      Return to main

  Users mode (admin only):

    n            Register new user with YubiKey
    q / esc      Return to main

(Press any key to return)

`
	return help
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

// === USERS MODE ===

func (m Model) withLoadedUsers() Model {
	if m.userStore == nil {
		m.userList = []*users.User{}
		return m
	}

	ctx := context.Background()
	userList, err := m.userStore.ListUsers(ctx)
	if err != nil {
		m.registerStatus = fmt.Sprintf("Error loading users: %v", err)
		m.userList = []*users.User{}
		return m
	}

	m.userList = userList
	items := []list.Item{}
	for _, u := range userList {
		items = append(items, userItem{user: u})
	}

	m.list.SetItems(items)
	m.list.Title = "Users (admin only)"
	return m
}

func (m Model) updateUsers(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-7)
	case userListMsg:
		if msg.err != nil {
			m.registerStatus = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.userList = msg.users
			items := []list.Item{}
			for _, u := range msg.users {
				items = append(items, userItem{user: u})
			}
			m.list.SetItems(items)
		}
		return m, nil
	case userRegistrationMsg:
		m.registeringUser = false
		if msg.err != nil {
			m.registerStatus = fmt.Sprintf("Registration failed: %v", msg.err)
		} else {
			m.registerStatus = fmt.Sprintf("User %s registered successfully!", msg.userID)
			// Reload user list
			return m.withLoadedUsers(), nil
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.mode = modeMain
			m.registerStatus = ""
			return m, nil
		case "n":
			if !m.registeringUser {
				m.registeringUser = true
				m.registerStatus = "Starting registration..."
				return m, m.registerNewUser()
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) viewUsers() string {
	s := "User Management (admin only)\n\n"

	if m.registeringUser {
		s += "Registering new user...\n"
		s += "Please touch your YubiKey...\n\n"
	}

	if m.registerStatus != "" {
		s += m.registerStatus + "\n\n"
	}

	s += m.list.View() + "\n"
	s += "[n:register new user] [q/esc:return to main]\n"

	return s
}

func (m Model) registerNewUser() tea.Cmd {
	return func() tea.Msg {
		if m.userStore == nil {
			return userRegistrationMsg{err: fmt.Errorf("user store not available")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// For now, we'll use a simple registration flow
		// In a full implementation, you'd prompt for user ID, SSH users, and roles
		// For this implementation, we'll use defaults and register the YubiKey

		// Generate a temporary user ID (in real implementation, prompt for this)
		userIDBytes := make([]byte, 8)
		if _, err := rand.Read(userIDBytes); err != nil {
			return userRegistrationMsg{err: fmt.Errorf("generate user ID: %w", err)}
		}

		// Use default RP ID from config
		rpID := "lazyadmin.local"
		if m.cfg.Auth.YubiKeyMode != "" {
			// Could be configured per environment
		}

		// Register the credential
		result, err := auth.RegisterFIDO2Credential(ctx, rpID, "lazyadmin", "newuser", userIDBytes)
		if err != nil {
			return userRegistrationMsg{err: fmt.Errorf("register credential: %w", err)}
		}

		// Create user with default values (in production, prompt for these)
		// For now, use a placeholder user ID
		newUserID := fmt.Sprintf("user_%d", time.Now().Unix())
		newUser := &users.User{
			ID:       newUserID,
			SSHUsers: []string{newUserID}, // In production, prompt for SSH username
			Roles:    []string{"read_only"}, // Default role, admin can change later
		}

		// Create user in database
		if err := m.userStore.CreateUser(ctx, newUser); err != nil {
			return userRegistrationMsg{err: fmt.Errorf("create user: %w", err)}
		}

		// Add credential
		cred := &users.Credential{
			RPID:        rpID,
			CredentialID: result.CredentialID,
			PublicKey:   result.PublicKey,
		}

		if err := m.userStore.AddCredential(ctx, newUserID, cred); err != nil {
			// Try to clean up user if credential add fails
			_ = m.userStore.DeleteUser(ctx, newUserID)
			return userRegistrationMsg{err: fmt.Errorf("add credential: %w", err)}
		}

		return userRegistrationMsg{userID: newUserID}
	}
}
