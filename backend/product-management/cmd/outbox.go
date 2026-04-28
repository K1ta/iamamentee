package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"product-management/internal/app"
	"product-management/internal/infra/config"
	"product-management/internal/infra/messaging/kafka"
	"product-management/internal/infra/storage/postgres"
	"product-management/internal/pkg/sharding"
	"product-management/internal/workers/outbox"

	"github.com/spf13/cobra"
)

var outboxCmd = &cobra.Command{
	Use:   "outbox",
	Short: "run outbox processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse()
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		l = l.With("service", "product-management-outbox")
		slog.SetDefault(l)

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		// Outbox процессор запускает раннер для каждого шарда. Нам не важно имя шарда,
		// важно только запустить один раннер для одной базы. Поэтому мы игнорируем shardName
		// из конфига и используем имя коннекта как shardName
		shards := make(map[sharding.ShardName]outbox.Repository)
		for _, dbConnName := range cfg.Shards {
			if _, ok := dbs[dbConnName]; !ok {
				return fmt.Errorf("connection %s not found", dbConnName)
			}
			shards[sharding.ShardName(dbConnName)] = postgres.NewOutboxProcessorRepository(
				dbs[dbConnName],
				cfg.OutboxConfig.AttemptDurationSec,
				cfg.OutboxConfig.BatchLimit,
			)
		}
		for _, dbConnName := range cfg.PrevShards {
			if _, ok := dbs[dbConnName]; !ok {
				return fmt.Errorf("connection %s not found", dbConnName)
			}
			// Если мы уже добавили шард по такому dbConnName, то не заменяем репозиторий
			if _, ok := shards[sharding.ShardName(dbConnName)]; ok {
				continue
			}
			shards[sharding.ShardName(dbConnName)] = postgres.NewOutboxProcessorRepository(
				dbs[dbConnName],
				cfg.OutboxConfig.AttemptDurationSec,
				cfg.OutboxConfig.BatchLimit,
			)
		}

		kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaWriterBatchSize)

		processor, err := outbox.NewProcessor(
			shards,
			kafkaProducer,
			cfg.OutboxConfig.PauseWhenNoWork,
		)
		if err != nil {
			return fmt.Errorf("new outbox processor: %w", err)
		}

		app := app.NewOutboxApp(processor, kafkaProducer, dbs)
		return app.Run(cmd.Context())
	},
}
