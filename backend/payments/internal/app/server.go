package app

import (
	"context"
	"database/sql"
	"log"
	"payments/internal/transport/httpapi"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type worker interface {
	Run(ctx context.Context) error
}

type ServerApp struct {
	dbs        map[string]*sql.DB
	httpServer *httpapi.Server
	workers    []worker
}

func NewServerApp(dbs map[string]*sql.DB, httpServer *httpapi.Server, workers ...worker) *ServerApp {
	return &ServerApp{dbs: dbs, httpServer: httpServer, workers: workers}
}

func (app *ServerApp) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return app.httpServer.Run(egCtx)
	})
	for _, w := range app.workers {
		eg.Go(func() error {
			return w.Run(egCtx)
		})
	}
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
	closeDBs(wg, app.dbs)

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
