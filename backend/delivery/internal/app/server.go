package app

import (
	"context"
	"database/sql"
	"delivery/internal/transport/httpapi"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type ServerApp struct {
	dbs        map[string]*sql.DB
	httpServer *httpapi.Server
}

func NewServerApp(dbs map[string]*sql.DB, httpServer *httpapi.Server) *ServerApp {
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := db.Close(); err != nil {
				log.Printf("failed to close %s db: %v", name, err)
			}
		}()
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
