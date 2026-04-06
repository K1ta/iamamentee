package config

import (
	"fmt"
	"os"
	"product-management/internal/pkg/sharding"
	"strings"

	"github.com/caarlos0/env/v11"
)

type postgresName = string

type Config struct {
	Listen       string                              `env:"APP_LISTEN"`
	KafkaBrokers []string                            `env:"APP_KAFKA_BROKERS"`
	LogToken     string                              `env:"APP_LOG_TOKEN"`
	Shards       map[sharding.ShardName]postgresName `env:"APP_SHARDS"`
	PrevShards   map[sharding.ShardName]postgresName `env:"APP_PREV_SHARDS"`

	// Динамический конфиг, заполняется вручную. Формат названия - APP_POSTGRES_[NAME]_[VARIABLE]=[VALUE].
	// Названия VARIABLE смотреть в [PostgresConfig]
	PostgresDatabases map[postgresName]PostgresConfig
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

	postgresConfigsEnv := make(map[postgresName]map[string]string)
	for k, v := range env.ToMap(os.Environ()) {
		if !strings.HasPrefix("APP_POSTGRES_", k) {
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
		}
		m[parts[1]] = v
	}
	for name, environment := range postgresConfigsEnv {
		var postgresConfig PostgresConfig
		if err := env.ParseWithOptions(&postgresConfig, env.Options{Environment: environment}); err != nil {
			return nil, fmt.Errorf("parse '%s' postgres config: %w", name, err)
		}
		cfg.PostgresDatabases[name] = postgresConfig
	}

	return &cfg, nil
}
