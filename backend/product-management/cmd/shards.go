package cmd

import (
	"fmt"
	"log"
	"product-management/internal/app"
	"product-management/internal/app/jobs/shardsmigrator"
	"product-management/internal/infra/config"

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

		migrator, err := shardsmigrator.New(
			shards,
			prevShards,
			cfg.ShardsMigratorConfig.PrevShardsStartFrom,
			cfg.ShardsMigratorConfig.ExcludedPrevShards,
			cfg.ShardsMigratorConfig.BatchLimit,
			true,
		)
		if err != nil {
			return fmt.Errorf("new shards migrator: %w", err)
		}

		app := app.NewShardsMigratorApp(dbs, migrator)
		return app.Run(cmd.Context())
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

		migrator, err := shardsmigrator.New(
			shards,
			prevShards,
			cfg.ShardsMigratorConfig.PrevShardsStartFrom,
			cfg.ShardsMigratorConfig.ExcludedPrevShards,
			cfg.ShardsMigratorConfig.BatchLimit,
			false,
		)
		if err != nil {
			return fmt.Errorf("new shards migrator: %w", err)
		}

		app := app.NewShardsMigratorApp(dbs, migrator)
		return app.Run(cmd.Context())
	},
}
