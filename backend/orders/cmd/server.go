package cmd

import (
	"fmt"
	"log"
	"orders/internal/app"
	"orders/internal/infra/client/productmanagement"
	"orders/internal/infra/client/storage"
	"orders/internal/infra/config"
	"orders/internal/infra/storage/postgres"
	"orders/internal/service"
	"orders/internal/transport/httpapi"
	"time"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run http api server",
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
		storClient := storage.NewClient(cfg.StorageURL)

		orderService := service.NewOrderService(repo, pmClient, storClient, service.ProcessingConfig{
			Created:   service.StatusConfig{MaxAttempts: cfg.MaxAttemptsCreated, IntervalSec: cfg.IntervalSecCreated},
			Confirmed: service.StatusConfig{MaxAttempts: cfg.MaxAttemptsConfirmed, IntervalSec: cfg.IntervalSecConfirmed},
		})
		orderHandler := httpapi.NewOrderHandler(orderService)
		router := httpapi.NewRouter(orderHandler)

		serverApp := app.NewServerApp(httpapi.NewServer(cfg.Listen, router, time.Second*5))
		return serverApp.Run(cmd.Context())
	},
}
