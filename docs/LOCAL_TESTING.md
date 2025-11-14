# Local Testing Guide

Quick reference for testing the lazyadmin stack locally.

## Prerequisites

- **Go 1.25+** (managed via mise)
- **Docker & docker-compose**
- **mise** (for tool version management)

## Quick Start

### 1. Install Tools

```bash
mise install  # Installs Go 1.25 and Node.js 25.2.0
```

### 2. Start the Development Stack

```bash
# Start backend, postgres, and caddy services
docker compose up backend postgres caddy -d

# Or use the mise task
mise run dev
```

**Services started:**
- **Backend** (Express): `http://localhost:3000` (internal) → `http://localhost:8080` (via Caddy)
- **PostgreSQL**: `localhost:5433` (user: `testuser`, password: `testpass`, db: `testdb`)
- **Caddy** (reverse proxy): `http://localhost:8080`

### 3. Initialize Test Database (Optional)

```bash
# Create a test table
docker exec -it $(docker ps -qf "name=postgres") psql -U testuser -d testdb -c 'CREATE TABLE users(id serial primary key, email text);'

# Insert test data
docker exec -it $(docker ps -qf "name=postgres") psql -U testuser -d testdb -c "INSERT INTO users(email) VALUES ('test@example.com');"
```

### 4. Run lazyadmin

**Option A: Run in Docker (Recommended)**

```bash
# SSH_USER is automatically set from your $USER environment variable
docker compose run --rm lazyadmin

# Or explicitly override it:
SSH_USER=$USER docker compose run --rm lazyadmin
```

**Option B: Run Natively**

```bash
# Set environment variables
export SSH_USER=$USER
export LAZYADMIN_PG_DSN="postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable"
export LAZYADMIN_CONFIG_PATH="config/lazyadmin.yaml"

# Run the TUI
go run ./cmd/lazyadmin

# Or use the mise task
mise run tui
```

## Testing Commands

### Run Unit Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/config
```

### Run Linters

```bash
# Check code with go vet
go vet ./...

# Check formatting
go fmt -d .

# Verify dependencies
go mod verify
```

### Build & Test Binary

```bash
# Build the binary
go build ./cmd/lazyadmin

# Run the built binary
./lazyadmin
```

## Testing the Stack

### 1. Test HTTP Operations

The stack includes a test backend with a `/health` endpoint:

```bash
# Test backend directly
curl http://localhost:3000/health

# Test via Caddy proxy
curl http://localhost:8080/health
```

In lazyadmin TUI:
- Press `h` to filter HTTP operations
- Select "Backend HTTP health" operation
- Press `enter` to execute

### 2. Test Postgres Operations

```bash
# Connect to database directly
docker exec -it $(docker ps -qf "name=postgres") psql -U testuser -d testdb

# Or from host
psql -h localhost -p 5433 -U testuser -d testdb
```

In lazyadmin TUI:
- Press `p` to filter Postgres operations
- Select "Count users" operation
- Press `enter` to execute

### 3. Test Tasks

Tasks are multi-step workflows. Check `config/lazyadmin.yaml` for configured tasks.

In lazyadmin TUI:
- Press `t` to switch to tasks view
- Select a task
- Press `enter` to execute

### 4. View Audit Logs

In lazyadmin TUI:
- Press `l` to view recent audit logs
- Shows all operation and task executions

## Useful Docker Commands

```bash
# View logs
docker compose logs -f lazyadmin
docker compose logs -f backend
docker compose logs -f postgres

# Check service status
docker compose ps

# Stop all services
docker compose down

# Stop and remove volumes
docker compose down -v

# Rebuild and restart
docker compose up --build -d

# Execute commands in containers
docker compose exec postgres psql -U testuser -d testdb
docker compose exec backend sh
```

## Configuration

Edit `config/lazyadmin.yaml` to:
- Add/modify users and roles
- Configure resources (HTTP endpoints, Postgres databases)
- Define operations (HTTP requests, SQL queries)
- Create tasks (multi-step workflows)

See [`docs/CONFIG.md`](docs/CONFIG.md) for complete configuration reference.

## Troubleshooting

### Services Won't Start

```bash
# Check if ports are in use
lsof -i :3000  # Backend
lsof -i :5433  # Postgres
lsof -i :8080  # Caddy

# Check Docker logs
docker compose logs
```

### Database Connection Issues

```bash
# Verify postgres is running
docker compose ps postgres

# Test connection
docker compose exec postgres psql -U testuser -d testdb -c "SELECT 1;"

# Check environment variable
echo $LAZYADMIN_PG_DSN
```

### TUI Not Displaying

- Ensure terminal supports ANSI colors
- Check `SSH_USER` or `USER` environment variable is set
- Verify user exists in `config/lazyadmin.yaml` (matches your `$USER`)

### Go Toolchain Issues

```bash
# Verify Go version
go version

# Check GOTOOLCHAIN setting
go env GOTOOLCHAIN

# Should be "local" (set in .mise.toml)
```

## Development Workflow

1. **Make code changes**
2. **Run tests**: `go test ./...`
3. **Check linting**: `go vet ./...`
4. **Test locally**: `go run ./cmd/lazyadmin` or `docker compose run --rm lazyadmin`
5. **Commit changes**

## Testing Checklist

- [ ] All unit tests pass (`go test ./...`)
- [ ] Linting passes (`go vet ./...`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] Docker stack starts successfully
- [ ] HTTP operations work (backend health check)
- [ ] Postgres operations work (user count query)
- [ ] Tasks execute successfully
- [ ] Audit logs are recorded
- [ ] TUI displays correctly
- [ ] User authentication works (SSH user mapping)

