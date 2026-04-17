// Package app содержит re-exports для обратной совместимости с shard_migrator.
// Новый код должен использовать пакеты внутри internal/.
package app

import (
	"database/sql"
	"fmt"
	"maps"
	"products/internal/domain"
	"products/internal/pkg/sharding"
	"slices"

	_ "github.com/lib/pq"
)

// Type aliases — обратная совместимость с shard_migrator.
// ShardName намеренно алиасируется к string, а не к sharding.ShardName,
// чтобы оставаться совместимым с []string в конфиге shard_migrator.
type (
	DBConnectionName = string
	DSN              = string
	ShardName        = string
	Product          = domain.Product
)

// NewDBConnections — обратная совместимость с shard_migrator.
// Принимает карту DSN-строк напрямую (shard_migrator использует свой формат конфига).
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

// GetShard — обратная совместимость с shard_migrator.
func GetShard[V any](shards map[ShardName]V, key string) (ShardName, V) {
	names := slices.Collect(maps.Keys(shards))
	shardNames := make([]sharding.ShardName, len(names))
	for i, n := range names {
		shardNames[i] = sharding.ShardName(n)
	}
	name := sharding.RendezvousResolver(shardNames, key)
	return string(name), shards[string(name)]
}
