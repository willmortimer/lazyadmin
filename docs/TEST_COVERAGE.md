# Test Coverage Analysis

This document analyzes the current test coverage and provides guidance on testing each component.

## Current Status

**No tests exist** - This is documented in `NEXT_STEPS.md` as a high-priority item.

## Package-by-Package Analysis

### ✅ `internal/config` - **Highly Testable**

**Why it needs tests:**
- Core functionality - config loading is critical
- Pure functions - easy to test
- File I/O - needs error handling verification

**What to test:**
- `Load()` with valid YAML
- `Load()` with invalid YAML (malformed, wrong types)
- `Load()` with missing file
- `Load()` with custom path via `LAZYADMIN_CONFIG_PATH`
- Default path fallback

**How to test:**
- Use temporary files with test YAML content
- Test various valid and invalid configurations
- Mock `os.ReadFile` or use real temp files

**Difficulty:** ⭐ Easy

---

### ✅ `internal/auth` - **Partially Testable**

**Why it needs tests:**
- Security-critical - authentication bugs are serious
- Role checking logic needs verification
- Principal resolution logic

**What to test:**
- `CurrentSSHUser()` with different env vars (`SSH_USER`, `USER`, fallback)
- `ResolvePrincipal()` with matching SSH user
- `ResolvePrincipal()` with no matching user
- `Principal.HasRole()` - true/false cases
- `Principal.HasAnyRole()` - various combinations

