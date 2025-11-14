# Configuration Reference

lazyadmin configuration is defined in a single YAML file, resolved via `LAZYADMIN_CONFIG_PATH` environment variable or defaulting to `config/lazyadmin.yaml`.

## Top-Level Structure

```yaml
project: string
env: string
logging:
  sqlite_path: string
auth:
  require_yubikey: boolean
  yubikey_mode: string
users: []
resources:
  http: {}
  postgres: {}
operations: []
tasks: []
openapi:
  backends: {}
```

## Project and Environment

### `project`

- **Type**: string
- **Required**: Yes
- **Description**: Project identifier for display in TUI

### `env`

- **Type**: string
- **Required**: Yes
- **Description**: Environment identifier (e.g., "dev", "prod")

**Example:**

```yaml
project: lazyadmin-demo
env: dev
```

## Logging

### `logging.sqlite_path`

- **Type**: string
- **Required**: Yes
- **Description**: File system path for SQLite audit log database
- **Default**: None

**Example:**

```yaml
logging:
  sqlite_path: /var/lib/lazyadmin/db.sqlite
```

## Authentication

### `auth.require_yubikey`

- **Type**: boolean
- **Required**: Yes
- **Description**: Require FIDO2 YubiKey authentication before TUI access
- **Default**: `false`

### `auth.yubikey_mode`

- **Type**: string
- **Required**: Yes
- **Description**: Authentication mode identifier (currently "fido2")
- **Default**: `"fido2"`

**Example:**

```yaml
auth:
  require_yubikey: false
  yubikey_mode: fido2
```

## Users

### `users[]`

- **Type**: array of user objects
- **Required**: Yes
- **Description**: List of configured users with SSH mappings and roles

### User Object

```yaml
id: string                    # Unique user identifier
ssh_users: []                 # List of SSH/Unix usernames
roles: []                     # List of role strings
yubikey_credentials: []       # List of FIDO2 credentials
```

### `users[].id`

- **Type**: string
- **Required**: Yes
- **Description**: Unique identifier for this user

### `users[].ssh_users[]`

- **Type**: array of strings
- **Required**: Yes
- **Description**: SSH/Unix usernames that map to this user
- **Note**: Must be non-empty

### `users[].roles[]`

- **Type**: array of strings
- **Required**: Yes
- **Description**: Role identifiers assigned to this user
- **Note**: Must contain at least one role

### `users[].yubikey_credentials[]`

- **Type**: array of credential objects
- **Required**: No
- **Description**: FIDO2 credential configurations for this user

### YubiKey Credential Object

```yaml
rp_id: string                 # Relying Party ID
credential_id: string         # Base64URL-encoded credential ID
public_key: string            # Base64URL-encoded public key
```

**Example:**

```yaml
users:
  - id: admin
    ssh_users: ["admin", "root"]
    roles: ["owner", "admin"]
    yubikey_credentials:
      - rp_id: "lazyadmin.local"
        credential_id: "BASE64URL_CRED_ID"
        public_key: "BASE64URL_PUBLIC_KEY"
  - id: operator
    ssh_users: ["operator"]
    roles: ["read_only"]
    yubikey_credentials: []
```

## Resources

### `resources.http`

- **Type**: map of HTTP resource objects
- **Required**: No
- **Description**: Named HTTP endpoint configurations

### HTTP Resource Object

```yaml
base_url: string              # Base URL for HTTP requests
```

**Example:**

```yaml
resources:
  http:
    backend:
      base_url: http://backend:3000
    api:
      base_url: https://api.example.com
```

### `resources.postgres`

- **Type**: map of Postgres resource objects
- **Required**: No
- **Description**: Named PostgreSQL connection configurations

### Postgres Resource Object

```yaml
dsn_env: string               # Environment variable containing DSN
```

**Example:**

```yaml
resources:
  postgres:
    main:
      dsn_env: LAZYADMIN_PG_DSN
    analytics:
      dsn_env: LAZYADMIN_ANALYTICS_DSN
```

## Operations

### `operations[]`

- **Type**: array of operation objects
- **Required**: No
- **Description**: Static operation definitions

### Operation Object

```yaml
id: string                    # Unique operation identifier
label: string                 # Display name in TUI
type: string                  # "http" or "postgres"
target: string                # Resource name
method: string                # HTTP method (for http type)
path: string                  # HTTP path (for http type)
query: string                 # SQL query (for postgres type)
allowed_roles: []             # List of role strings
```

### `operations[].id`

- **Type**: string
- **Required**: Yes
- **Description**: Unique identifier for this operation

### `operations[].label`

- **Type**: string
- **Required**: Yes
- **Description**: Human-readable name displayed in TUI

### `operations[].type`

- **Type**: string
- **Required**: Yes
- **Values**: `"http"` or `"postgres"`
- **Description**: Operation type

### `operations[].target`

- **Type**: string
- **Required**: Yes
- **Description**: Name of resource in `resources.http` or `resources.postgres`

### `operations[].method`

- **Type**: string
- **Required**: Yes (for http type)
- **Description**: HTTP method (GET, POST, PUT, DELETE, etc.)

### `operations[].path`

- **Type**: string
- **Required**: Yes (for http type)
- **Description**: HTTP path appended to resource base URL

