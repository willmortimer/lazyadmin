# Architecture

This document describes the internal structure and data flow of lazyadmin.

## Module Overview

### `cmd/lazyadmin`

Application entry point. Responsibilities:

- Load and validate configuration
- Resolve principal from environment
- Initialize clients, logger, and task runner
- Start TUI

### `internal/config`

Configuration loading and type definitions. Responsibilities:

- Parse YAML configuration file
- Define Go structs matching config schema
- Provide `Load()` function for configuration loading

### `internal/auth`

Authentication and authorization. Responsibilities:

- Resolve SSH/Unix user to configured user
- Create Principal with roles
- FIDO2 authentication (YubiKey integration)
- Role-based access control checks

### `internal/clients`

Resource client implementations. Responsibilities:

- HTTP client for HTTP resources
- PostgreSQL client for Postgres resources
- Abstract resource access behind interfaces

### `internal/logging`

Audit logging. Responsibilities:

- SQLite database management (WAL mode)
- Audit entry creation and storage
- Recent log retrieval for TUI display

### `internal/openapi`

OpenAPI operation generation. Responsibilities:

- Load OpenAPI specifications from URLs
- Parse and validate OpenAPI documents
- Generate Operation entries from endpoints
- Filter endpoints by tags and request body requirements

### `internal/tasks`

Task execution engine. Responsibilities:

- Execute task steps in order
- Apply error handling policies
- Render summary templates
- Log step and task execution

### `internal/ui`

Terminal user interface (Bubble Tea). Responsibilities:

- Operations view with filtering
- Tasks view
- Logs view
- Help view
- User input handling and display

## Startup Flow

```
1. Load Configuration
   └─> config.Load()
       └─> Parse YAML file
       └─> Validate structure

2. Load OpenAPI Operations (if configured)
   └─> openapi.GenerateOperations()
       └─> Fetch OpenAPI specs
       └─> Generate Operation entries
       └─> Append to operations list

3. Resolve Principal
   └─> auth.ResolvePrincipal()
       └─> Get SSH/Unix username
       └─> Match against users[].ssh_users[]
       └─> Create Principal

4. FIDO2 Authentication (if required)
   └─> auth.RequireYubiKeyIfConfigured()
       └─> Generate challenge
       └─> Request assertion
       └─> Verify signature

5. Initialize Components
   ├─> logging.NewAuditLogger()
   ├─> clients.NewHTTPClient() for each HTTP resource
   ├─> clients.NewPostgresClient() for each Postgres resource
   └─> tasks.NewRunner()

6. Start TUI
   └─> ui.NewModel()
   └─> tea.NewProgram().Start()
```

## Runtime Flows

### Operation Execution

```
User selects operation in TUI
  └─> ui.Model.runOperation()
      └─> clients.HTTPClient.Request() or PostgresClient.RunScalarQuery()
          └─> Execute request/query
          └─> Return result
      └─> logging.AuditLogger.Log()
          └─> Write audit entry
      └─> Update TUI with result
```

### Task Execution

```
User selects task in TUI
  └─> ui.Model.runTask()
      └─> tasks.Runner.Run()
          ├─> For each step:
          │   ├─> Execute step (HTTP/Postgres/Sleep)
          │   ├─> logging.AuditLogger.Log() (step entry)
          │   └─> Apply error policy
          ├─> tasks.RenderSummary()
          │   └─> Execute Go template
          └─> logging.AuditLogger.Log() (task entry)
      └─> Update TUI with result and summary
```

### Log View

```
User opens logs view
  └─> ui.Model.withLoadedLogs()
      └─> logging.ReadRecent()
          └─> Query SQLite database
          └─> Return recent entries
      └─> Display in table format
```

## Data Structures

### Principal

```go
type Principal struct {
    ConfigUser *config.User  // Matched user from config
    SSHUser    string        // Actual SSH/Unix username
}
```

### Operation Result

```go
type operationResultMsg struct {
    op     config.Operation
    output string
    errMsg string
}
```

### Task Result

```go
type TaskResult struct {
    Task      config.Task
    Success   bool
    StepOrder []string
    Steps     map[string]StepResult
}

type StepResult struct {
    Step   config.TaskStep
    OK     bool
    Output string
    Err    error
}
```

## Component Interactions

### Configuration → Operations/Tasks

Operations and tasks reference resources by name. At runtime:

1. Operation/Task references resource name
2. Client map lookup resolves to client instance
3. Client executes request with resource configuration

### Auth → UI

Principal determines what operations and tasks are visible:

1. Principal resolved at startup
2. UI filters operations/tasks by `allowed_roles[]`
3. Only matching items shown in TUI

### Logging → All Components

All execution paths write to audit log:

1. Operation execution → log entry
2. Task step execution → log entry per step
3. Task completion → log entry for task

## Error Handling

### Configuration Errors

- Invalid YAML → fatal error at startup
- Missing required fields → fatal error at startup
- Invalid resource references → fatal error at startup

### Runtime Errors

- Operation execution failure → logged, displayed in TUI
- Task step failure → handled by error policy, logged
- Resource connection failure → logged, operation/task fails

### FIDO2 Errors

- No credentials configured → error if required
- Assertion failure → fatal error
- Signature verification failure → fatal error

## Extension Points

### Adding Resource Types

1. Add resource type to `config.ResourcesConfig`
2. Implement client in `internal/clients`
3. Add client initialization in `cmd/lazyadmin/main.go`
4. Update `internal/tasks` to support new step type

### Adding Step Types

1. Add step type string constant
2. Implement execution in `tasks.Runner.runStep()`
3. Update config documentation

### Adding TUI Views

1. Add mode constant
2. Implement `update{Mode}()` and `view{Mode}()` methods
3. Add keybinding in `updateMain()`

