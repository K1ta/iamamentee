package cmd

import (
	"fmt"
	"log"
	"orders/internal/app"
	"orders/internal/infra/config"
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

		orderHandler := httpapi.NewOrderHandler()
		router := httpapi.NewRouter(orderHandler)

		serverApp := app.NewServerApp(httpapi.NewServer(cfg.Listen, router, time.Second*5))
		return serverApp.Run(cmd.Context())
	},
}
