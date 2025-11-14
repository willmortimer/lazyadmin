# FIDO2 YubiKey Setup Guide

This guide explains how to configure FIDO2 YubiKey authentication for lazyadmin.

## Overview

FIDO2 authentication provides hardware-backed security using YubiKey devices. When enabled, users must touch their YubiKey to authenticate before accessing the lazyadmin TUI.

## Current Status

✅ **FIDO2 is fully implemented:**

- Configuration structure exists
- Signature verification is implemented
- Hardware integration (`performAssertion`) is complete
- Registration system (`RegisterFIDO2Credential`) is complete
- CLI registration utility (`lazyadmin-register`) is available
- SQLite-based user management system is available

## Configuration

### 1. Enable FIDO2 in Config

Edit `config/lazyadmin.yaml`:

```yaml
auth:
  require_yubikey: true # Set to true to enable
  yubikey_mode: fido2
```

### 2. Configure User Credentials

Add YubiKey credentials to user entries:

```yaml
users:
  - id: will
    ssh_users: ["will"]
    roles: ["owner", "admin"]
    yubikey_credentials:
      - rp_id: "lazyadmin.local" # Relying Party ID
        credential_id: "BASE64URL_CRED_ID" # Base64URL-encoded credential ID
        public_key: "BASE64URL_PUBLIC_KEY" # Base64URL-encoded public key (SPKI format)
```

### Credential Fields

- **`rp_id`**: Relying Party ID (e.g., domain name or identifier)
- **`credential_id`**: Base64URL-encoded credential ID from YubiKey registration
- **`public_key`**: Base64URL-encoded SubjectPublicKeyInfo (SPKI) for P-256 ECDSA key

## Credential Registration Process

### CLI Registration Tool

Use the `lazyadmin-register` command-line utility to register YubiKey credentials:

```bash
# Build the registration tool
go build ./cmd/lazyadmin-register

# Register a YubiKey (may require sudo for device access)
sudo ./lazyadmin-register \
  --rp-id lazyadmin.local \
  --rp-name lazyadmin \
  --user-name will \
  --user-id will \
  --output yaml
```

The tool will:

1. Detect connected YubiKey devices
2. Prompt you to touch your YubiKey
3. Generate a new FIDO2 credential
4. Output the credential ID and public key in YAML format

### Registration Options

- `--rp-id`: Relying Party ID (default: "lazyadmin.local")
- `--rp-name`: Relying Party Name (default: "lazyadmin")
- `--user-name`: User name for the credential (defaults to current username)
- `--user-id`: User ID for the credential (defaults to current username)
- `--output`: Output format - "yaml" or "json" (default: "yaml")

### Adding Credentials to Config

After registration, add the output to your `config/lazyadmin.yaml`:

```yaml
users:
  - id: will
    ssh_users: ["will"]
    roles: ["owner", "admin"]
    yubikey_credentials:
      - rp_id: "lazyadmin.local"
        credential_id: "dGVzdF9jcmVkZW50aWFsX2lkX2Jhc2U2NHVybA"
        public_key: "dGVzdF9wdWJsaWNfa2V5X2Jhc2U2NHVybF9zcGtpX2Zvcm1hdA"
```

### SQLite User Management

Users can also be managed via SQLite database (same database as audit logs). The default config user (hardcoded in YAML) has admin privileges and can register additional users through the TUI (when implemented) or directly via SQLite.

**Note:** The default config user remains hardcoded and always has admin/owner privileges. Additional users can be added to SQLite for dynamic user management.

## Implementation Status

### ✅ Completed

- Configuration structure (`YubiKeyCredential` type)
- Signature verification (`verifyFIDO2Signature`)
- Challenge generation
- Credential matching logic
- Error handling
- **`performAssertion()` function** - Full libfido2 integration
- **`RegisterFIDO2Credential()` function** - Complete registration implementation
- **CLI registration tool** (`cmd/lazyadmin-register`)
- **SQLite user management** (`internal/users` package)
- **Hardware device communication** - Full YubiKey support
- **COSE public key parsing** - Converts COSE format to SPKI

### 🔄 In Progress

- TUI user management interface for admin users to register new users

## Required Dependencies

To complete FIDO2 implementation, you'll need:

1. **libfido2** system library

   ```bash
   # Ubuntu/Debian
   sudo apt-get install libfido2-dev

   # macOS
   brew install libfido2
   ```

2. **Go wrapper** for libfido2
   - Option: `github.com/keys-pub/go-libfido2`
   - Or implement custom wrapper using cgo

## Testing Without Hardware

For development/testing, you can:

1. Set `require_yubikey: false` in config
2. Leave `yubikey_credentials` empty or with placeholder values
3. Authentication will be skipped

## Security Considerations

### Credential Storage

- Credentials are stored in plain text in `config/lazyadmin.yaml`
- Consider encrypting credentials at rest for production
- Use secure configuration management (e.g., HashiCorp Vault, AWS Secrets Manager)

### RP ID

- RP ID should match your domain or application identifier
- Use consistent RP IDs across environments
- Don't use localhost or IP addresses (use domain names)

### Credential Rotation

- Rotate credentials periodically
- Revoke immediately if YubiKey is lost/stolen
- Support multiple credentials per user for backup keys

## Example Configuration

```yaml
auth:
  require_yubikey: true
  yubikey_mode: fido2

users:
  - id: admin
    ssh_users: ["admin"]
    roles: ["owner"]
    yubikey_credentials:
      - rp_id: "lazyadmin.prod.example.com"
        credential_id: "dGVzdF9jcmVkZW50aWFsX2lkX2Jhc2U2NHVybA"
        public_key: "dGVzdF9wdWJsaWNfa2V5X2Jhc2U2NHVybF9zcGtpX2Zvcm1hdA"
      - rp_id: "lazyadmin.prod.example.com" # Backup key
        credential_id: "YmFja3VwX2NyZWRlbnRpYWxfaWRfYmFzZTY0dXJs"
        public_key: "YmFja3VwX3B1YmxpY19rZXlfYmFzZTY0dXJsX3Nwa2lfZm9ybWF0"

  - id: operator
    ssh_users: ["operator"]
    roles: ["admin"]
    yubikey_credentials:
      - rp_id: "lazyadmin.prod.example.com"
        credential_id: "b3BlcmF0b3JfY3JlZGVudGlhbF9pZF9iYXNlNjR1cmw"
        public_key: "b3BlcmF0b3JfcHVibGljX2tleV9iYXNlNjR1cmxfc3BraV9mb3JtYXQ"
```

## Troubleshooting

### "FIDO2 not implemented" Error

This means `performAssertion()` is still stubbed. You need to:

1. Install libfido2
2. Integrate libfido2 Go wrapper
3. Implement `performAssertion()` function

### "no matching lazyadmin user" Error

- Check that `SSH_USER` or `USER` matches an entry in `ssh_users[]`
- Verify the user exists in `config/lazyadmin.yaml`

### "user has no configured YubiKey credentials"

- User exists but has no `yubikey_credentials` entries
- Add credentials or set `require_yubikey: false` for that user

## Next Steps

See [`NEXT_STEPS.md`](../NEXT_STEPS.md) for implementation roadmap:

1. Install libfido2 system library
2. Integrate Go wrapper (e.g., `github.com/keys-pub/go-libfido2`)
3. Implement `performAssertion()` in `internal/auth/fido2.go`
4. Create registration tool/command
5. Add tests with mock FIDO2 devices
6. Document registration process
