package app

import (
	"context"
	"database/sql"
	"log"
	"product-management/internal/app/jobs/shardsmigrator"
	"product-management/internal/infra/config"
	"sync"
	"time"
)

type ShardsMigratorApp struct {
	dbs      map[config.PostgresName]*sql.DB
	migrator *shardsmigrator.Migrator
}

func NewShardsMigratorApp(
	dbs map[config.PostgresName]*sql.DB,
	migrator *shardsmigrator.Migrator,
) *ShardsMigratorApp {
	return &ShardsMigratorApp{dbs: dbs, migrator: migrator}
}

func (app *ShardsMigratorApp) Run(ctx context.Context) error {
	err := app.migrator.Run(ctx)
	app.shutdown()
	return err
}

func (app *ShardsMigratorApp) shutdown() {
	log.Println("shutting down migrator")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	wg := &sync.WaitGroup{}
	for name, db := range app.dbs {
		wg.Go(func() {
			if err := db.Close(); err != nil {
				log.Printf("failed to close %s db: %v", name, err)
			}
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		log.Println("shutting down context timeout")
	}
}
