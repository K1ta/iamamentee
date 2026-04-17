package app

import (
	"context"
	"database/sql"
	"log"
	"products/internal/transport/events"
	"products/internal/transport/httpapi"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type ServerApp struct {
	dbs           map[string]*sql.DB
	kafkaConsumer *events.ProductEventConsumer
	httpServer    *httpapi.Server
}

func NewServerApp(
	dbs map[string]*sql.DB,
	kafkaConsumer *events.ProductEventConsumer,
	httpServer *httpapi.Server,
) *ServerApp {
	return &ServerApp{dbs: dbs, kafkaConsumer: kafkaConsumer, httpServer: httpServer}
}

func (a *ServerApp) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return a.kafkaConsumer.Run(egCtx)
	})
	eg.Go(func() error {
		return a.httpServer.Run(egCtx)
	})
	log.Println("service is running")
	err := eg.Wait()
	a.shutdown()
	return err
}

func (a *ServerApp) shutdown() {
	log.Println("shutting down server")
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

func closeDBs(wg *sync.WaitGroup, dbs map[string]*sql.DB) {
	for name, db := range dbs {
		wg.Go(func() {
			if err := db.Close(); err != nil {
				log.Printf("failed to close %s db: %v", name, err)
			}
		})
	}
}
