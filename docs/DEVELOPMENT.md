# Development Guide

## Project Overview

lazyadmin is a terminal-based operations management system built in Go. The system provides a TUI for executing administrative operations across HTTP endpoints and PostgreSQL databases with role-based access control and audit logging.

## Components

### Configuration System (`internal/config`)

YAML-based configuration loader supporting users, roles, resources, operations, tasks, and OpenAPI backends. Includes configuration validation and invariant checking.

### Authentication & Authorization (`internal/auth`)

SSH/Unix user to config user mapping with role-based access control. Includes FIDO2 authentication framework (requires libfido2 integration).

### Resource Clients (`internal/clients`)

HTTP and PostgreSQL clients for typed resource access. Each resource type has a dedicated client implementation.

### Audit Logging (`internal/logging`)

SQLite-based audit logger using WAL mode. Logs all operation and task executions with user attribution and success/failure status.

### Terminal User Interface (`internal/ui`)

Bubble Tea TUI with multiple views: operations, tasks, logs, and help. Supports operation filtering and task execution with progress display.

### Task Workflow System (`internal/tasks`)

Multi-step task execution engine with configurable error handling policies (fail_fast, best_effort). Supports HTTP, Postgres, and sleep step types with summary template rendering.

### OpenAPI Integration (`internal/openapi`)

OpenAPI specification loader that generates operations automatically from endpoint definitions. Includes tag filtering and request body validation.

### Docker Infrastructure

Distroless Dockerfile for hardened container builds. Includes docker-compose.yml with dev stack (lazyadmin, backend, postgres, caddy) and Express test backend.

### Documentation

Complete documentation suite including normative specification (SPEC.md), configuration reference (CONFIG.md), architecture guide (ARCHITECTURE.md), security model (SECURITY.md), and UX specification (UX.md).

## Local Development

### Prerequisites

- Go 1.25+
- Docker and docker-compose
- [mise](https://mise.jdx.dev/) (optional, for tool version management)

### Setup

1. **Install development tools:**

   ```bash
   mise install
   ```

2. **Start the development stack:**

   ```bash
   docker compose up backend postgres caddy -d
   ```

3. **Initialize test database (if needed):**
   ```bash
   docker exec -it $(docker ps -qf "name=postgres") psql -U testuser -d testdb -c 'CREATE TABLE users(id serial primary key, email text);'
   ```

### Running Locally

**Option 1: Run in Docker (recommended for testing):**

```bash
SSH_USER=$USER docker compose run --rm lazyadmin
```

**Option 2: Run natively:**

```bash
LAZYADMIN_PG_DSN="postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable" \
go run ./cmd/lazyadmin
```

### Building Locally

**Build the binary:**

```bash
go build ./cmd/lazyadmin
```

**Build the Docker image:**

```bash
docker build -t lazyadmin:local .
```

**Run the built image:**

```bash
docker run --rm -it \
  -v $(pwd)/config/lazyadmin.yaml:/run/lazyadmin/config.yaml:ro \
  -v lazyadmin-data:/var/lib/lazyadmin \
  -e SSH_USER=$USER \
  -e LAZYADMIN_PG_DSN="postgres://testuser:testpass@host.docker.internal:5433/testdb?sslmode=disable" \
  lazyadmin:local
```

### Testing

**Run Go tests:**

```bash
go test ./...
```

**Run with verbose output:**

```bash
go test -v ./...
```

**Run specific package tests:**

```bash
go test ./internal/config
```

### Development Workflow

1. Make code changes
2. Test locally:
   ```bash
   go build ./cmd/lazyadmin
   go test ./...
   ```
3. Test with Docker:
   ```bash
   docker compose up --build lazyadmin
   ```
4. Update documentation if behavior changes
5. Commit changes with descriptive messages

## Docker Compose Stack

The `docker-compose.yml` includes:

- **backend**: Express test server (port 3000)
- **postgres**: PostgreSQL database (port 5433)
- **caddy**: Reverse proxy (port 8080)
- **lazyadmin**: Main application (built from Dockerfile)

**Useful commands:**

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f lazyadmin

# Rebuild and restart
docker compose up --build -d

# Stop all services
docker compose down

# Remove volumes
docker compose down -v
```

## Configuration

Edit `config/lazyadmin.yaml` to add users, define resources, create operations and tasks, and configure OpenAPI backends.

See [`docs/CONFIG.md`](docs/CONFIG.md) for complete reference.

## Project Structure

```
lazyadmin/
  cmd/lazyadmin/      Application entry point
  internal/
    auth/             Authentication and authorization
    clients/          HTTP and PostgreSQL clients
    config/           Configuration loading and types
    logging/          SQLite audit logger
    openapi/          OpenAPI operation generator
    tasks/            Task execution engine
    ui/               Bubble Tea TUI implementation
  config/             YAML configuration files
  backend/            Express test backend
  docs/               Documentation
  Dockerfile          Container build definition
  docker-compose.yml  Development stack
```

## Troubleshooting

**mise trust error:**

```bash
mise trust
```

**Postgres connection issues:**

- Verify `LAZYADMIN_PG_DSN` environment variable
- Check postgres container is running: `docker compose ps`
- Test connection: `docker compose exec postgres psql -U testuser -d testdb`

**TUI not displaying:**

- Ensure terminal supports ANSI colors
- Check `SSH_USER` or `USER` environment variable is set
- Verify user exists in `config/lazyadmin.yaml`

**Build failures:**

- Run `go mod tidy` to update dependencies
- Check Go version: `go version` (requires 1.23+)
- Clear module cache: `go clean -modcache`
