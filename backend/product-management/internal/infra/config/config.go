package config

import (
	"fmt"
	"os"
	"product-management/internal/pkg/sharding"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type PostgresName = string

type Config struct {
	Listen               string                              `env:"APP_LISTEN"`
	PaymentsURL          string                              `env:"APP_PAYMENTS_URL"`
	KafkaBrokers         []string                            `env:"APP_KAFKA_BROKERS"`
	KafkaWriterBatchSize int                                 `env:"APP_KAFKA_WRITER_BATCH_SIZE"`
	LogToken             string                              `env:"APP_LOG_TOKEN"`
	Shards               map[sharding.ShardName]PostgresName `env:"APP_SHARDS"`
	PrevShards           map[sharding.ShardName]PostgresName `env:"APP_PREV_SHARDS"`
	MainDB               PostgresName                        `env:"APP_MAIN_DB"`

	OutboxConfig            OutboxConfig
	ShardsMigratorConfig    ShardsMigratorConfig
	ReservationWorkerConfig ReservationWorkerConfig
	PaymentWorkerConfig     PaymentWorkerConfig

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

type OutboxConfig struct {
	PauseWhenNoWork    time.Duration `env:"APP_OUTBOX_PAUSE_WHEN_NO_WORK"`
	MaxAttempts        int           `env:"APP_OUTBOX_MAX_ATTEMPTS"`
	BatchLimit         int           `env:"APP_OUTBOX_PROCESSOR_BATCH_SIZE"`
	AttemptDurationSec int           `env:"APP_OUTBOX_PROCESSOR_ATTEMPT_DURATION_SECONDS"`
}

type ShardsMigratorConfig struct {
	PrevShardsStartFrom map[sharding.ShardName]int64 `env:"APP_SHARDSMIGRATOR_PREV_SHARDS_START_FROM"`
	ExcludedPrevShards  []sharding.ShardName         `env:"APP_SHARDSMIGRATOR_EXCLUDED_PREV_SHARDS"`
	BatchLimit          int                          `env:"APP_SHARDSMIGRATOR_BATCH_LIMIT"`
}

type ReservationWorkerConfig struct {
	IntervalSec     int           `env:"APP_RESERVATION_INTERVAL_SEC"`
	MaxAttempts     int           `env:"APP_RESERVATION_MAX_ATTEMPTS"`
	PauseWhenNoWork time.Duration `env:"APP_RESERVATION_PAUSE_WHEN_NO_WORK"`
}

type PaymentWorkerConfig struct {
	IntervalSec     int           `env:"APP_PAYMENT_INTERVAL_SEC"`
	PauseWhenNoWork time.Duration `env:"APP_PAYMENT_PAUSE_WHEN_NO_WORK"`
}

func Parse() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Заполняем коннекты к базе
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