**What NOT to test (or why it's hard):**
- `RequireYubiKeyIfConfigured()` - requires FIDO2 hardware
- `RequireFIDO2Assertion()` - requires hardware integration

**How to test:**
- Mock environment variables
- Create test configs with various user/role setups
- Use table-driven tests for role checking

**Difficulty:** ⭐⭐ Medium (due to env var mocking)

---

### ⚠️ `internal/auth/fido2` - **Partially Testable**

**Why it needs tests:**
- Cryptographic verification is critical
- Signature verification can be tested without hardware

**What to test:**
- `verifyFIDO2Signature()` with valid signatures
- `verifyFIDO2Signature()` with invalid signatures
- `verifyFIDO2Signature()` with wrong key types
- Base64URL decoding edge cases

**What NOT to test:**
- `performAssertion()` - requires libfido2 integration (currently stubbed)
- `RequireFIDO2Assertion()` - requires hardware

**How to test:**
- Generate test ECDSA keys
- Create valid/invalid assertion structures
- Test signature verification logic

**Difficulty:** ⭐⭐⭐ Hard (cryptographic testing)

---

### ✅ `internal/clients/http` - **Highly Testable**

**Why it needs tests:**
- Network client - needs error handling verification
- URL construction logic
- Response parsing

**What to test:**
- `NewHTTPClient()` - initialization
- `Request()` with successful responses (various status codes)
- `Request()` with network errors
- `Request()` with context cancellation
- URL path joining (baseURL + path)

**How to test:**
- Use `net/http/httptest` to create test servers
- Test various HTTP status codes
- Test timeout scenarios
- Test context cancellation

**Difficulty:** ⭐ Easy

---

### ⚠️ `internal/clients/postgres` - **Testable with Setup**

**Why it needs tests:**
- Database client - critical for data operations
- Query execution and error handling

**What to test:**
- `NewPostgresClient()` with valid DSN
- `NewPostgresClient()` with invalid DSN
- `NewPostgresClient()` with unreachable database
- `RunScalarQuery()` with successful query
- `RunScalarQuery()` with query errors
- `RunScalarQuery()` with context cancellation
- Various return types (string, int, null)

**How to test:**
- Use testcontainers or in-memory Postgres (pgx supports this)
- Or use a mock database driver
- Test with real Postgres in CI

**Difficulty:** ⭐⭐ Medium (requires database setup)

---

### ✅ `internal/logging` - **Highly Testable**

**Why it needs tests:**
- Audit logging is critical for compliance
- SQLite operations need verification
- Data persistence and retrieval

**What to test:**
- `NewAuditLogger()` with valid path
- `NewAuditLogger()` with invalid path
- Schema creation (idempotent)
- `Log()` with various entry types
- `ReadRecent()` - ordering, limiting
- `ReadRecent()` with empty database
- `Close()` - resource cleanup
- WAL mode verification

**How to test:**
- Use `:memory:` SQLite database for fast tests
- Or use temporary files
- Test with various entry combinations

**Difficulty:** ⭐ Easy

---

### ✅ `internal/openapi` - **Testable with Mocks**

**Why it needs tests:**
- OpenAPI parsing is complex
- Operation generation logic needs verification
- Tag filtering and eligibility checks

**What to test:**
- `GenerateOperations()` with valid OpenAPI spec
- `GenerateOperations()` with invalid spec
- `GenerateOperations()` with network errors
- `operationEligible()` - tag filtering logic
- `hasRequiredRequestBody()` - various cases
- `buildLabel()` - summary vs fallback
- `sanitizePath()` - various path formats
- Operation ID generation (with/without prefix)

**How to test:**
- Use `net/http/httptest` to serve mock OpenAPI specs
- Create test OpenAPI documents
- Test various tag/operation combinations

**Difficulty:** ⭐⭐ Medium (OpenAPI spec complexity)

---

### ✅ `internal/tasks` - **Highly Testable**

**Why it needs tests:**
- Task execution is core functionality
- Error handling policies are complex
- Step execution logic

**What to test:**
- `NewRunner()` - initialization
- `Run()` with successful task (all steps succeed)
- `Run()` with `fail_fast` policy
- `Run()` with `best_effort` policy
- `Run()` with step-level error policies (`inherit`, `fail`, `warn`, `continue`)
- `runStep()` for each step type (http, postgres, sleep)
- `runStep()` with missing resources
- `runStep()` with context cancellation (sleep)
- `logStep()` and `logTask()` - audit logging
- `stepOnErrorFromTask()` - policy mapping

**How to test:**
- Mock HTTP and Postgres clients
- Use table-driven tests for error policies
- Test with various step configurations

**Difficulty:** ⭐⭐ Medium (complex state machine)

---

### ✅ `internal/tasks/template` - **Highly Testable**

**Why it needs tests:**
- Template rendering is pure function
- Template syntax errors need handling

**What to test:**
- `executeTemplate()` with valid template
- `executeTemplate()` with invalid template syntax
- `executeTemplate()` with missing fields
- `RenderSummary()` with empty template
- `RenderSummary()` with various step results
- Template variable access (nested structs)

**How to test:**
- Use test data structures
- Test various template syntaxes
- Verify output strings

**Difficulty:** ⭐ Easy

---

### ⚠️ `internal/ui` - **Difficult to Test**

**Why it's hard to test:**
- TUI code is inherently interactive
- Bubble Tea models are stateful
- Requires terminal simulation

**What could be tested:**
- Helper functions (`splitLines`, `operationsToItems`, `tasksToItems`)
- Filter logic (`filterType` application)
- Item conversion logic
- Message handling (without full TUI)

**What NOT to test:**
- Full TUI rendering (requires terminal)
- User interaction flows (requires integration tests)
- Visual layout

**How to test:**
- Unit test helper functions
- Test model state transitions with mock messages
- Integration tests with `tea.Program` in headless mode

**Difficulty:** ⭐⭐⭐ Hard (TUI testing is complex)

---

### ❌ `cmd/lazyadmin/main` - **Not Unit Testable**

**Why it's not testable:**
- `main()` function is the entry point
- Orchestrates all components
- Exits on errors

**What to do instead:**
- Integration tests (test the full application)
- Test via CLI execution
- Extract testable logic from `main()` if needed

**Difficulty:** ❌ Not unit testable (integration tests only)

---

## Testing Strategy Recommendations

### Priority 1: Critical Path (Add First)
1. **`internal/config`** - Config loading is foundational
2. **`internal/auth`** (testable parts) - Security-critical
3. **`internal/logging`** - Audit logging must be reliable
4. **`internal/tasks`** - Core functionality

### Priority 2: Important Components
5. **`internal/clients/http`** - Network operations
6. **`internal/tasks/template`** - Template rendering
7. **`internal/openapi`** - OpenAPI integration

### Priority 3: Complex but Valuable
8. **`internal/clients/postgres`** - Requires DB setup
9. **`internal/auth/fido2`** - Cryptographic testing

### Priority 4: Lower Priority
10. **`internal/ui`** - TUI testing is complex, focus on helpers
11. **`cmd/lazyadmin/main`** - Integration tests only

---

## Test Utilities Needed

Consider creating:
- `testutil/` package with:
  - Test config generators
  - Mock HTTP servers
  - Test database helpers
  - Test principal/role builders

---

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package
go test ./internal/config

# Run with verbose output
go test -v ./internal/config
```

---

## Coverage Goals

- **Minimum:** 60% overall coverage
- **Target:** 80% overall coverage
- **Critical packages:** 90%+ coverage (config, auth, logging, tasks)

