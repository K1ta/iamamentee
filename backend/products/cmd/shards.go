package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"products/internal/app"
	"products/internal/infra/config"
	"products/internal/infra/storage/postgres"
	"products/internal/pkg/sharding"
	"products/internal/workers/shardsmigrator"

	"github.com/spf13/cobra"
)

var shardsRootCmd = &cobra.Command{
	Use:   "shards",
	Short: "manage shard migrations",
}

func init() {
	shardsRootCmd.AddCommand(shardsMigrateCmd)
	shardsRootCmd.AddCommand(shardsCleanupCmd)
}

var shardsMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate records from old shards to new shards (without deletion)",
	RunE:  runShardsMigrator(true),
}

var shardsCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "delete migrated records from old shards",
	RunE:  runShardsMigrator(false),
}

func runShardsMigrator(isMigrating bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse()
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
		log.SetPrefix(cfg.Hostname + " ")

		dbs, err := openConnections(cfg.PostgresDatabases)
		if err != nil {
			return fmt.Errorf("open postgres connections: %w", err)
		}

		prevShards := make(map[sharding.ShardName]shardsmigrator.Repository)
		for shardName, dbConnName := range cfg.PrevShards {
			db, ok := dbs[dbConnName]
			if !ok {
				return fmt.Errorf("connection %s not found for prev shard %s", dbConnName, shardName)
			}
			prevShards[shardName] = postgres.NewProductRepository(db)
		}

		newShardsPool, err := initShardsMigratorPool(dbs, cfg.Shards)
		if err != nil {
			return fmt.Errorf("init new shards pool: %w", err)
		}

		migrator := shardsmigrator.New(
			prevShards,
			newShardsPool,
			cfg.ShardsMigratorConfig.PrevShardsStartFrom,
			cfg.ShardsMigratorConfig.ExcludedPrevShards,
			cfg.ShardsMigratorConfig.BatchLimit,
			isMigrating,
		)
		app.NewShardsMigratorApp(dbs, migrator).Run(cmd.Context())
		return nil
	}
}

func initShardsMigratorPool(
	dbs map[config.PostgresName]*sql.DB,
	shardsConfig map[sharding.ShardName]config.PostgresName,
) (*sharding.Pool[shardsmigrator.Repository], error) {
	shards := make(map[sharding.ShardName]shardsmigrator.Repository, len(shardsConfig))
	for shardName, dbConnName := range shardsConfig {
		db, ok := dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s not found", dbConnName)
		}
		shards[shardName] = postgres.NewProductRepository(db)
	}
	return sharding.NewPool(shards, sharding.RendezvousResolver)
}
