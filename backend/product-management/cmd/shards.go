package cmd

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"product-management/internal/app"
	"product-management/internal/infra/config"
	"product-management/internal/infra/storage/postgres"
	"product-management/internal/pkg/sharding"
	"product-management/internal/workers/shardsmigrator"

	"github.com/spf13/cobra"
)

var shardsRootCmd = &cobra.Command{
	Use:   "shards",
	Short: "manage shards migrations",
}

func init() {
	shardsRootCmd.AddCommand(shardsMigrateCmd)
	shardsRootCmd.AddCommand(shardsCleanupCmd)
}

var shardsMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate records from old shards to new shards (without deletion)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse()
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		l = l.With("service", "product-management-shards-migrator")
		slog.SetDefault(l)

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		prevShards := make(map[sharding.ShardName]shardsmigrator.Repository)
		for shardName, dbConnName := range cfg.PrevShards {
			if _, ok := dbs[dbConnName]; !ok {
				return fmt.Errorf("connection %s not found for prev shards", dbConnName)
			}
			prevShards[shardName] = postgres.NewProductRepository(dbs[dbConnName])
		}
		newShardsPool, err := initShardsMigratorProductRepoPool(dbs, cfg.Shards)
		if err != nil {
			return fmt.Errorf("init new shards pool: %w", err)
		}

		migrator := shardsmigrator.New(
			prevShards,
			newShardsPool,
			cfg.ShardsMigratorConfig.PrevShardsStartFrom,
			cfg.ShardsMigratorConfig.ExcludedPrevShards,
			cfg.ShardsMigratorConfig.BatchLimit,
			true,
		)
		app := app.NewShardsMigratorApp(dbs, migrator)
		app.Run(cmd.Context())
		return nil
	},
}

var shardsCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "delete migrated records from old shards",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse()
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		l = l.With("service", "product-management-shards-cleanup")
		slog.SetDefault(l)

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		prevShards := make(map[sharding.ShardName]shardsmigrator.Repository)
		for shardName, dbConnName := range cfg.PrevShards {
			if _, ok := dbs[dbConnName]; !ok {
				return fmt.Errorf("connection %s not found for prev shards", dbConnName)
			}
			prevShards[shardName] = postgres.NewProductRepository(dbs[dbConnName])
		}
		newShardsPool, err := initShardsMigratorProductRepoPool(dbs, cfg.Shards)
		if err != nil {
			return fmt.Errorf("init new shards pool: %w", err)
		}

		migrator := shardsmigrator.New(
			prevShards,
			newShardsPool,
			cfg.ShardsMigratorConfig.PrevShardsStartFrom,
			cfg.ShardsMigratorConfig.ExcludedPrevShards,
			cfg.ShardsMigratorConfig.BatchLimit,
			false,
		)
		app := app.NewShardsMigratorApp(dbs, migrator)
		app.Run(cmd.Context())
		return nil
	},
}

func initShardsMigratorProductRepoPool(
	dbs map[config.PostgresName]*sql.DB,
	shardsConfig map[sharding.ShardName]config.PostgresName,
) (*sharding.Pool[shardsmigrator.Repository], error) {
	shards := make(map[sharding.ShardName]shardsmigrator.Repository)
	for shardName, dbConnName := range shardsConfig {
		db, ok := dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s not found", dbConnName)
		}
		shards[shardName] = postgres.NewProductRepository(db)
	}
	return sharding.NewPool(shards, sharding.RendezvousResolver)
}
