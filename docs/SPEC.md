# lazyadmin Specification v0.1.0

This document defines the normative behavior of lazyadmin. Code implementations MUST conform to this specification.

## 1. Scope and Non-Goals

### Scope

lazyadmin is a terminal-based operations management system that:

- Runs as a TUI application in a container or on a host system
- Communicates with HTTP endpoints and PostgreSQL databases over private networks
- Provides a control plane for administrative operations with role-based access control
- Maintains an audit log of all operations and task executions
- Supports FIDO2 authentication via YubiKey devices

### Non-Goals

- Arbitrary shell command execution
- Untrusted user input processing
- Multi-tenant RBAC (single config file, single deployment)
- Network listeners or remote access (TUI only)
- General-purpose scripting or automation

## 2. Core Domain Model

### 2.1 User

A **User** is a configured identity defined in the YAML configuration. Each user has:

- `id`: Unique identifier within the configuration
- `ssh_users`: List of SSH/Unix usernames that map to this user
- `roles`: List of role strings assigned to this user
- `yubikey_credentials`: List of FIDO2 credential configurations

### 2.2 Principal

A **Principal** is the resolved runtime identity, consisting of:

- A reference to the matched `User` from configuration
- The actual SSH/Unix username string used for mapping

The principal is resolved at startup by matching the current SSH/Unix user against `users[].ssh_users[]`.

### 2.3 Role

A **Role** is a string identifier used for access control. Roles have no inherent semantics; they are matched by string equality against `operation.allowed_roles[]` and `task.allowed_roles[]`.

### 2.4 Resource

A **Resource** is a named connection target for operations. Resources are typed:

- **HTTP Resource**: Named HTTP endpoint with a base URL
- **Postgres Resource**: Named PostgreSQL connection with a DSN from environment

Resources are referenced by name in operations and task steps via the `target` or `resource` field.

### 2.5 Operation

An **Operation** is an atomic, single-action request:

- **HTTP Operation**: Executes an HTTP method + path against a named HTTP resource
- **Postgres Operation**: Executes a scalar SQL query against a named Postgres resource

Operations have:
- `id`: Unique identifier
- `label`: Display name in TUI
- `type`: "http" or "postgres"
- `target`: Name of the resource to use
- `allowed_roles`: List of roles that may execute this operation

### 2.6 Task

A **Task** is an ordered sequence of Steps with:

- `id`: Unique identifier
- `label`: Display name in TUI
- `allowed_roles`: List of roles that may execute this task
- `risk_level`: "low", "medium", or "high"
- `require_yubikey`: Boolean flag for additional authentication
- `on_error`: Task-level error policy ("fail_fast" or "best_effort")
- `steps`: Ordered list of Step definitions
- `summary_template`: Optional Go template for rendering results

### 2.7 Step

A **Step** is a typed action within a task:

- **http**: HTTP request (method + path on resource)
- **postgres**: Scalar SQL query on resource
- **sleep**: Delay execution for N seconds

Steps have:
- `id`: Unique identifier within the task
- `type`: Step type identifier
- `resource`: Name of resource (except for sleep)
- `on_error`: Step-level error policy override

## 3. Configuration Model

### 3.1 Configuration Loading

Configuration MUST be loaded from a single YAML file. The file path is determined by:

1. Environment variable `LAZYADMIN_CONFIG_PATH` if set
2. Default path `config/lazyadmin.yaml` otherwise

Configuration loading MUST fail if:
- The file cannot be read
- The YAML is invalid or cannot be parsed
- Required top-level keys are missing

### 3.2 Configuration Invariants

The following invariants MUST be enforced:

- Every `operation.target` MUST reference a key in `resources.http` or `resources.postgres`
- Every `task.steps[].resource` MUST reference a key in the appropriate `resources.*` map (except sleep steps)
- Every role string in `operation.allowed_roles[]` and `task.allowed_roles[]` MUST exist in at least one `users[].roles[]` entry
- Every `users[].ssh_users[]` entry MUST be non-empty
- Every user MUST have at least one role

### 3.3 OpenAPI Integration

If `openapi.backends` is defined, the system MUST:

1. Load each OpenAPI specification from the configured URL
2. Validate the specification
3. Generate `Operation` entries for eligible endpoints
4. Append generated operations to the static `operations[]` list

OpenAPI operations are eligible if:
- The endpoint matches tag filters (if configured)
- The endpoint has no required request body
- The endpoint method is GET, POST, PUT, DELETE, or PATCH

## 4. Authentication and RBAC

### 4.1 Principal Resolution

At startup, the system MUST:

1. Determine the current SSH/Unix username via:
   - `SSH_USER` environment variable (if set)
   - `USER` environment variable (if set)
   - `os/user.Current()` system call (if available)
   - "unknown" as fallback

2. Match the username against `users[].ssh_users[]` entries

3. If no match is found, the program MUST exit with an error

4. If a match is found, create a Principal with the matched User and SSH username

### 4.2 Role-Based Access Control

A Principal MAY execute an Operation or Task if and only if:

- The Principal's User has at least one role that appears in the Operation's or Task's `allowed_roles[]` list

The TUI MUST filter Operations and Tasks to show only those allowed by the current Principal.

### 4.3 FIDO2 Authentication

If `auth.require_yubikey` is `true`, the system MUST:

