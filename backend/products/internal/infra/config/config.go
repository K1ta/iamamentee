package config

import (
	"fmt"
	"products/internal/pkg/sharding"

	"github.com/caarlos0/env/v11"
)

type (
	DBConnectionName = string
	DSN              = string
)

type Config struct {
	Listen           string                                  `env:"APP_LISTEN"`
	KafkaBrokers     []string                                `env:"APP_KAFKA_BROKERS"`
	ElasticAddresses []string                                `env:"APP_ELASTIC_ADDRESSES"`
	DBConnections    map[DBConnectionName]DSN                `env:"APP_DB_CONNECTIONS" envKeyValSeparator:">"`
	Shards           map[sharding.ShardName]DBConnectionName `env:"APP_SHARDS"`
	PrevShards       map[sharding.ShardName]DBConnectionName `env:"APP_PREV_SHARDS"`

	Hostname string `env:"HOSTNAME"` // k8s env
}

func Parse() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
