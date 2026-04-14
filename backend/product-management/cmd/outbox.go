package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"product-management/internal/app"
	"product-management/internal/app/jobs/outbox"
	"product-management/internal/infra/config"
	"product-management/internal/infra/messaging/kafka"
	"product-management/internal/infra/storage"

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
		log.SetPrefix(cfg.LogToken + " ")

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		shards, err := initDbShards(dbs, cfg.Shards)
		if err != nil {
			return fmt.Errorf("init shards: %w", err)
		}
		prevShards, err := initDbShards(dbs, cfg.PrevShards)
		if err != nil {
			return fmt.Errorf("init prev shards: %w", err)
		}

		kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers)

		processor, err := outbox.NewProcessor(
			[]storage.Shards[*sql.DB]{shards, prevShards},
			kafkaProducer,
			cfg.OutboxConfig.PauseWhenNoWork,
			cfg.OutboxConfig.MaxAttempts,
		)
		if err != nil {
			return fmt.Errorf("new outbox processor: %w", err)
		}

		app := app.NewOutboxApp(processor, kafkaProducer, dbs)
		return app.Run(cmd.Context())
	},
}
