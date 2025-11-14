package logging

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/glebarez/sqlite"
)

type AuditLogger struct {
	db *sql.DB
}

type AuditEntry struct {
	Time        time.Time
	UserID      string
	SSHUser     string
	OperationID string
	Success     bool
	Error       string
}

func NewAuditLogger(sqlitePath string) (*AuditLogger, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)", sqlitePath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	schema := `
CREATE TABLE IF NOT EXISTS audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  occurred_at TEXT NOT NULL,
  user_id TEXT NOT NULL,
  ssh_user TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  success INTEGER NOT NULL,
  error TEXT
);`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &AuditLogger{db: db}, nil
}

func (l *AuditLogger) Close() error {
	if l.db == nil {
		return nil
	}
	return l.db.Close()
}

func (l *AuditLogger) Log(ctx context.Context, entry AuditEntry) error {
	if l.db == nil {
		return nil
	}

	_, err := l.db.ExecContext(ctx,
		`INSERT INTO audit_log 
		 (occurred_at, user_id, ssh_user, operation_id, success, error)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		entry.Time.UTC().Format(time.RFC3339Nano),
		entry.UserID,
		entry.SSHUser,
		entry.OperationID,
		boolToInt(entry.Success),
		entry.Error,
	)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type AuditRow struct {
	OccurredAt  time.Time
	UserID      string
	SSHUser     string
	OperationID string
	Success     bool
	Error       string
}

// ReadRecent returns the most recent N audit log entries (newest first).
func ReadRecent(l *AuditLogger, limit int) ([]AuditRow, error) {
	if l == nil || l.db == nil {
		return nil, nil
	}

	rows, err := l.db.Query(`
SELECT occurred_at, user_id, ssh_user, operation_id, success, error
FROM audit_log
ORDER BY id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditRow
	for rows.Next() {
		var (
			tsStr  string
			userID string
			ssh    string
			opID   string
			succ   int
			errMsg *string
		)

		if err := rows.Scan(&tsStr, &userID, &ssh, &opID, &succ, &errMsg); err != nil {
			return nil, err
		}

		t, _ := time.Parse(time.RFC3339Nano, tsStr)
		row := AuditRow{
			OccurredAt:  t,
			UserID:      userID,
			SSHUser:     ssh,
			OperationID: opID,
			Success:     succ == 1,
		}
		if errMsg != nil {
			row.Error = *errMsg
		}

		out = append(out, row)
	}

	return out, nil
}

