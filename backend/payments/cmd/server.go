package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"payments/internal/app"
	"payments/internal/infra/config"
	"payments/internal/infra/storage/postgres"
	"payments/internal/transport/httpapi"
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

		handler := httpapi.NewPaymentHandler()
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
