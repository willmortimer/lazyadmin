package logging

import (
	"context"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "in-memory database",
			path:    ":memory:",
			wantErr: false,
		},
		{
			name:    "temp file",
			path:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if path == "" {
				path = t.TempDir() + "/test.db"
			}

			logger, err := NewAuditLogger(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAuditLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if logger == nil {
					t.Fatal("NewAuditLogger() returned nil logger")
				}
				if logger.db == nil {
					t.Error("logger.db is nil")
				}

				// Cleanup
				if err := logger.Close(); err != nil {
					t.Errorf("Close() error = %v", err)
				}
			}
		})
	}
}

func TestAuditLogger_Log(t *testing.T) {
	logger, err := NewAuditLogger(":memory:")
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		entry   AuditEntry
		wantErr bool
	}{
		{
			name: "successful operation",
			entry: AuditEntry{
				Time:        time.Now(),
				UserID:      "alice",
				SSHUser:     "alice",
				OperationID: "get-users",
				Success:     true,
				Error:       "",
			},
			wantErr: false,
		},
		{
			name: "failed operation",
			entry: AuditEntry{
				Time:        time.Now(),
				UserID:      "bob",
				SSHUser:     "bob",
				OperationID: "delete-user",
				Success:     false,
				Error:       "permission denied",
			},
			wantErr: false,
		},
		{
			name: "task execution",
			entry: AuditEntry{
				Time:        time.Now(),
				UserID:      "alice",
				SSHUser:     "alice",
				OperationID: "task:deploy",
				Success:     true,
				Error:       "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logger.Log(ctx, tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("Log() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditLogger_ReadRecent(t *testing.T) {
	logger, err := NewAuditLogger(":memory:")
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	// Insert test entries
	entries := []AuditEntry{
		{
			Time:        time.Now().Add(-5 * time.Minute),
			UserID:      "alice",
			SSHUser:     "alice",
			OperationID: "op1",
			Success:     true,
		},
		{
			Time:        time.Now().Add(-3 * time.Minute),
			UserID:      "bob",
			SSHUser:     "bob",
			OperationID: "op2",
			Success:     false,
			Error:       "failed",
		},
		{
			Time:        time.Now().Add(-1 * time.Minute),
			UserID:      "alice",
			SSHUser:     "alice",
			OperationID: "op3",
			Success:     true,
		},
	}

	for _, entry := range entries {
		if err := logger.Log(ctx, entry); err != nil {
			t.Fatalf("Log() error = %v", err)
		}
	}

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"limit 1", 1, 1},
		{"limit 2", 2, 2},
		{"limit 10", 10, 3},
		{"limit 0", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := ReadRecent(logger, tt.limit)
			if err != nil {
				t.Fatalf("ReadRecent() error = %v", err)
			}

			if len(rows) != tt.want {
				t.Errorf("ReadRecent() returned %d rows, want %d", len(rows), tt.want)
			}

			// Verify ordering (newest first)
			if len(rows) > 1 {
				for i := 0; i < len(rows)-1; i++ {
					if rows[i].OccurredAt.Before(rows[i+1].OccurredAt) {
						t.Error("ReadRecent() returned rows in wrong order (should be newest first)")
					}
				}
			}
		})
	}
}

func TestAuditLogger_ReadRecent_Empty(t *testing.T) {
	logger, err := NewAuditLogger(":memory:")
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	rows, err := ReadRecent(logger, 10)
	if err != nil {
		t.Fatalf("ReadRecent() error = %v", err)
	}

	if len(rows) != 0 {
		t.Errorf("ReadRecent() returned %d rows, want 0", len(rows))
	}
}

func TestAuditLogger_Close(t *testing.T) {
	logger, err := NewAuditLogger(":memory:")
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Closing again should be safe
	if err := logger.Close(); err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestAuditLogger_Log_NilDB(t *testing.T) {
	// Test that Log handles nil db gracefully
	logger := &AuditLogger{db: nil}
	ctx := context.Background()

	err := logger.Log(ctx, AuditEntry{})
	if err != nil {
		t.Errorf("Log() with nil db error = %v, want nil", err)
	}
}

func TestReadRecent_NilLogger(t *testing.T) {
	var logger *AuditLogger

	rows, err := ReadRecent(logger, 10)
	if err != nil {
		t.Errorf("ReadRecent() on nil logger error = %v, want nil", err)
	}

	if rows != nil {
		t.Errorf("ReadRecent() on nil logger returned %v, want nil", rows)
	}
}

func TestBoolToInt(t *testing.T) {
	tests := []struct {
		name string
		b    bool
		want int
	}{
		{"true", true, 1},
		{"false", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolToInt(tt.b)
			if got != tt.want {
				t.Errorf("boolToInt(%v) = %d, want %d", tt.b, got, tt.want)
			}
		})
	}
}
