package app

import (
	"context"
	"database/sql"
	"log"
	"products/internal/workers/shardsmigrator"
	"sync"
	"time"
)

type ShardsMigratorApp struct {
	dbs      map[string]*sql.DB
	migrator *shardsmigrator.Migrator
}

func NewShardsMigratorApp(dbs map[string]*sql.DB, migrator *shardsmigrator.Migrator) *ShardsMigratorApp {
	return &ShardsMigratorApp{dbs: dbs, migrator: migrator}
}

func (a *ShardsMigratorApp) Run(ctx context.Context) {
	a.migrator.Run(ctx)
	a.shutdown()
}

func (a *ShardsMigratorApp) shutdown() {
	log.Println("shutting down migrator")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	wg := &sync.WaitGroup{}
	closeDBs(wg, a.dbs)

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
