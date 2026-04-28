package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"products/internal/app"
	"products/internal/infra/config"
	"products/internal/infra/search/elasticsearch"
	"products/internal/infra/storage/postgres"
	"products/internal/pkg/sharding"
	"products/internal/service"
	"products/internal/transport/events"
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

		l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		l = l.With("service", "products")
		slog.SetDefault(l)

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		shardsPool, err := initShardsPool(dbs, cfg.Shards)
		if err != nil {
			return fmt.Errorf("init shards: %w", err)
		}
		var prevShardsPool *sharding.Pool[*postgres.ProductRepository]
		if len(cfg.PrevShards) > 0 {
			log.Println("found prev shards in config")
			prevShardsPool, err = initShardsPool(dbs, cfg.PrevShards)
			if err != nil {
				return fmt.Errorf("init prev shards: %w", err)
			}
		}

		repo := postgres.NewShardedProductRepository(shardsPool, prevShardsPool)

		store, err := elasticsearch.NewSearchStore(cfg.ElasticAddresses)
		if err != nil {
			return fmt.Errorf("new search store: %w", err)
		}

		svc := service.NewProductService(repo, store)

		kafkaConsumer := events.NewProductEventConsumer(cfg.KafkaBrokers, svc)
		handler := httpapi.NewSearchHandler(svc)
		router := httpapi.NewRouter(handler)
		server := httpapi.NewServer(cfg.Listen, router, time.Second*5)

		return app.NewServerApp(dbs, kafkaConsumer, server).Run(cmd.Context())
	},
}

func openConnections(configs map[config.PostgresName]config.PostgresConfig) (map[config.PostgresName]*sql.DB, error) {
	dbs := make(map[config.PostgresName]*sql.DB, len(configs))
	for name, pgCfg := range configs {
		db, err := postgres.NewDB(&pgCfg)
		if err != nil {
			return nil, fmt.Errorf("open '%s' postgres connection: %w", name, err)
		}
		dbs[name] = db
	}
	return dbs, nil
}

func initShardsPool(
	dbs map[config.PostgresName]*sql.DB,
	shardsConfig map[sharding.ShardName]config.PostgresName,
) (*sharding.Pool[*postgres.ProductRepository], error) {
	shards := make(map[sharding.ShardName]*postgres.ProductRepository, len(shardsConfig))
	for shardName, dbName := range shardsConfig {
		db, ok := dbs[dbName]
		if !ok {
			return nil, fmt.Errorf("connection %s not found", dbName)
		}
		shards[shardName] = postgres.NewProductRepository(db)
	}
	return sharding.NewPool(shards, sharding.RendezvousResolver)
}
