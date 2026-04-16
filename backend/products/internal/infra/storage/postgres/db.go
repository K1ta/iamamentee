package postgres

import (
	"database/sql"
	"fmt"
	"products/internal/infra/config"

	_ "github.com/lib/pq"
)

func NewDBConnections(dbs map[config.DBConnectionName]config.DSN) (map[config.DBConnectionName]*sql.DB, error) {
	res := make(map[config.DBConnectionName]*sql.DB, len(dbs))
	for name, dsn := range dbs {
		db, err := sql.Open("postgres", string(dsn))
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}
		res[name] = db
	}
	return res, nil
}
