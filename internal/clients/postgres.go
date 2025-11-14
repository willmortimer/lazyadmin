package clients

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresClient struct {
	DB *sql.DB
}

func NewPostgresClient(dsn string) (*PostgresClient, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open pg: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping pg: %w", err)
	}
	return &PostgresClient{DB: db}, nil
}

func (c *PostgresClient) RunScalarQuery(ctx context.Context, query string) (string, error) {
	row := c.DB.QueryRowContext(ctx, query)
	var value any
	if err := row.Scan(&value); err != nil {
		return "", fmt.Errorf("scan: %w", err)
	}
	return fmt.Sprintf("%v", value), nil
}

