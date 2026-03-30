package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Listen           string   `env:"APP_LISTEN"`
	PgHost           string   `env:"APP_PG_HOST"`
	PgDatabase       string   `env:"APP_PG_DATABASE"`
	PgUser           string   `env:"APP_PG_USER"`
	PgPassword       string   `env:"APP_PG_PASSWORD"`
	KafkaBrokers     []string `env:"APP_KAFKA_BROKERS"`
	ElasticAddresses []string `env:"APP_ELASTIC_ADDRESSES"`
	Hostname         string   `env:"HOSTNAME"`
}

type Worker func(ctx context.Context) error

func Run(ctx context.Context) error {
	var conf Config
	if err := env.Parse(&conf); err != nil {
		log.Panicln("parse config:", err)
	}
	log.SetPrefix(conf.Hostname + " ")

	db, err := NewDB(ctx, conf.PgUser, conf.PgPassword, conf.PgHost, conf.PgDatabase)
	if err != nil {
		return fmt.Errorf("new db: %w", err)
	}
	repo := NewSearchRepository(db)
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
	eg.Wait()
	return nil
}
