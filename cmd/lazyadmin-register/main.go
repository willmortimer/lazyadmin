package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os/user"
	"time"

	"github.com/you/lazyadmin/internal/auth"
)

func main() {
	var (
		rpID     = flag.String("rp-id", "lazyadmin.local", "Relying Party ID")
		rpName   = flag.String("rp-name", "lazyadmin", "Relying Party Name")
		userName = flag.String("user-name", "", "User name (defaults to current user)")
		userID   = flag.String("user-id", "", "User ID (defaults to current username)")
		output   = flag.String("output", "yaml", "Output format: yaml or json")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Determine user name and ID
	if *userName == "" {
		u, err := user.Current()
		if err != nil {
			log.Fatalf("get current user: %v", err)
		}
		*userName = u.Username
	}
	if *userID == "" {
		u, err := user.Current()
		if err != nil {
			log.Fatalf("get current user: %v", err)
		}
		*userID = u.Username
	}

	// Generate user ID bytes (8 bytes)
	userIDBytes := make([]byte, 8)
	if _, err := rand.Read(userIDBytes); err != nil {
		log.Fatalf("generate user ID: %v", err)
	}

	fmt.Printf("FIDO2 YubiKey Registration\n")
	fmt.Printf("==========================\n")
	fmt.Printf("RP ID: %s\n", *rpID)
	fmt.Printf("RP Name: %s\n", *rpName)
	fmt.Printf("User Name: %s\n", *userName)
	fmt.Printf("User ID: %s\n", *userID)
	fmt.Printf("\nPlease touch your YubiKey...\n")

	// Register credential
	result, err := auth.RegisterFIDO2Credential(ctx, *rpID, *rpName, *userName, userIDBytes)
	if err != nil {
		log.Fatalf("registration failed: %v", err)
	}

	fmt.Printf("\n✓ Registration successful!\n\n")

	// Output credentials
	switch *output {
	case "yaml":
		fmt.Printf("Add this to your config:\n\n")
		fmt.Printf("yubikey_credentials:\n")
		fmt.Printf("  - rp_id: %q\n", *rpID)
		fmt.Printf("    credential_id: %q\n", result.CredentialID)
		fmt.Printf("    public_key: %q\n", result.PublicKey)
	case "json":
		fmt.Printf("{\n")
		fmt.Printf("  \"rp_id\": %q,\n", *rpID)
		fmt.Printf("  \"credential_id\": %q,\n", result.CredentialID)
		fmt.Printf("  \"public_key\": %q\n", result.PublicKey)
		fmt.Printf("}\n")
	default:
		fmt.Printf("Credential ID: %s\n", result.CredentialID)
		fmt.Printf("Public Key: %s\n", result.PublicKey)
	}
}
