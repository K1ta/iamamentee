package app

import (
	"context"
	"log"
	"orders/internal/transport/httpapi"

	"golang.org/x/sync/errgroup"
)

type ServerApp struct {
	httpServer *httpapi.Server
}

func NewServerApp(httpServer *httpapi.Server) *ServerApp {
	return &ServerApp{httpServer: httpServer}
}

func (a *ServerApp) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return a.httpServer.Run(egCtx)
	})
	log.Println("server is running")
	return eg.Wait()
}
