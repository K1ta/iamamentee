package app

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewDBConnections(dbs map[DBConnectionName]DSN) (map[DBConnectionName]*sql.DB, error) {
	res := make(map[DBConnectionName]*sql.DB, len(dbs))
	for name, dsn := range dbs {
		db, err := sql.Open("postgres", string(dsn))
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}
		res[name] = db
	}
	return res, nil
}
