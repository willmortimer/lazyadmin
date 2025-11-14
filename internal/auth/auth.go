package auth

import (
	"context"
	"errors"
	"os"
	"os/user"

	"github.com/you/lazyadmin/internal/config"
	"github.com/you/lazyadmin/internal/users"
)

type Principal struct {
	ConfigUser *config.User
	DBUser     *users.User
	SSHUser    string
}

var (
	ErrNoMatchingUser = errors.New("no matching lazyadmin user for current SSH user")
)

func CurrentSSHUser() string {
	if v := os.Getenv("SSH_USER"); v != "" {
		return v
	}
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	u, err := user.Current()
	if err == nil && u.Username != "" {
		return u.Username
	}
	return "unknown"
}

// ResolvePrincipal resolves the principal from config and optionally from SQLite user store.
// Config users are checked first (for hardcoded admin), then SQLite users.
func ResolvePrincipal(cfg *config.Config, userStore *users.Store) (*Principal, error) {
	sshUser := CurrentSSHUser()

	// First, check config users (for hardcoded admin)
	for i := range cfg.Users {
		u := &cfg.Users[i]
		for _, su := range u.SSHUsers {
			if su == sshUser {
				return &Principal{
					ConfigUser: u,
					SSHUser:    sshUser,
				}, nil
			}
		}
	}

	// Then check SQLite users if store is provided
	if userStore != nil {
		ctx := context.Background()
		dbUser, err := userStore.FindUserBySSHUser(ctx, sshUser)
		if err == nil {
			// Convert DB user to config user format for compatibility
			configUser := &config.User{
				ID:           dbUser.ID,
				SSHUsers:     dbUser.SSHUsers,
				Roles:        dbUser.Roles,
				YubiKeyCreds: []config.YubiKeyCredential{},
			}

			// Load credentials from DB
			creds, err := userStore.GetCredentials(ctx, dbUser.ID)
			if err == nil {
				for _, cred := range creds {
					configUser.YubiKeyCreds = append(configUser.YubiKeyCreds, config.YubiKeyCredential{
						RPID:         cred.RPID,
						CredentialID: cred.CredentialID,
						PublicKey:    cred.PublicKey,
					})
				}
			}

			return &Principal{
				DBUser:     dbUser,
				SSHUser:    sshUser,
				ConfigUser: configUser,
			}, nil
		}
	}

	return nil, ErrNoMatchingUser
}

func (p *Principal) HasRole(role string) bool {
	if p.ConfigUser != nil {
		for _, r := range p.ConfigUser.Roles {
			if r == role {
				return true
			}
		}
	}
	if p.DBUser != nil {
		for _, r := range p.DBUser.Roles {
			if r == role {
				return true
			}
		}
	}
	return false
}

// IsAdmin returns true if the principal has admin or owner role.
func (p *Principal) IsAdmin() bool {
	return p.HasRole("admin") || p.HasRole("owner")
}

func (p *Principal) HasAnyRole(roles []string) bool {
	for _, role := range roles {
		if p.HasRole(role) {
			return true
		}
	}
	return false
}

func RequireYubiKeyIfConfigured(cfg *config.Config, p *Principal) error {
	if !cfg.Auth.RequireYubiKey {
		return nil
	}

	if p.ConfigUser == nil {
		return ErrNoYubiCreds
	}

	ctx, cancel := ContextWithTimeout()
	defer cancel()

	return RequireFIDO2Assertion(ctx, p.ConfigUser)
}
