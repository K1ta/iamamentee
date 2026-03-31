package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env"
	"golang.org/x/sync/errgroup"
)

type (
	DBConnectionName string
	ShardName        string
	DSN              string
)

type Config struct {
	Listen           string                         `env:"APP_LISTEN"`
	KafkaBrokers     []string                       `env:"APP_KAFKA_BROKERS"`
	ElasticAddresses []string                       `env:"APP_ELASTIC_ADDRESSES"`
	DBConnections    map[DBConnectionName]DSN       `env:"APP_DB_CONNECTIONS"`
	Shards           map[ShardName]DBConnectionName `env:"APP_SHARDS"`
	PrevShards       map[ShardName]DBConnectionName `env:"APP_PREV_SHARDS"`

	Hostname string `env:"HOSTNAME"` // k8s env
}

func Run(ctx context.Context) error {
	var conf Config
	if err := env.Parse(&conf); err != nil {
		log.Panicln("parse config:", err)
	}
	log.SetPrefix(conf.Hostname + " ")

	dbConnections, err := NewDBConnections(conf.DBConnections)
	if err != nil {
		return fmt.Errorf("new db connections: %w", err)
	}
	var repo SearchRepository
	shardedRepo, err := NewShardedSearchRepository(conf.Shards, dbConnections)
	if err != nil {
		return fmt.Errorf("new sharded search repository: %w", err)
	}
	repo = shardedRepo
	if len(conf.PrevShards) > 0 {
		prevShardsRepo, err := NewShardedSearchRepository(conf.PrevShards, dbConnections)
		if err != nil {
			return fmt.Errorf("new sharded search repository for prev shards: %w", err)
		}
		repo = NewMigratingShardedSearchRepository(shardedRepo, prevShardsRepo)
	}
	store, err := NewSearchStore(conf.ElasticAddresses)
	if err != nil {
		return fmt.Errorf("new search store: %w", err)
	}
	kafkaConsumer := NewProductEventConsumer(conf.KafkaBrokers, repo, store)
	handler := NewSearchHandler(repo, store)
	router := NewRouter(handler)
	server := NewHttpServer(conf.Listen, router, time.Second*5)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return kafkaConsumer.Run(egCtx)
	})
	eg.Go(func() error {
		return server.Run(egCtx)
	})
	log.Println("service is running")
	return eg.Wait()
}
