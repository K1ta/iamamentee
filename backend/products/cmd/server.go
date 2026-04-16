package cmd

import (
	"fmt"
	"log"
	"products/internal/app"
	"products/internal/domain"
	"products/internal/infra/config"
	"products/internal/infra/messaging/kafka"
	"products/internal/infra/search/elasticsearch"
	"products/internal/infra/storage/postgres"
	"products/internal/pkg/sharding"
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

		repoShards := make(map[sharding.ShardName]domain.SearchRepository)
		for shardName, dbConnName := range cfg.Shards {
			db, ok := dbConnections[dbConnName]
			if !ok {
				return fmt.Errorf("connection %s for shard %s not found", dbConnName, shardName)
			}
			repoShards[shardName] = postgres.NewSearchRepository(db)
		}
		shardedRepo, err := postgres.NewShardedSearchRepository(repoShards)
		if err != nil {
			return fmt.Errorf("new sharded search repository: %w", err)
		}

		var repo domain.SearchRepository = shardedRepo
		if len(cfg.PrevShards) > 0 {
			prevRepoShards := make(map[sharding.ShardName]domain.SearchRepository)
			for shardName, dbConnName := range cfg.PrevShards {
				db, ok := dbConnections[dbConnName]
				if !ok {
					return fmt.Errorf("connection %s for prev shard %s not found", dbConnName, shardName)
				}
				prevRepoShards[shardName] = postgres.NewSearchRepository(db)
			}
			prevShardsRepo, err := postgres.NewShardedSearchRepository(prevRepoShards)
			if err != nil {
				return fmt.Errorf("new sharded search repository for prev shards: %w", err)
			}
			repo = postgres.NewMigratingShardedSearchRepository(shardedRepo, prevShardsRepo)
		}

		store, err := elasticsearch.NewSearchStore(cfg.ElasticAddresses)
		if err != nil {
			return fmt.Errorf("new search store: %w", err)
		}

		kafkaConsumer := kafka.NewProductEventConsumer(cfg.KafkaBrokers, repo, store)
		handler := httpapi.NewSearchHandler(repo, store)
		router := httpapi.NewRouter(handler)
		server := httpapi.NewServer(cfg.Listen, router, time.Second*5)

		return app.NewServerApp(kafkaConsumer, server).Run(cmd.Context())
	},
}
