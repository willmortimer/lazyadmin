# lazyadmin

Terminal-based operations management system with config-driven RBAC, audit logging, and FIDO2 authentication support.

lazyadmin provides a TUI for executing administrative operations across HTTP endpoints and PostgreSQL databases. Operations are defined in YAML configuration with role-based access control. All operations are logged to SQLite for audit purposes.

## Features

- Typed **operations** (HTTP + Postgres)
- Multi-step **tasks** with error handling policies
- YAML configuration for users, roles, resources, operations, and tasks
- WAL-backed SQLite audit log
- Optional YubiKey (FIDO2) authentication
- TUI with operations list, tasks list, logs view, and help

## Quick Start

### Prerequisites

- Go 1.23+
- Docker and docker-compose
- [mise](https://mise.jdx.dev/) (optional, for tool version management)

### Development Setup

```bash
mise install                                    # Install Go/Node versions
docker compose up backend postgres caddy -d     # Start dev stack
SSH_USER=$USER docker compose run --rm lazyadmin  # Run lazyadmin
```

See [`docs/CONFIG.md`](docs/CONFIG.md) for configuration details.

## Concepts

- **[Operation](docs/SPEC.md#operations)**: Atomic single-action request (HTTP or SQL query)
- **[Task](docs/SPEC.md#tasks-and-steps)**: Multi-step workflow with error handling
- **[Resource](docs/SPEC.md#resources)**: Named connection target (HTTP endpoint or PostgreSQL database)
- **[User / Roles](docs/SPEC.md#authentication-and-rbac)**: Identity mapping and role-based access control
- **[Audit Logging](docs/SPEC.md#audit-logging)**: SQLite-based operation and task execution logging

## Documentation

- **[SPEC.md](docs/SPEC.md)**: Canonical behavior specification (start here)
- **[CONFIG.md](docs/CONFIG.md)**: YAML configuration reference
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)**: Code structure and data flow
- **[SECURITY.md](docs/SECURITY.md)**: Threat model and security guarantees
- **[UX.md](docs/UX.md)**: TUI layout, keybindings, and UX rules

## Status

**v0.1.0** – Experimental, not production-ready.

Current implementation includes:

- Operations (HTTP, Postgres)
- Tasks with multi-step execution
- OpenAPI operation generation
- FIDO2 authentication (requires libfido2 integration)
- Audit logging

## License

MIT
