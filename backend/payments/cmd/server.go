package cmd

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"payments/internal/app"
	"payments/internal/infra/client/delivery"
	"payments/internal/infra/client/productmanagement"
	"payments/internal/infra/config"
	"payments/internal/infra/storage/postgres"
	"payments/internal/service"
	"payments/internal/transport/httpapi"
	cancellationworker "payments/internal/workers/cancellation"
	compensationworker "payments/internal/workers/compensation"
	deliveryworker "payments/internal/workers/delivery"
	failingworker "payments/internal/workers/failing"
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
		l = l.With("service", "payments")
		slog.SetDefault(l)

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		db, ok := dbs["PG0"]
		if !ok {
			return fmt.Errorf("PG0 db connection not found")
		}

		orderPaymentRepo := postgres.NewOrderPaymentRepository(db)
		deliveryClient := delivery.NewClient(cfg.DeliveryURL)
		productManagementClient := productmanagement.NewClient(cfg.ProductManagementURL)
		orderPaymentService := service.NewOrderPaymentService(orderPaymentRepo, deliveryClient, productManagementClient, service.DeliveryWorkerConfig{
			IntervalSec:             cfg.DeliveryWorkerConfig.IntervalSec,
			FailingIntervalSec:      cfg.FailingWorkerConfig.IntervalSec,
			CompensationIntervalSec: cfg.CompensationWorkerConfig.IntervalSec,
			CancellationIntervalSec: cfg.CancellationWorkerConfig.IntervalSec,
		})

		worker := deliveryworker.NewDeliveryWorker(orderPaymentService, cfg.DeliveryWorkerConfig.PauseWhenNoWork)
		fWorker := failingworker.NewFailingWorker(orderPaymentService, cfg.FailingWorkerConfig.PauseWhenNoWork)
		cWorker := compensationworker.NewCompensationWorker(orderPaymentService, cfg.CompensationWorkerConfig.PauseWhenNoWork)
		cancelWorker := cancellationworker.NewCancellationWorker(orderPaymentService, cfg.CancellationWorkerConfig.PauseWhenNoWork)

		handler := httpapi.NewPaymentHandler(orderPaymentService)
		router := httpapi.NewRouter(handler)
		server := httpapi.NewServer(cfg.Listen, router, time.Second*5)

		a := app.NewServerApp(dbs, server, worker, fWorker, cWorker, cancelWorker)
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
