//go:build libfido2
// +build libfido2

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
	"math/big"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/keys-pub/go-libfido2"
	"github.com/you/lazyadmin/internal/config"
)

// FIDO2 authentication implementation.
// The performAssertion function requires libfido2 integration via a Go wrapper
// such as github.com/keys-pub/go-libfido2.

var (
	ErrNoYubiCreds        = errors.New("user has no configured YubiKey credentials")
	ErrAssertionFailed    = errors.New("YubiKey assertion failed")
	ErrNoMatchingCredID   = errors.New("assertion credential ID did not match any configured credential")
	ErrNoDeviceFound      = errors.New("no FIDO2 device found")
	ErrRegistrationFailed = errors.New("FIDO2 registration failed")
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
func performAssertion(ctx context.Context, rpID string, challenge []byte, allowCredentialID string) (*AssertionResult, error) {
	locations, err := libfido2.DeviceLocations()
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	if len(locations) == 0 {
		return nil, ErrNoDeviceFound
	}

	device, err := libfido2.NewDevice(locations[0].Path)
	if err != nil {
		return nil, fmt.Errorf("open device: %w", err)
	}
	// Device is automatically closed when it goes out of scope

	credIDBytes, err := base64.RawURLEncoding.DecodeString(allowCredentialID)
	if err != nil {
		return nil, fmt.Errorf("decode credential ID: %w", err)
	}

	clientHash := sha256.Sum256(challenge)
	credentialIDs := [][]byte{credIDBytes}

	assertion, err := device.Assertion(rpID, clientHash[:], credentialIDs, "", nil)
	if err != nil {
		return nil, fmt.Errorf("device assertion: %w", err)
	}

	if assertion == nil {
		return nil, ErrAssertionFailed
	}

	credIDB64 := base64.RawURLEncoding.EncodeToString(assertion.CredentialID)

	// Extract authData from CBOR-encoded authData
	authData := assertion.AuthDataCBOR
	if len(authData) == 0 {
		return nil, fmt.Errorf("empty auth data")
	}

	return &AssertionResult{
		CredentialID: credIDB64,
		Signature:    assertion.Sig,
		AuthData:     authData,
	}, nil
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

// RegistrationResult represents a FIDO2 registration response.
type RegistrationResult struct {
	CredentialID string
	PublicKey    string // Base64URL-encoded SPKI
}

// RegisterFIDO2Credential registers a new FIDO2 credential on a YubiKey device.
// Returns the credential ID and public key in base64url format.
func RegisterFIDO2Credential(ctx context.Context, rpID string, rpName string, userName string, userID []byte) (*RegistrationResult, error) {
	locations, err := libfido2.DeviceLocations()
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	if len(locations) == 0 {
		return nil, ErrNoDeviceFound
	}

	device, err := libfido2.NewDevice(locations[0].Path)
	if err != nil {
		return nil, fmt.Errorf("open device: %w", err)
	}
	// Device is automatically closed when it goes out of scope

	// Generate random challenge
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("generate challenge: %w", err)
	}

	clientHash := sha256.Sum256(challenge)

	// Create user entity
	user := libfido2.User{
		ID:          userID,
		Name:        userName,
		DisplayName: userName,
	}

	// Create relying party
	rp := libfido2.RelyingParty{
		ID:   rpID,
		Name: rpName,
	}

	// Register credential
	attestation, err := device.MakeCredential(clientHash[:], rp, user, libfido2.ES256, "", nil)
	if err != nil {
		return nil, fmt.Errorf("make credential: %w", err)
	}

	// Extract public key from COSE format and convert to SPKI
	// attestation.PubKey is in COSE format, we need to parse it and convert to SPKI
	pubKey, err := parseCOSEPublicKey(attestation.PubKey)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	pubKeySPKI, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}

	credIDB64 := base64.RawURLEncoding.EncodeToString(attestation.CredentialID)
	pubKeyB64 := base64.RawURLEncoding.EncodeToString(pubKeySPKI)

	return &RegistrationResult{
		CredentialID: credIDB64,
		PublicKey:    pubKeyB64,
	}, nil
}

// parseCOSEPublicKey parses a COSE-encoded public key and returns an ECDSA public key.
// COSE format for ES256: map with kty=2 (EC2), crv=-7 (P-256), x and y coordinates.
func parseCOSEPublicKey(coseKey []byte) (*ecdsa.PublicKey, error) {
	// Try to parse as SPKI first (in case the library already converts it)
	pub, err := x509.ParsePKIXPublicKey(coseKey)
	if err == nil {
		ecdsaPub, ok := pub.(*ecdsa.PublicKey)
		if ok && ecdsaPub.Curve == elliptic.P256() {
			return ecdsaPub, nil
		}
	}

	// Parse COSE format (CBOR map)
	var coseMap map[interface{}]interface{}
	if err := cbor.Unmarshal(coseKey, &coseMap); err != nil {
		return nil, fmt.Errorf("unmarshal COSE key: %w", err)
	}

	// Extract kty (key type) - should be 2 for EC2
	kty, ok := coseMap[int64(1)].(int64)
	if !ok || kty != 2 {
		return nil, fmt.Errorf("invalid key type: expected EC2 (2), got %v", kty)
	}

	// Extract crv (curve) - should be -7 for P-256
	crv, ok := coseMap[int64(-1)].(int64)
	if !ok || crv != -7 {
		return nil, fmt.Errorf("invalid curve: expected P-256 (-7), got %v", crv)
	}

	// Extract x coordinate
	xBytes, ok := coseMap[int64(-2)].([]byte)
	if !ok || len(xBytes) != 32 {
		return nil, fmt.Errorf("invalid x coordinate: expected 32 bytes, got %d", len(xBytes))
	}

	// Extract y coordinate
	yBytes, ok := coseMap[int64(-3)].([]byte)
	if !ok || len(yBytes) != 32 {
		return nil, fmt.Errorf("invalid y coordinate: expected 32 bytes, got %d", len(yBytes))
	}

	// Convert to big integers
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	// Create ECDSA public key
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	return pubKey, nil
}

// ContextWithTimeout returns a context with a 30-second timeout for FIDO2 operations.
func ContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
