package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"product-management/internal/app"
	"product-management/internal/infra/client/payments"
	"product-management/internal/infra/config"
	"product-management/internal/infra/storage/postgres"
	"product-management/internal/pkg/sharding"
	"product-management/internal/pkg/snowflake"
	"product-management/internal/service"
	"product-management/internal/transport/httpapi"
	ordersworker "product-management/internal/workers/orders"
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

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		shardsPool, err := initShardsPool(dbs, cfg.Shards)
		if err != nil {
			return fmt.Errorf("init shards: %w", err)
		}
		var prevShardsPool *sharding.Pool[*sql.DB]
		if len(cfg.PrevShards) > 0 {
			log.Println("found prev shards in config")
			prevShardsPool, err = initShardsPool(dbs, cfg.PrevShards)
			if err != nil {
				return fmt.Errorf("init prev shards: %w", err)
			}
		}

		mainDB, ok := dbs[cfg.MainDB]
		if !ok {
			return fmt.Errorf("main db connection not found")
		}

		productView := postgres.NewProductView(shardsPool, prevShardsPool)
		uowFactory := postgres.NewUnitOfWorkFactory(shardsPool, cfg.OutboxConfig.MaxAttempts)

		productService := service.NewProductService(productView, snowflake.NewSnowflake(), uowFactory)
		productHandler := httpapi.NewProductHandler(productService)

		orderRepo := postgres.NewOrderRepository(mainDB)
		paymentsClient := payments.NewClient(cfg.PaymentsURL)
		orderService := service.NewOrderService(orderRepo, paymentsClient, service.OrderConfig{
			MaxAttempts:            cfg.ReservationWorkerConfig.MaxAttempts,
			ReservationIntervalSec: cfg.ReservationWorkerConfig.IntervalSec,
			PaymentIntervalSec:     cfg.PaymentWorkerConfig.IntervalSec,
		})
		reservationHandler := httpapi.NewReservationHandler(orderService)
		reservationWorker := ordersworker.NewReservationWorker(orderService, cfg.ReservationWorkerConfig.PauseWhenNoWork)
		paymentWorker := ordersworker.NewPaymentWorker(orderService, cfg.PaymentWorkerConfig.PauseWhenNoWork)

		router := httpapi.NewRouter(productHandler, reservationHandler)

		app := app.NewServerApp(dbs, httpapi.NewServer(cfg.Listen, router, time.Second*5), reservationWorker, paymentWorker)
		return app.Run(cmd.Context())
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

func initShardsPool(dbs map[config.PostgresName]*sql.DB, shardsConfig map[sharding.ShardName]config.PostgresName) (*sharding.Pool[*sql.DB], error) {
	shards := make(map[sharding.ShardName]*sql.DB)
	for shardName, dbConnName := range shardsConfig {
		db, ok := dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s not found", dbConnName)
		}
		shards[shardName] = db
	}
	return sharding.NewPool(shards, sharding.RendezvousResolver)
}