1. Require a successful FIDO2 assertion before entering the TUI
2. Use the first credential from `principal.ConfigUser.YubiKeyCreds[]`
3. Generate a random 32-byte challenge
4. Request an assertion from the YubiKey device
5. Verify the assertion signature against the stored public key
6. If verification fails, the program MUST exit with an error

For tasks with `require_yubikey: true` or `risk_level: "high"`, the system MAY require an additional FIDO2 assertion (currently not implemented).

## 5. Operations

### 5.1 HTTP Operations

An HTTP Operation executes a single HTTP request:

- Method: From `operation.method` (MUST be uppercase)
- Path: From `operation.path` (appended to resource base URL)
- Resource: Resolved from `operation.target` in `resources.http`

Success criteria:
- HTTP status code 2xx-3xx is considered success
- Any other status code or network error is failure

Output representation:
- Success: `"HTTP {status_code} {status_text}"`
- Failure: Error message string

### 5.2 Postgres Operations

A Postgres Operation executes a scalar SQL query:

- Query: From `operation.query`
- Resource: Resolved from `operation.target` in `resources.postgres`

Success criteria:
- Query executes without error
- Query returns exactly one row with one column
- Any error or wrong row count is failure

Output representation:
- Success: String representation of the scalar value
- Failure: Error message string

### 5.3 Operation Execution

Every Operation execution MUST:

1. Create an audit log entry before execution
2. Execute the operation via the appropriate client
3. Update the audit log entry with success/failure status
4. Display the result in the TUI

## 6. Tasks and Steps

### 6.1 Step Types

#### HTTP Step

- Type: `"http"`
- Fields: `resource`, `method`, `path`
- Success: HTTP status 2xx-3xx
- Output: Status code and text

#### Postgres Step

- Type: `"postgres"`
- Fields: `resource`, `query`
- Success: Query executes and returns one scalar value
- Output: String representation of value

#### Sleep Step

- Type: `"sleep"`
- Fields: `seconds`
- Success: Delay completes without context cancellation
- Output: Duration string

### 6.2 Error Handling Policies

Task-level `on_error` policy:

- `fail_fast` (default): Stop execution after first failing step
- `best_effort`: Continue executing remaining steps regardless of failures

Step-level `on_error` override:

- `inherit`: Use task-level policy
- `fail`: Mark task as failed and stop (overrides task policy)
- `warn`: Mark task as failed but continue executing steps
- `continue`: Ignore failure and continue (does not mark task as failed)

### 6.3 Task Success Definition

A Task is considered successful if:

- All steps executed without error, OR
- All step failures used `on_error: continue` policy

A Task is considered failed if:

- Any step failed with `on_error: fail`, OR
- Any step failed with `on_error: warn`, OR
- Task-level `fail_fast` policy triggered

### 6.4 Task Execution

Task execution MUST:

1. Log an audit entry for the task start
2. Execute steps in order
3. Log an audit entry for each step execution
4. Apply error policies as defined
5. Render summary template if provided
6. Log final task result

## 7. Audit Logging

### 7.1 Log Entries

The system MUST create audit log entries for:

- Every Operation execution (one entry per execution)
- Every Task Step execution (one entry per step)
- Every Task completion (one entry per task)

### 7.2 Log Entry Fields

Each audit log entry contains:

- `occurred_at`: RFC3339Nano timestamp (UTC)
- `user_id`: User ID from configuration
- `ssh_user`: Actual SSH/Unix username
- `operation_id`: Operation ID, task ID, or "task:{id} step:{step_id}"
- `success`: Boolean success indicator
- `error`: Error message string (if failure)

### 7.3 Log Storage

Audit logs are stored in SQLite with:

- WAL (Write-Ahead Logging) mode enabled
- Append-only schema (no UPDATE or DELETE operations)
- Automatic schema creation on first use

## 8. TUI Behavior

### 8.1 Views

The TUI provides four views:

- **Operations View**: List of allowed operations, filterable by type
- **Tasks View**: List of allowed tasks
- **Logs View**: Recent audit log entries in table format
- **Help View**: Keybinding reference

### 8.2 View Filtering

- Operations View MUST show only operations allowed by current Principal
- Tasks View MUST show only tasks allowed by current Principal
- Operations View MAY be filtered by type (all, HTTP, Postgres)
- Logs View MUST show the N most recent entries ordered by time descending

### 8.3 Operation Execution

When an operation is executed:

1. The operation runs asynchronously
2. The TUI displays a result message
3. Success or error is shown in the details area
4. The audit log entry is written

### 8.4 Task Execution

When a task is executed:

1. The task runs asynchronously
2. Step-by-step progress is tracked
3. Final result and summary are displayed
4. Audit log entries are written for each step and the task

## 9. Versioning and Compatibility

### 9.1 Specification Versioning

Specification versions follow semantic versioning:

- Major version: Breaking changes to behavior or config
- Minor version: New features, backward-compatible additions
- Patch version: Clarifications, bug fixes

### 9.2 Configuration Compatibility

- Breaking changes to config structure MUST increment major or minor version
- Backward-compatible additions MUST use optional fields with sensible defaults
- Deprecated fields MUST be supported for at least one major version

### 9.3 Implementation Version

The current implementation is v0.1.0. This version is experimental and not production-ready.

