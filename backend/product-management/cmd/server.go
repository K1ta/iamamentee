package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"product-management/internal/app"
	"product-management/internal/app/service"
	"product-management/internal/app/transport/http"
	"product-management/internal/infra/config"
	"product-management/internal/infra/storage"
	"product-management/internal/infra/storage/postgres"
	"product-management/internal/pkg/snowflake"
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

		shards, err := initDbShards(dbs, cfg.Shards)
		if err != nil {
			return fmt.Errorf("init shards: %w", err)
		}
		prevShards, err := initDbShards(dbs, cfg.PrevShards)
		if err != nil {
			return fmt.Errorf("init prev shards: %w", err)
		}

		productRepo, err := createShardedProductRepository(mapDBsToProductRepos(shards), mapDBsToProductRepos(prevShards))
		if err != nil {
			return err
		}

		log.Println("max outbox attemts:", cfg.OutboxConfig.MaxAttempts)
		uowManager, err := postgres.NewUnitOfWorkManager(shards, cfg.OutboxConfig.MaxAttempts)
		if err != nil {
			return fmt.Errorf("new unit of work manager: %w", err)
		}

		productService := service.NewProductService(productRepo, snowflake.NewSnowflake(), uowManager)
		productHandler := http.NewProductHandler(productService)
		router := http.NewRouter(productHandler)

		app := app.NewServerApp(dbs, http.NewHttpServer(cfg.Listen, router, time.Second*5))
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

func initDbShards(dbs map[config.PostgresName]*sql.DB, shardsConfig map[storage.ShardName]config.PostgresName) (storage.Shards[*sql.DB], error) {
	shards := make(storage.Shards[*sql.DB])
	for shardName, dbConnName := range shardsConfig {
		db, ok := dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s for shard %s not found", dbConnName, shardName)
		}
		shards[shardName] = db
	}
	return shards, nil
}

func mapDBsToProductRepos(shards storage.Shards[*sql.DB]) storage.Shards[*postgres.ProductRepository] {
	repos := make(storage.Shards[*postgres.ProductRepository])
	for shardName, db := range shards {
		repos[shardName] = postgres.NewProductRepository(db)
	}
	return repos
}

func createShardedProductRepository(shards, prevShards storage.Shards[*postgres.ProductRepository]) (service.ProductRepository, error) {
	shardsRepo, err := postgres.NewShardedProductRepository(shards)
	if err != nil {
		return nil, fmt.Errorf("new sharded product repo for new shards: %w", err)
	}
	if len(prevShards) == 0 {
		log.Println("prev shards empty, shard migration mode is off")
		return shardsRepo, nil
	}

	log.Println("prev shards not empty, use db in shard migration mode")
	prevShardsRepo, err := postgres.NewShardedProductRepository(prevShards)
	if err != nil {
		return nil, fmt.Errorf("new sharded product repo for prev shards: %w", err)
	}
	return postgres.NewMigratingProductRepository(shardsRepo, prevShardsRepo), nil
}
