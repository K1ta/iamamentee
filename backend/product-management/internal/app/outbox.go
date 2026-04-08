package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"product-management/internal/app/jobs/outbox"
	"product-management/internal/infra/config"
	"product-management/internal/infra/messaging/kafka"
	"product-management/internal/infra/storage"
	"product-management/internal/infra/storage/postgres"
	"sync"
	"time"
)

type OutboxApp struct {
	processor     *outbox.Processor
	kafkaProducer *kafka.Producer
	dbs           map[string]*sql.DB
}

func NewOutboxApp(ctx context.Context) (*OutboxApp, error) {
	cfg, err := config.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	log.SetPrefix(cfg.LogToken + " ")

	app := OutboxApp{
		dbs: make(map[string]*sql.DB),
	}

	for name, dbCfg := range cfg.PostgresDatabases {
		db, err := postgres.NewDB(&dbCfg)
		if err != nil {
			return nil, fmt.Errorf("init '%s' postgres db: %w", name, err)
		}
		app.dbs[name] = db
	}

	shards := make(storage.Shards[*sql.DB])
	for shardName, dbConnName := range cfg.Shards {
		db, ok := app.dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s for shard %s not found", dbConnName, shardName)
		}
		shards[shardName] = db
	}
	prevShards := make(storage.Shards[*sql.DB])
	for shardName, dbConnName := range cfg.PrevShards {
		db, ok := app.dbs[dbConnName]
		if !ok {
			return nil, fmt.Errorf("connection %s for prev shard %s not found", dbConnName, shardName)
		}
		prevShards[shardName] = db
	}

	app.kafkaProducer = kafka.NewProducer(cfg.KafkaBrokers)

	app.processor, err = outbox.NewProcessor(
		[]storage.Shards[*sql.DB]{shards, prevShards},
		app.kafkaProducer,
		cfg.OutboxConfig.PauseWhenNoWork,
	)
	if err != nil {
		return nil, fmt.Errorf("new outbox processor: %w", err)
	}
	return &app, nil
}

func (app *OutboxApp) Run(ctx context.Context) error {
	log.Println("outbox app is running")
	err := app.processor.Run(ctx)
	app.shutdown()
	return err
}

func (app *OutboxApp) shutdown() {
	log.Println("shutting down outbox app")
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
	wg.Go(func() {
		if err := app.kafkaProducer.Close(); err != nil {
			log.Printf("failed to close kafka producer: %v", err)
		}
	})

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
