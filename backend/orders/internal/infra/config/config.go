package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

type PostgresName = string

type Config struct {
	Listen               string `env:"APP_LISTEN"`
	LogToken             string `env:"APP_LOG_TOKEN"`
	ProductManagementURL string `env:"APP_PRODUCT_MANAGEMENT_URL"`
	StorageURL           string `env:"APP_STORAGE_URL"`
	MaxAttemptsCreated   int `env:"APP_MAX_ATTEMPTS_CREATED"`
	MaxAttemptsConfirmed int `env:"APP_MAX_ATTEMPTS_CONFIRMED"`
	IntervalSecCreated   int `env:"APP_INTERVAL_SEC_CREATED"`
	IntervalSecConfirmed int `env:"APP_INTERVAL_SEC_CONFIRMED"`

	// Динамический конфиг, заполняется вручную. Формат названия - APP_POSTGRES_[NAME]_[VARIABLE]=[VALUE].
	// Названия VARIABLE смотреть в [PostgresConfig]
	PostgresDatabases map[PostgresName]PostgresConfig
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
	cfg.PostgresDatabases = make(map[PostgresName]PostgresConfig)
	for name, environment := range postgresConfigsEnv {
		var postgresConfig PostgresConfig
		if err := env.ParseWithOptions(&postgresConfig, env.Options{Environment: environment}); err != nil {
			return nil, fmt.Errorf("parse '%s' postgres config: %w", name, err)
		}
		cfg.PostgresDatabases[name] = postgresConfig
	}

	return &cfg, nil
}
