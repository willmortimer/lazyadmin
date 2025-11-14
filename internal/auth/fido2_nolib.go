//go:build !libfido2
// +build !libfido2

package auth

import (
	"context"
	"errors"
	"time"

	"github.com/you/lazyadmin/internal/config"
)

var (
	ErrNoYubiCreds        = errors.New("user has no configured YubiKey credentials")
	ErrAssertionFailed    = errors.New("YubiKey assertion failed")
	ErrNoMatchingCredID   = errors.New("assertion credential ID did not match any configured credential")
	ErrNoDeviceFound      = errors.New("no FIDO2 device found")
	ErrRegistrationFailed = errors.New("FIDO2 registration failed")
	ErrFIDO2NotAvailable  = errors.New("FIDO2 support not available: libfido2-dev not installed. Install with: sudo apt-get install libfido2-dev")
)

// RequireFIDO2Assertion prompts the user to touch a key and verifies an assertion
// against the configured YubiKey credentials.
func RequireFIDO2Assertion(ctx context.Context, user *config.User) error {
	if len(user.YubiKeyCreds) == 0 {
		return ErrNoYubiCreds
	}
	return ErrFIDO2NotAvailable
}

// AssertionResult represents a FIDO2 assertion response.
type AssertionResult struct {
	CredentialID string
	Signature    []byte
	AuthData     []byte
}

// RegistrationResult represents a FIDO2 registration response.
type RegistrationResult struct {
	CredentialID string
	PublicKey    string // Base64URL-encoded SPKI
}

// RegisterFIDO2Credential registers a new FIDO2 credential on a YubiKey device.
// Returns the credential ID and public key in base64url format.
func RegisterFIDO2Credential(ctx context.Context, rpID string, rpName string, userName string, userID []byte) (*RegistrationResult, error) {
	return nil, ErrFIDO2NotAvailable
}

// ContextWithTimeout returns a context with a 30-second timeout for FIDO2 operations.
func ContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
