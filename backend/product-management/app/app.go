package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env"
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
	handler := NewProductHandler(repo, producer)
	router := NewRouter(handler)
	server := http.Server{
		Addr:    conf.Listen,
		Handler: router,
	}
	errCh := make(chan error)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("run server: %w", err)
		}
	}()

	log.Println("service is running")
	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.Println("caught error:", err)
	}
	log.Println("stopping service")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("shutdown server: %w", err)
	}
	if err := producer.Close(); err != nil {
		return fmt.Errorf("close kafka: %w", err)
	}
	return nil
}
