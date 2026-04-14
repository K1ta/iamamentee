package app

import (
	"context"
	"database/sql"
	"log"
	"product-management/internal/app/transport/http"
	"product-management/internal/infra/config"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type ServerApp struct {
	dbs        map[config.PostgresName]*sql.DB
	httpServer *http.HttpServer
}

func NewServerApp(dbs map[config.PostgresName]*sql.DB, httpServer *http.HttpServer) *ServerApp {
	return &ServerApp{dbs: dbs, httpServer: httpServer}
}

func (app *ServerApp) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return app.httpServer.Run(egCtx)
	})
	log.Println("server is running")
	err := eg.Wait()
	app.shutdown()
	return err
}

func (app *ServerApp) shutdown() {
	log.Println("shutting down server")
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
