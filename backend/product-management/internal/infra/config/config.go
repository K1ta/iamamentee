package config

import (
	"fmt"
	"os"
	"product-management/internal/infra/storage"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type postgresName = string

type Config struct {
	Listen       string                             `env:"APP_LISTEN"`
	KafkaBrokers []string                           `env:"APP_KAFKA_BROKERS"`
	LogToken     string                             `env:"APP_LOG_TOKEN"`
	Shards       map[storage.ShardName]postgresName `env:"APP_SHARDS"`
	PrevShards   map[storage.ShardName]postgresName `env:"APP_PREV_SHARDS"`

	OutboxConfig OutboxConfig

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

type OutboxConfig struct {
	PauseWhenNoWork time.Duration `env:"APP_OUTBOX_PAUSE_WHEN_NO_WORK"`
	MaxAttempts     int           `env:"APP_OUTBOX_MAX_ATTEMPTS"`
}

func Parse() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	postgresConfigsEnv := make(map[postgresName]map[string]string)
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
	cfg.PostgresDatabases = make(map[postgresName]PostgresConfig)
	for name, environment := range postgresConfigsEnv {
		var postgresConfig PostgresConfig
		if err := env.ParseWithOptions(&postgresConfig, env.Options{Environment: environment}); err != nil {
			return nil, fmt.Errorf("parse '%s' postgres config: %w", name, err)
		}
		cfg.PostgresDatabases[name] = postgresConfig
	}

	return &cfg, nil
}
