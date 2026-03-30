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
	Listen       string   `env:"APP_LISTEN"`
	PgHost       string   `env:"APP_PG_HOST"`
	PgDatabase   string   `env:"APP_PG_DATABASE"`
	PgUser       string   `env:"APP_PG_USER"`
	PgPassword   string   `env:"APP_PG_PASSWORD"`
	KafkaBrokers []string `env:"APP_KAFKA_BROKERS"`
	Hostname     string   `env:"HOSTNAME"`
}

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
	repo := NewProductRepository(db)
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
