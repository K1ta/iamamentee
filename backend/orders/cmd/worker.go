package cmd

import (
	"fmt"
	"log"
	"orders/internal/app"
	"orders/internal/infra/client/productmanagement"
	"orders/internal/infra/config"
	"orders/internal/infra/storage/postgres"
	"orders/internal/service"
	"orders/internal/transport/workers"
	"time"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "run background order processing workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse()
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
		log.SetPrefix(cfg.LogToken + " ")

		pgCfg, ok := cfg.PostgresDatabases["PG"]
		if !ok {
			return fmt.Errorf("postgres config 'PG' not found")
		}
		db, err := postgres.NewDB(&pgCfg)
		if err != nil {
			return fmt.Errorf("open postgres: %w", err)
		}
		defer db.Close()

		repo := postgres.NewOrderRepository(db)
		pmClient := productmanagement.NewClient(cfg.ProductManagementURL)

		orderService := service.NewOrderService(repo, pmClient, service.ProcessingConfig{
			Created: service.StatusConfig{MaxAttempts: cfg.MaxAttemptsCreated, IntervalSec: cfg.IntervalSecCreated},
		})

		pollInterval := time.Duration(cfg.WorkerPollIntervalMs) * time.Millisecond
		orderWorker := workers.NewOrderWorker(orderService, pollInterval)

		workerApp := app.NewWorkerApp(orderWorker)
		return workerApp.Run(cmd.Context())
	},
}
