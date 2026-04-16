// Package app содержит re-exports для обратной совместимости с shard_migrator.
// Новый код должен использовать пакеты внутри internal/.
package app

import (
	"products/internal/domain"
	"products/internal/infra/config"
	"products/internal/infra/storage/postgres"
	"products/internal/pkg/sharding"
)

// Type aliases — обратная совместимость с shard_migrator.
type (
	DBConnectionName = config.DBConnectionName
	DSN              = config.DSN
	ShardName        = sharding.ShardName
	Product          = domain.Product
)

// NewDBConnections — обратная совместимость с shard_migrator.
var NewDBConnections = postgres.NewDBConnections

// GetShard — обратная совместимость с shard_migrator.
func GetShard[V any](shards map[ShardName]V, key string) (ShardName, V) {
	return sharding.GetShard(shards, key)
}
