package cmd

import (
	"database/sql"
	"delivery/internal/app"
	"delivery/internal/infra/client/orders"
	"delivery/internal/infra/config"
	"delivery/internal/infra/storage/postgres"
	"delivery/internal/service"
	"delivery/internal/transport/httpapi"
	ordersworker "delivery/internal/workers/orders"
	"fmt"
	"log/slog"
	"os"
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

		l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		l = l.With("service", "delivery")
		slog.SetDefault(l)

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		db, ok := dbs["PG0"]
		if !ok {
			return fmt.Errorf("PG0 db connection not found")
		}

		orderDeliveryRepo := postgres.NewOrderDeliveryRepository(db)
		ordersClient := orders.NewClient(cfg.OrdersURL)
		orderDeliveryService := service.NewOrderDeliveryService(orderDeliveryRepo, ordersClient, service.Config{
			MaxAttempts: cfg.MaxAttempts,
			IntervalSec: cfg.OrdersWorkerConfig.IntervalSec,
		})

		worker := ordersworker.NewOrdersWorker(orderDeliveryService, cfg.OrdersWorkerConfig.PauseWhenNoWork)

		handler := httpapi.NewDeliveryHandler(orderDeliveryService)
		router := httpapi.NewRouter(handler)
		server := httpapi.NewServer(cfg.Listen, router, time.Second*5)

		a := app.NewServerApp(dbs, server, worker)
		return a.Run(cmd.Context())
	},
}

func openConnections(configs map[config.PostgresName]config.PostgresConfig) (map[config.PostgresName]*sql.DB, error) {
	dbs := make(map[string]*sql.DB)
	for name, dbCfg := range configs {
		db, err := postgres.NewDB(&dbCfg)
		if err != nil {
			return nil, fmt.Errorf("init '%s' postgres db: %w", name, err)
		}
		dbs[name] = db
	}
	return dbs, nil
}
