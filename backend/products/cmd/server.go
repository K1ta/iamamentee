package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"products/internal/app"
	"products/internal/infra/config"
	"products/internal/infra/search/elasticsearch"
	"products/internal/transport/events"
	"products/internal/infra/storage/postgres"
	"products/internal/pkg/sharding"
	"products/internal/service"
	"products/internal/transport/httpapi"
	"time"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run http api server with kafka consumer",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse()
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
		log.SetPrefix(cfg.Hostname + " ")

		dbConnections, err := postgres.NewDBConnections(cfg.DBConnections)
		if err != nil {
			return fmt.Errorf("new db connections: %w", err)
		}

		shards, err := buildShardRepos(dbConnections, cfg.Shards)
		if err != nil {
			return fmt.Errorf("build shards: %w", err)
		}
		var prevShards map[sharding.ShardName]*postgres.ProductRepository
		if len(cfg.PrevShards) > 0 {
			prevShards, err = buildShardRepos(dbConnections, cfg.PrevShards)
			if err != nil {
				return fmt.Errorf("build prev shards: %w", err)
			}
		}

		repo := postgres.NewShardedProductRepository(shards, prevShards)

		store, err := elasticsearch.NewSearchStore(cfg.ElasticAddresses)
		if err != nil {
			return fmt.Errorf("new search store: %w", err)
		}

		svc := service.NewProductService(repo, store)

		kafkaConsumer := events.NewProductEventConsumer(cfg.KafkaBrokers, svc)
		handler := httpapi.NewSearchHandler(svc)
		router := httpapi.NewRouter(handler)
		server := httpapi.NewServer(cfg.Listen, router, time.Second*5)

		return app.NewServerApp(kafkaConsumer, server).Run(cmd.Context())
	},
}

func buildShardRepos(
	dbs map[config.DBConnectionName]*sql.DB,
	shardsConfig map[sharding.ShardName]config.DBConnectionName,
) (map[sharding.ShardName]*postgres.ProductRepository, error) {
	shards := make(map[sharding.ShardName]*postgres.ProductRepository, len(shardsConfig))
	for shardName, dbConnName := range shardsConfig {
		db, ok := dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s for shard %s not found", dbConnName, shardName)
		}
		shards[shardName] = postgres.NewProductRepository(db)
	}
	return shards, nil
}
