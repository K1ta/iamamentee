package app

import (
	"context"
	"database/sql"
	"log"
	"product-management/internal/app/jobs/shardsmigrator"
	"sync"
	"time"
)

type ShardsMigratorApp struct {
	dbs      map[string]*sql.DB
	migrator *shardsmigrator.Migrator
}

func NewShardsMigratorApp(
	dbs map[string]*sql.DB,
	migrator *shardsmigrator.Migrator,
) *ShardsMigratorApp {
	return &ShardsMigratorApp{dbs: dbs, migrator: migrator}
}

func (app *ShardsMigratorApp) Run(ctx context.Context) {
	app.migrator.Run(ctx)
	app.shutdown()
}

func (app *ShardsMigratorApp) shutdown() {
	log.Println("shutting down migrator")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	wg := &sync.WaitGroup{}
	closeDBs(wg, app.dbs)

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
