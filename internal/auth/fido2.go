package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/you/lazyadmin/internal/config"
)

// FIDO2 authentication implementation.
// The performAssertion function requires libfido2 integration via a Go wrapper
// such as github.com/keys-pub/go-libfido2.

var (
	ErrNoYubiCreds      = errors.New("user has no configured YubiKey credentials")
	ErrAssertionFailed  = errors.New("YubiKey assertion failed")
	ErrNoMatchingCredID = errors.New("assertion credential ID did not match any configured credential")
)

// RequireFIDO2Assertion prompts the user to touch a key and verifies an assertion
// against the configured YubiKey credentials.
func RequireFIDO2Assertion(ctx context.Context, user *config.User) error {
	if len(user.YubiKeyCreds) == 0 {
		return ErrNoYubiCreds
	}

	cred := user.YubiKeyCreds[0]

	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return fmt.Errorf("challenge: %w", err)
	}

	fmt.Println("YubiKey FIDO2 authentication required")
	fmt.Printf("RP ID: %s\n", cred.RPID)
	fmt.Printf("User: %s\n", user.ID)
	fmt.Println("Please touch your YubiKey...")

	assertion, err := performAssertion(ctx, cred.RPID, challenge, cred.CredentialID)
	if err != nil {
		return fmt.Errorf("fido2 assertion: %w", err)
	}

	if assertion.CredentialID != cred.CredentialID {
		return ErrNoMatchingCredID
	}

	if err := verifyFIDO2Signature(assertion, cred.PublicKey, challenge); err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}

	fmt.Println("YubiKey assertion verified")
	return nil
}

// AssertionResult represents a FIDO2 assertion response.
type AssertionResult struct {
	CredentialID string
	Signature    []byte
	AuthData     []byte
}

// performAssertion communicates with a FIDO2 device to obtain an assertion.
// This function must be implemented with libfido2 or a similar library.
func performAssertion(ctx context.Context, rpID string, challenge []byte, allowCredentialID string) (*AssertionResult, error) {
	_ = rpID
	_ = challenge
	_ = allowCredentialID
	return nil, errors.New("FIDO2 not implemented: libfido2 integration required")
}

// verifyFIDO2Signature verifies the assertion signature against the stored public key.
// Expects base64url-encoded SPKI (SubjectPublicKeyInfo) for a P-256 ECDSA key,
// and ASN.1 DER-encoded signature. The signed data is SHA256(authData || SHA256(challenge)).
func verifyFIDO2Signature(assertion *AssertionResult, publicKeyB64URL string, challenge []byte) error {
	pubBytes, err := base64.RawURLEncoding.DecodeString(publicKeyB64URL)
	if err != nil {
		return fmt.Errorf("decode public key: %w", err)
	}

	pub, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok || ecdsaPub.Curve != elliptic.P256() {
		return fmt.Errorf("public key is not P-256 ECDSA")
	}

	clientHash := sha256.Sum256(challenge)
	msg := make([]byte, 0, len(assertion.AuthData)+len(clientHash))
	msg = append(msg, assertion.AuthData...)
	msg = append(msg, clientHash[:]...)
	digest := sha256.Sum256(msg)

	if !ecdsa.VerifyASN1(ecdsaPub, digest[:], assertion.Signature) {
		return ErrAssertionFailed
	}

	return nil
}

// ContextWithTimeout returns a context with a 30-second timeout for FIDO2 operations.
func ContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

