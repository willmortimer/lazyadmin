# Next Steps and Project Status

## Current Status: v0.1.0 (Experimental)

lazyadmin is functional but not production-ready. The core features are implemented, but several areas need work before production use.

## Immediate Next Steps

### 1. FIDO2 Implementation
**Priority: High**

- **Current State**: Framework exists, but `performAssertion` is stubbed
- **Required**: Integrate libfido2 via Go wrapper (e.g., `github.com/keys-pub/go-libfido2`)
- **Action Items**:
  - Install libfido2 system library
  - Implement device communication in `internal/auth/fido2.go`
  - Test with actual YubiKey devices
  - Add credential registration tool/process

### 2. Testing
**Priority: High**

- **Current State**: No tests written
- **Required**: Unit tests for core components
- **Action Items**:
  - Add tests for config loading and validation
  - Add tests for auth/principal resolution
  - Add tests for task execution and error policies
  - Add integration tests for TUI flows
  - Add tests for OpenAPI operation generation

### 3. Error Handling
**Priority: Medium**

- **Current State**: Basic error handling, some errors may not be user-friendly
- **Required**: Better error messages and recovery
- **Action Items**:
  - Improve error messages throughout
  - Add retry logic for transient failures
  - Better handling of resource connection failures
  - Graceful degradation when resources unavailable

### 4. Configuration Validation
**Priority: Medium**

- **Current State**: Basic validation, some invalid configs may pass
- **Required**: Comprehensive validation with clear error messages
- **Action Items**:
  - Validate all resource references at startup
  - Validate role references
  - Validate task step resource references
  - Provide helpful error messages with line numbers

### 5. TUI Enhancements
**Priority: Medium**

- **Current State**: Functional but basic
- **Required**: Better UX and visual feedback
- **Action Items**:
  - Add progress indicators for long-running operations
  - Show step-by-step progress for tasks
  - Visual highlighting for high-risk tasks
  - Better error display formatting
  - Search/filter within lists

## Known Weaknesses

### Security

1. **FIDO2 Not Implemented**
   - Currently stubbed, provides no actual security
   - Must be implemented before production use

2. **No Input Validation**
   - SQL queries come from config (trusted), but no validation
   - HTTP paths are from config, but no sanitization
   - Consider adding validation layer

3. **Single Config File**
   - All users/roles in one file
   - No multi-tenant isolation
   - Config file access = full access

4. **No Rate Limiting**
   - Operations can be executed rapidly
   - No protection against accidental loops
   - Consider adding rate limits per user/operation

### Functionality

1. **Limited Resource Types**
   - Only HTTP and Postgres supported
   - Redis mentioned but not implemented
   - No generic resource abstraction

2. **No Operation Parameters**
   - Operations are static (no runtime parameters)
   - Cannot prompt for values before execution
   - Limits flexibility

3. **No Task Dependencies**
   - Tasks cannot depend on other tasks
   - No conditional execution
   - No loops or branching

4. **Basic Audit Logging**
   - SQLite only, no rotation
   - No log aggregation or analysis
   - No alerting on failures

### Operational

1. **No Health Checks**
   - Container has no health check endpoint
   - Cannot monitor application health
   - Consider adding `/health` endpoint

2. **No Metrics**
   - No Prometheus/metrics export
   - No performance monitoring
   - No operation timing

3. **No Configuration Hot Reload**
   - Config changes require restart
   - No way to update config without downtime
   - Consider config watching/reload

4. **Limited Documentation**
   - Missing deployment guide
   - No troubleshooting guide
   - No examples for common scenarios

## Future Enhancements

### Short Term (v0.2.0)

- [ ] Implement FIDO2 authentication
- [ ] Add comprehensive test suite
- [ ] Improve error handling and messages
- [ ] Add Redis resource support
- [ ] Add operation parameters/prompts
- [ ] Add health check endpoint

### Medium Term (v0.3.0)

- [ ] Task dependencies and conditional execution
- [ ] Configuration hot reload
- [ ] Metrics export (Prometheus)
- [ ] Log rotation and management
- [ ] Rate limiting
- [ ] Better TUI with progress indicators

### Long Term (v1.0.0)

- [ ] Multi-tenant support
- [ ] Web UI option
- [ ] Plugin system for custom resource types
- [ ] Distributed audit logging
- [ ] Operation scheduling/cron
- [ ] Configuration management UI

## Production Readiness Checklist

Before considering lazyadmin production-ready:

- [ ] FIDO2 authentication fully implemented and tested
- [ ] Comprehensive test coverage (>80%)
- [ ] Security audit completed
- [ ] Performance testing completed
- [ ] Documentation complete
- [ ] Deployment guide written
- [ ] Monitoring and alerting configured
- [ ] Backup and recovery procedures documented
- [ ] Incident response plan created

## Contributing

When contributing:

1. **Update SPEC.md first** if behavior changes
2. **Add tests** for new functionality
3. **Update documentation** as needed
4. **Follow existing code style**
5. **Reference SPEC sections** in code comments where behavior is subtle

## Questions to Resolve

1. **Credential Management**: How should YubiKey credentials be registered? Manual process or tool?
2. **Config Management**: How should config be managed in production? GitOps? Config service?
3. **Multi-Environment**: How to handle dev/staging/prod configs? Separate files? Environment variables?
4. **Audit Log Retention**: How long to keep logs? Rotation policy?
5. **High-Risk Operations**: Should high-risk tasks require additional confirmation beyond YubiKey?

