package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"product-management/internal/app/service"
	"product-management/internal/app/transport/http"
	"product-management/internal/infra/config"
	"product-management/internal/infra/storage"
	"product-management/internal/infra/storage/postgres"
	"product-management/internal/pkg/snowflake"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type App struct {
	dbs        map[string]*sql.DB
	httpServer *http.HttpServer
	service    *service.ProductService
}

func New() (*App, error) {
	cfg, err := config.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	log.SetPrefix(cfg.LogToken + " ")

	app := App{
		dbs: make(map[string]*sql.DB),
	}

	for name, dbCfg := range cfg.PostgresDatabases {
		db, err := postgres.NewDB(&dbCfg)
		if err != nil {
			return nil, fmt.Errorf("init '%s' postgres db: %w", name, err)
		}
		app.dbs[name] = db
	}

	shards := make(storage.Shards[*postgres.ProductRepository])
	dbShards := make(storage.Shards[*sql.DB])
	for shardName, dbConnName := range cfg.Shards {
		db, ok := app.dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s for shard %s not found", dbConnName, shardName)
		}
		shards[shardName] = postgres.NewProductRepository(db)
		dbShards[shardName] = db
	}
	prevShards := make(storage.Shards[*postgres.ProductRepository])
	for shardName, dbConnName := range cfg.PrevShards {
		db, ok := app.dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s for prev shard %s not found", dbConnName, shardName)
		}
		prevShards[shardName] = postgres.NewProductRepository(db)
	}
	repo, err := createShardedProductRepository(shards, prevShards)
	if err != nil {
		return nil, err
	}

	uowManager, err := postgres.NewUnitOfWorkManager(dbShards, cfg.OutboxConfig.MaxAttempts)
	if err != nil {
		return nil, fmt.Errorf("new unit of work manager: %w", err)
	}

	service := service.NewProductService(repo, snowflake.NewSnowflake(), uowManager, cfg.OutboxConfig.MaxAttempts)
	handler := http.NewProductHandler(service)
	router := http.NewRouter(handler)
	app.httpServer = http.NewHttpServer(cfg.Listen, router, time.Second*5)

	return &app, nil
}

func createShardedProductRepository(shards, prevShards storage.Shards[*postgres.ProductRepository]) (service.ProductRepository, error) {
	shardsRepo, err := postgres.NewShardedProductRepository(shards)
	if err != nil {
		return nil, fmt.Errorf("new sharded product repo: %w", err)
	}
	var repo service.ProductRepository = shardsRepo
	if len(prevShards) > 0 {
		log.Println("prev shards not empty, use db in shard migration mode")
		prevShardsRepo, err := postgres.NewShardedProductRepository(prevShards)
		if err != nil {
			return nil, fmt.Errorf("new sharded product repo for prev shards: %w", err)
		}
		repo = postgres.NewMigratingProductRepository(shardsRepo, prevShardsRepo)
	}
	return repo, nil
}

func (app *App) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return app.httpServer.Run(egCtx)
	})
	log.Println("app is running")
	err := eg.Wait()
	app.shutdown()
	return err
}

func (app *App) shutdown() {
	log.Println("shutting down app")
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
