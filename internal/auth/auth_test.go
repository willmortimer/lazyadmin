package auth

import (
	"os"
	"testing"

	"github.com/you/lazyadmin/internal/config"
)

func TestCurrentSSHUser(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func()
		want     string
	}{
		{
			name: "SSH_USER set",
			setupEnv: func() {
				os.Setenv("SSH_USER", "alice")
				os.Unsetenv("USER")
			},
			want: "alice",
		},
		{
			name: "USER set when SSH_USER not set",
			setupEnv: func() {
				os.Unsetenv("SSH_USER")
				os.Setenv("USER", "bob")
			},
			want: "bob",
		},
		{
			name: "SSH_USER takes precedence over USER",
			setupEnv: func() {
				os.Setenv("SSH_USER", "alice")
				os.Setenv("USER", "bob")
			},
			want: "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			oldSSHUser := os.Getenv("SSH_USER")
			oldUser := os.Getenv("USER")

			// Setup test environment
			tt.setupEnv()

			// Restore original values after test
			defer func() {
				if oldSSHUser == "" {
					os.Unsetenv("SSH_USER")
				} else {
					os.Setenv("SSH_USER", oldSSHUser)
				}
				if oldUser == "" {
					os.Unsetenv("USER")
				} else {
					os.Setenv("USER", oldUser)
				}
			}()

			got := CurrentSSHUser()
			if got != tt.want {
				t.Errorf("CurrentSSHUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvePrincipal(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		sshUser  string
		wantErr  bool
		validate func(*testing.T, *Principal)
	}{
		{
			name: "matching SSH user",
			cfg: &config.Config{
				Users: []config.User{
					{
						ID:       "alice",
						SSHUsers: []string{"alice", "alice-dev"},
						Roles:    []string{"admin"},
					},
				},
			},
			sshUser: "alice",
			wantErr: false,
			validate: func(t *testing.T, p *Principal) {
				if p.SSHUser != "alice" {
					t.Errorf("Principal.SSHUser = %q, want %q", p.SSHUser, "alice")
				}
				if p.ConfigUser.ID != "alice" {
					t.Errorf("Principal.ConfigUser.ID = %q, want %q", p.ConfigUser.ID, "alice")
				}
			},
		},
		{
			name: "matching alternate SSH user",
			cfg: &config.Config{
				Users: []config.User{
					{
						ID:       "alice",
						SSHUsers: []string{"alice", "alice-dev"},
						Roles:    []string{"admin"},
					},
				},
			},
			sshUser: "alice-dev",
			wantErr: false,
			validate: func(t *testing.T, p *Principal) {
				if p.SSHUser != "alice-dev" {
					t.Errorf("Principal.SSHUser = %q, want %q", p.SSHUser, "alice-dev")
				}
				if p.ConfigUser.ID != "alice" {
					t.Errorf("Principal.ConfigUser.ID = %q, want %q", p.ConfigUser.ID, "alice")
				}
			},
		},
		{
			name: "no matching user",
			cfg: &config.Config{
				Users: []config.User{
					{
						ID:       "alice",
						SSHUsers: []string{"alice"},
						Roles:    []string{"admin"},
					},
				},
			},
			sshUser: "bob",
			wantErr: true,
		},
		{
			name: "empty users list",
			cfg: &config.Config{
				Users: []config.User{},
			},
			sshUser: "alice",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and set SSH_USER
			oldSSHUser := os.Getenv("SSH_USER")
			oldUser := os.Getenv("USER")
			os.Setenv("SSH_USER", tt.sshUser)
			os.Unsetenv("USER")
			defer func() {
				if oldSSHUser == "" {
					os.Unsetenv("SSH_USER")
				} else {
					os.Setenv("SSH_USER", oldSSHUser)
				}
				if oldUser == "" {
					os.Unsetenv("USER")
				} else {
					os.Setenv("USER", oldUser)
				}
			}()

			principal, err := ResolvePrincipal(tt.cfg, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePrincipal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, principal)
			}
		})
	}
}

func TestPrincipal_HasRole(t *testing.T) {
	p := &Principal{
		ConfigUser: &config.User{
			ID:    "alice",
			Roles: []string{"admin", "owner"},
		},
		SSHUser: "alice",
	}

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"has admin role", "admin", true},
		{"has owner role", "owner", true},
		{"does not have user role", "user", false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.HasRole(tt.role)
			if got != tt.want {
				t.Errorf("HasRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestPrincipal_HasAnyRole(t *testing.T) {
	p := &Principal{
		ConfigUser: &config.User{
			ID:    "alice",
			Roles: []string{"admin", "owner"},
		},
		SSHUser: "alice",
	}

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"has one matching role", []string{"admin", "user"}, true},
		{"has all matching roles", []string{"admin", "owner"}, true},
		{"has no matching roles", []string{"user", "guest"}, false},
		{"empty roles list", []string{}, false},
		{"single matching role", []string{"admin"}, true},
		{"single non-matching role", []string{"user"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.HasAnyRole(tt.roles)
			if got != tt.want {
				t.Errorf("HasAnyRole(%v) = %v, want %v", tt.roles, got, tt.want)
			}
		})
	}
}

