package config

import (
	"fmt"
	"os"
	"products/internal/pkg/sharding"
	"strings"

	"github.com/caarlos0/env/v11"
)

type PostgresName = string

type Config struct {
	Listen           string                              `env:"APP_LISTEN"`
	KafkaBrokers     []string                            `env:"APP_KAFKA_BROKERS"`
	ElasticAddresses []string                            `env:"APP_ELASTIC_ADDRESSES"`
	Shards           map[sharding.ShardName]PostgresName `env:"APP_SHARDS"`
	PrevShards       map[sharding.ShardName]PostgresName `env:"APP_PREV_SHARDS"`
	Hostname         string                              `env:"HOSTNAME"` // k8s env

	ShardsMigratorConfig ShardsMigratorConfig

	// Динамический конфиг, заполняется вручную. Формат - APP_POSTGRES_[NAME]_[VARIABLE]=[VALUE].
	// Названия VARIABLE смотреть в [PostgresConfig].
	PostgresDatabases map[PostgresName]PostgresConfig
}

type ShardsMigratorConfig struct {
	PrevShardsStartFrom map[sharding.ShardName]int64 `env:"APP_SHARDSMIGRATOR_PREV_SHARDS_START_FROM"`
	ExcludedPrevShards  []sharding.ShardName         `env:"APP_SHARDSMIGRATOR_EXCLUDED_PREV_SHARDS"`
	BatchLimit          int                          `env:"APP_SHARDSMIGRATOR_BATCH_LIMIT"`
}

type PostgresConfig struct {
	Host     string `env:"HOST"`
	Database string `env:"DATABASE"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	SSLMode  string `env:"SSL_MODE"`
}

func Parse() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	postgresConfigsEnv := make(map[PostgresName]map[string]string)
	for k, v := range env.ToMap(os.Environ()) {
		if !strings.HasPrefix(k, "APP_POSTGRES_") {
			continue
		}
		k = strings.TrimPrefix(k, "APP_POSTGRES_")
		parts := strings.SplitN(k, "_", 2)
		if len(parts) != 2 {
			continue
		}
		m, ok := postgresConfigsEnv[parts[0]]
		if !ok {
			m = make(map[string]string)
			postgresConfigsEnv[parts[0]] = m
		}
		m[parts[1]] = v
	}
	cfg.PostgresDatabases = make(map[PostgresName]PostgresConfig, len(postgresConfigsEnv))
	for name, environment := range postgresConfigsEnv {
		var pgCfg PostgresConfig
		if err := env.ParseWithOptions(&pgCfg, env.Options{Environment: environment}); err != nil {
			return nil, fmt.Errorf("parse '%s' postgres config: %w", name, err)
		}
		cfg.PostgresDatabases[name] = pgCfg
	}

	return &cfg, nil
}
