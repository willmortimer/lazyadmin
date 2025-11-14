package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/glebarez/sqlite"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// User represents a user stored in SQLite.
type User struct {
	ID        string
	SSHUsers  []string
	Roles     []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Credential represents a FIDO2 credential for a user.
type Credential struct {
	ID           int64
	UserID       string
	RPID         string
	CredentialID string // Base64URL-encoded
	PublicKey    string // Base64URL-encoded SPKI
	CreatedAt    time.Time
}

// Store manages users and credentials in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a new user store with the given SQLite database path.
func NewStore(sqlitePath string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)", sqlitePath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

func (s *Store) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  ssh_users TEXT NOT NULL, -- JSON array
  roles TEXT NOT NULL,     -- JSON array
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS credentials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id TEXT NOT NULL,
  rp_id TEXT NOT NULL,
  credential_id TEXT NOT NULL,
  public_key TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  UNIQUE(user_id, rp_id, credential_id)
);

CREATE INDEX IF NOT EXISTS idx_credentials_user_id ON credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_credentials_rp_id ON credentials(rp_id);
`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// CreateUser creates a new user.
func (s *Store) CreateUser(ctx context.Context, user *User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	sshUsersJSON := marshalStringArray(user.SSHUsers)
	rolesJSON := marshalStringArray(user.Roles)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, ssh_users, roles, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		user.ID, sshUsersJSON, rolesJSON,
		user.CreatedAt.Format(time.RFC3339Nano),
		user.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrUserExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by ID.
func (s *Store) GetUser(ctx context.Context, userID string) (*User, error) {
	var (
		id        string
		sshUsers  string
		roles     string
		createdAt string
		updatedAt string
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT id, ssh_users, roles, created_at, updated_at
		 FROM users WHERE id = ?`,
		userID,
	).Scan(&id, &sshUsers, &roles, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	createdAtTime, _ := time.Parse(time.RFC3339Nano, createdAt)
	updatedAtTime, _ := time.Parse(time.RFC3339Nano, updatedAt)

	return &User{
		ID:        id,
		SSHUsers:  unmarshalStringArray(sshUsers),
		Roles:     unmarshalStringArray(roles),
		CreatedAt: createdAtTime,
		UpdatedAt: updatedAtTime,
	}, nil
}

// FindUserBySSHUser finds a user by SSH username.
func (s *Store) FindUserBySSHUser(ctx context.Context, sshUser string) (*User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, ssh_users, roles, created_at, updated_at
		 FROM users`,
	)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id        string
			sshUsers  string
			roles     string
			createdAt string
			updatedAt string
		)

		if err := rows.Scan(&id, &sshUsers, &roles, &createdAt, &updatedAt); err != nil {
			continue
		}

		userSSHUsers := unmarshalStringArray(sshUsers)
		for _, su := range userSSHUsers {
			if su == sshUser {
				createdAtTime, _ := time.Parse(time.RFC3339Nano, createdAt)
				updatedAtTime, _ := time.Parse(time.RFC3339Nano, updatedAt)

				return &User{
					ID:        id,
					SSHUsers:  userSSHUsers,
					Roles:     unmarshalStringArray(roles),
					CreatedAt: createdAtTime,
					UpdatedAt: updatedAtTime,
				}, nil
			}
		}
	}

	return nil, ErrUserNotFound
}

// ListUsers returns all users.
func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, ssh_users, roles, created_at, updated_at
		 FROM users ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var (
			id        string
			sshUsers  string
			roles     string
			createdAt string
			updatedAt string
		)

		if err := rows.Scan(&id, &sshUsers, &roles, &createdAt, &updatedAt); err != nil {
			continue
		}

		createdAtTime, _ := time.Parse(time.RFC3339Nano, createdAt)
		updatedAtTime, _ := time.Parse(time.RFC3339Nano, updatedAt)

		users = append(users, &User{
			ID:        id,
			SSHUsers:  unmarshalStringArray(sshUsers),
			Roles:     unmarshalStringArray(roles),
			CreatedAt: createdAtTime,
			UpdatedAt: updatedAtTime,
		})
	}

	return users, nil
}

// AddCredential adds a FIDO2 credential to a user.
func (s *Store) AddCredential(ctx context.Context, userID string, cred *Credential) error {
	now := time.Now().UTC()
	cred.UserID = userID
	cred.CreatedAt = now

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO credentials (user_id, rp_id, credential_id, public_key, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		cred.UserID, cred.RPID, cred.CredentialID, cred.PublicKey,
		cred.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("credential already exists")
		}
		return fmt.Errorf("add credential: %w", err)
	}
	return nil
}

// GetCredentials returns all credentials for a user.
func (s *Store) GetCredentials(ctx context.Context, userID string) ([]*Credential, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, rp_id, credential_id, public_key, created_at
		 FROM credentials WHERE user_id = ? ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}
	defer rows.Close()

	var creds []*Credential
	for rows.Next() {
		var (
			id           int64
			uid          string
			rpID         string
			credentialID string
			publicKey    string
			createdAt    string
		)

		if err := rows.Scan(&id, &uid, &rpID, &credentialID, &publicKey, &createdAt); err != nil {
			continue
		}

		createdAtTime, _ := time.Parse(time.RFC3339Nano, createdAt)

		creds = append(creds, &Credential{
			ID:           id,
			UserID:       uid,
			RPID:         rpID,
			CredentialID: credentialID,
			PublicKey:    publicKey,
			CreatedAt:    createdAtTime,
		})
	}

	return creds, nil
}

// DeleteUser deletes a user and all their credentials.
func (s *Store) DeleteUser(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// Helper functions for JSON array marshaling (simple implementation)
func marshalStringArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	result := "["
	for i, s := range arr {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s"`, s)
	}
	result += "]"
	return result
}

func unmarshalStringArray(s string) []string {
	// Simple JSON array parser - assumes format ["a","b","c"]
	if s == "" || s == "[]" {
		return []string{}
	}
	s = s[1 : len(s)-1] // Remove [ and ]
	if s == "" {
		return []string{}
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if start < i {
				val := s[start:i]
				val = val[1 : len(val)-1] // Remove quotes
				result = append(result, val)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		val := s[start:]
		val = val[1 : len(val)-1] // Remove quotes
		result = append(result, val)
	}
	return result
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "UNIQUE constraint") || contains(errStr, "unique constraint")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
