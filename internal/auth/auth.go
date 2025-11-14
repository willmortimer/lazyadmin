package auth

import (
	"errors"
	"os"
	"os/user"

	"github.com/you/lazyadmin/internal/config"
)

type Principal struct {
	ConfigUser *config.User
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

func ResolvePrincipal(cfg *config.Config) (*Principal, error) {
	sshUser := CurrentSSHUser()

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
	return nil, ErrNoMatchingUser
}

func (p *Principal) HasRole(role string) bool {
	for _, r := range p.ConfigUser.Roles {
		if r == role {
			return true
		}
	}
	return false
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

	ctx, cancel := ContextWithTimeout()
	defer cancel()

	return RequireFIDO2Assertion(ctx, p.ConfigUser)
}

