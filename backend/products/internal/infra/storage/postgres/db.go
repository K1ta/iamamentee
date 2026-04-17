package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"products/internal/infra/config"

	_ "github.com/lib/pq"
)

func NewDB(cfg *config.PostgresConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Database, cfg.SSLMode)
	db, err := sql.Open("postgres", dsn)
	return db, err
}

type DBTX interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
