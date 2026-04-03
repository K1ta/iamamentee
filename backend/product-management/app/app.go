package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
)

type (
	DBConnectionName = string
	ShardName        = string
	DSN              = string
)

type Config struct {
	Listen        string                         `env:"APP_LISTEN"`
	KafkaBrokers  []string                       `env:"APP_KAFKA_BROKERS"`
	DBConnections map[DBConnectionName]DSN       `env:"APP_DB_CONNECTIONS" envKeyValSeparator:">"`
	Shards        map[ShardName]DBConnectionName `env:"APP_SHARDS"`
	PrevShards    map[ShardName]DBConnectionName `env:"APP_PREV_SHARDS"`

	Hostname string `env:"HOSTNAME"`
}

func Run(ctx context.Context) error {
	var conf Config
	if err := env.Parse(&conf); err != nil {
		log.Panicln("parse config:", err)
	}
	log.SetPrefix(conf.Hostname + " ")

	snowflake := NewSnowflake()
	dbConnections, err := NewDBConnections(conf.DBConnections)
	if err != nil {
		return fmt.Errorf("new db connections: %w", err)
	}
	repoShards := make(map[ShardName]ProductRepository)
	for shardName, dbConnName := range conf.Shards {
		db, ok := dbConnections[dbConnName]
		if !ok {
			return fmt.Errorf("connection %s for shard %s not found", dbConnName, shardName)
		}
		repoShards[shardName] = NewProductRepository(db, snowflake)
	}
	var repo ProductRepository
	shardedRepo, err := NewShardedProductRepository(repoShards)
	if err != nil {
		return fmt.Errorf("new sharded search repository: %w", err)
	}
	repo = shardedRepo
	if len(conf.PrevShards) > 0 {
		log.Println("prev shards not empty, use db in shard migration mode")
		prevRepoShards := make(map[ShardName]ProductRepository)
		for shardName, dbConnName := range conf.PrevShards {
			db, ok := dbConnections[dbConnName]
			if !ok {
				return fmt.Errorf("connection %s for prev shard %s not found", dbConnName, shardName)
			}
			prevRepoShards[shardName] = NewProductRepository(db, snowflake)
		}
		prevShardsRepo, err := NewShardedProductRepository(prevRepoShards)
		if err != nil {
			return fmt.Errorf("new sharded search repository for prev shards: %w", err)
		}
		repo = NewMigratingShardedProductRepository(shardedRepo, prevShardsRepo)
	}
	producer := NewKafkaProductProducer(conf.KafkaBrokers)
	defer func() {
		log.Println("closing kafka producer")
		if err := producer.Close(); err != nil {
			log.Println("failed close kafka producer:", err)
		} else {
			log.Println("kafka producer closed")
		}
	}()
	handler := NewProductHandler(repo, producer)
	router := NewRouter(handler)
	server := NewHttpServer(conf.Listen, router, time.Second*5)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return server.Run(egCtx)
	})
	log.Println("service is running")
	return eg.Wait()
}