### `operations[].query`

- **Type**: string
- **Required**: Yes (for postgres type)
- **Description**: SQL query returning a single scalar value

### `operations[].allowed_roles[]`

- **Type**: array of strings
- **Required**: Yes
- **Description**: Role identifiers that may execute this operation

**Example:**

```yaml
operations:
  - id: backend_health
    label: "Backend HTTP health"
    type: http
    target: backend
    method: GET
    path: /health
    allowed_roles: ["owner", "admin", "read_only"]
  
  - id: db_user_count
    label: "Count users"
    type: postgres
    target: main
    query: "SELECT COUNT(*) FROM users"
    allowed_roles: ["owner", "admin"]
```

## Tasks

### `tasks[]`

- **Type**: array of task objects
- **Required**: No
- **Description**: Multi-step workflow definitions

### Task Object

```yaml
id: string                    # Unique task identifier
label: string                 # Display name in TUI
allowed_roles: []             # List of role strings
risk_level: string            # "low", "medium", or "high"
require_yubikey: boolean      # Require additional YubiKey auth
on_error: string              # "fail_fast" or "best_effort"
steps: []                     # List of step objects
summary_template: string      # Go template for results
```

### `tasks[].risk_level`

- **Type**: string
- **Required**: No
- **Values**: `"low"`, `"medium"`, `"high"`
- **Default**: `"low"`
- **Description**: Risk classification for UX and security policies

### `tasks[].require_yubikey`

- **Type**: boolean
- **Required**: No
- **Default**: `false`
- **Description**: Require additional FIDO2 authentication for this task

### `tasks[].on_error`

- **Type**: string
- **Required**: No
- **Values**: `"fail_fast"`, `"best_effort"`
- **Default**: `"fail_fast"`
- **Description**: Task-level error handling policy

### `tasks[].steps[]`

- **Type**: array of step objects
- **Required**: Yes
- **Description**: Ordered list of steps to execute

### Step Object

```yaml
id: string                    # Unique step identifier within task
type: string                  # "http", "postgres", or "sleep"
resource: string              # Resource name (except sleep)
method: string                # HTTP method (for http type)
path: string                  # HTTP path (for http type)
query: string                 # SQL query (for postgres type)
seconds: integer              # Delay seconds (for sleep type)
on_error: string              # Step-level error policy override
```

### `tasks[].steps[].on_error`

- **Type**: string
- **Required**: No
- **Values**: `"inherit"`, `"fail"`, `"warn"`, `"continue"`
- **Default**: `"inherit"`
- **Description**: Override task-level error policy for this step

### `tasks[].summary_template`

- **Type**: string
- **Required**: No
- **Description**: Go `text/template` string for rendering task results

Template context:

```go
type Context struct {
    Task    config.Task
    Success bool
    Steps   map[string]StepView
}

type StepView struct {
    OK     bool
    Output string
    Error  string
}
```

**Example:**

```yaml
tasks:
  - id: full_health_check
    label: "Full health check"
    allowed_roles: ["owner", "admin"]
    risk_level: low
    on_error: fail_fast
    steps:
      - id: http_health
        type: http
        resource: backend
        method: GET
        path: /health
        on_error: fail
      - id: db_user_count
        type: postgres
        resource: main
        query: "SELECT COUNT(*) FROM users"
        on_error: warn
      - id: sleep_short
        type: sleep
        seconds: 1
    summary_template: |
      Health: {{ .Steps.http_health.Output }}
      Users:  {{ .Steps.db_user_count.Output }}
      Overall: {{ if .Success }}Success{{ else }}Failed{{ end }}
```

## OpenAPI Integration

### `openapi.backends`

- **Type**: map of backend objects
- **Required**: No
- **Description**: OpenAPI specification sources for automatic operation generation

### OpenAPI Backend Object

```yaml
doc_url: string               # URL to OpenAPI JSON specification
tag_filter: []                # List of tag strings to include
include_untagged: boolean     # Include endpoints without tags
op_id_prefix: string          # Prefix for generated operation IDs
```

### `openapi.backends[].tag_filter[]`

- **Type**: array of strings
- **Required**: No
- **Description**: Only endpoints with these tags are included
- **Note**: If empty and `include_untagged: false`, only tagged endpoints are included

### `openapi.backends[].include_untagged`

- **Type**: boolean
- **Required**: No
- **Default**: `false`
- **Description**: Include endpoints that have no tags

### `openapi.backends[].op_id_prefix`

- **Type**: string
- **Required**: No
- **Description**: Prefix added to generated operation IDs

**Example:**

```yaml
openapi:
  backends:
    backend:
      doc_url: "http://backend:3000/openapi.json"
      tag_filter: ["admin", "lazyadmin"]
      include_untagged: false
      op_id_prefix: "api_"
```

## Configuration Validation

The following validation rules are enforced:

1. All `operation.target` values must reference existing resources
2. All `task.steps[].resource` values must reference existing resources (except sleep)
3. All role strings in `allowed_roles[]` must exist in at least one user's roles
4. All users must have at least one role
5. All `ssh_users[]` arrays must be non-empty
6. HTTP operations must have `method` and `path` fields
7. Postgres operations must have `query` field
8. Tasks must have at least one step

