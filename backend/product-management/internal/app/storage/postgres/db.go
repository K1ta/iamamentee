package postgres

import (
	"database/sql"
	"fmt"
	"product-management/internal/app/config"

	_ "github.com/lib/pq"
)

func NewDB(cfg *config.PostgresConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Database, cfg.SSLMode)
	db, err := sql.Open("postgres", dsn)
	return db, err
}
