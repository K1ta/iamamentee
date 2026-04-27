package cmd

import (
	"database/sql"
	"delivery/internal/app"
	"delivery/internal/infra/config"
	"delivery/internal/infra/storage/postgres"
	"delivery/internal/service"
	"delivery/internal/transport/httpapi"
	"fmt"
	"log"
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

		db, ok := dbs["PG0"]
		if !ok {
			return fmt.Errorf("PG0 db connection not found")
		}

		orderDeliveryRepo := postgres.NewOrderDeliveryRepository(db)
		orderDeliveryService := service.NewOrderDeliveryService(orderDeliveryRepo, service.Config{
			MaxAttempts: cfg.MaxAttempts,
		})

		handler := httpapi.NewDeliveryHandler(orderDeliveryService)
		router := httpapi.NewRouter(handler)
		server := httpapi.NewServer(cfg.Listen, router, time.Second*5)

		a := app.NewServerApp(dbs, server)
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
