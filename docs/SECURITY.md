# Security Model

This document describes the security assumptions, guarantees, and threat model for lazyadmin.

## Trust Boundaries

### Container Boundary

lazyadmin runs in a container with:

- Read-only root filesystem
- No new privileges (`no-new-privileges:true`)
- All capabilities dropped (`cap_drop: ALL`)
- Minimal base image (distroless)

**Assumption**: The container runtime and host are trusted. Container escape is outside the threat model.

### Network Boundary

lazyadmin communicates with:

- HTTP backends over private Docker networks
- PostgreSQL databases over private networks or trusted connections
- No network listeners (TUI only, no remote access)

**Assumption**: Network traffic occurs over trusted networks (Docker bridge, Tailscale, VPN). Network interception is outside the threat model for private networks.

### Configuration Boundary

Configuration is:

- Loaded from a single YAML file
- Mounted read-only in container
- Validated at startup

**Assumption**: Configuration file is trusted and maintained by a privileged operator. Configuration tampering is outside the threat model.

### Identity Boundary

User identity is derived from:

- SSH/Unix username from environment
- Mapping to configured users in YAML
- FIDO2 credentials for additional authentication

**Assumption**: SSH access to the host/container is already strongly authenticated. SSH key management is outside the threat model.

## Security Guarantees

### No Shell Execution

lazyadmin does NOT execute:

- Arbitrary shell commands
- User-provided scripts
- Untrusted code

All operations are typed (HTTP requests, SQL queries) with no code execution.

### No Network Listeners

lazyadmin does NOT:

- Listen on network ports
- Accept remote connections
- Expose network services

The TUI is local-only, requiring direct access to the container or host.

### Read-Only Configuration

Configuration is:

- Mounted read-only in container
- Loaded once at startup
- Not modified at runtime

### Append-Only Audit Log

Audit logs are:

- Stored in SQLite with WAL mode
- Schema prevents UPDATE or DELETE operations
- Append-only at the database level

### Role-Based Access Control

Operations and tasks are:

- Filtered by user roles at TUI level
- Validated before execution
- Logged with user identity

Users can only see and execute operations/tasks allowed by their roles.

## Security Assumptions

### FIDO2 Implementation

FIDO2 authentication assumes:

- libfido2 library is correctly implemented
- YubiKey device is genuine and not compromised
- Credential public keys are correctly stored in configuration
- Challenge-response protocol is correctly implemented

**Current Status**: FIDO2 assertion is stubbed and requires libfido2 integration.

### SQL Injection Prevention

PostgreSQL operations:

- Use parameterized queries where applicable
- Execute scalar queries only
- Do not accept user input directly in queries

**Note**: Query strings are defined in configuration, not user input. Configuration is trusted.

### HTTP Request Safety

HTTP operations:

- Execute predefined methods and paths
- Do not accept user-provided URLs or headers
- Use configured base URLs only

**Note**: Operation definitions are in configuration, not user input.

## Threat Model

### In Scope

lazyadmin protects against:

- Unauthorized operation execution (via RBAC)
- Audit log tampering (via append-only schema)
- Configuration modification at runtime (via read-only mount)

### Out of Scope

lazyadmin does NOT protect against:

- Root attacker on host (container escape)
- Network interception on trusted networks
- Configuration file tampering before startup
- SSH key compromise
- FIDO2 device compromise
- Malicious configuration content
- Multi-tenant isolation (single config, single deployment)

## Security Considerations

### High-Risk Tasks

Tasks with `risk_level: "high"` or `require_yubikey: true`:

- Currently marked in TUI but do not require additional authentication
- Future: May require additional FIDO2 assertion before execution

### Audit Log Integrity

Audit logs provide:

- Immutable record of all operations
- User attribution for accountability
- Success/failure tracking

Logs should be:
- Backed up regularly
- Monitored for anomalies
- Protected from deletion

### Configuration Management

Configuration should be:

- Version controlled
- Reviewed before deployment
- Validated in CI/CD pipelines
- Rotated when credentials change

### FIDO2 Credential Management

YubiKey credentials should be:

- Generated securely during registration
- Stored encrypted at rest (if possible)
- Rotated periodically
- Revoked immediately if compromised

## Best Practices

### Deployment

1. Use read-only configuration mounts
2. Run in minimal container with dropped capabilities
3. Use private networks for backend communication
4. Enable FIDO2 authentication in production
5. Monitor audit logs for suspicious activity

### Configuration

1. Use least-privilege role assignments
2. Limit high-risk operations to trusted roles
3. Review operation definitions regularly
4. Use environment variables for sensitive DSNs
5. Keep configuration in version control

### Operations

1. Prefer read-only operations where possible
2. Use tasks for complex workflows with error handling
3. Review audit logs regularly
4. Rotate credentials periodically

## Reporting Security Issues

Security issues should be reported privately to the maintainers. Do not open public issues for security vulnerabilities.

