package app

import (
	"context"
	"log"
	"products/internal/transport/events"
	"products/internal/transport/httpapi"

	"golang.org/x/sync/errgroup"
)

type ServerApp struct {
	kafkaConsumer *events.ProductEventConsumer
	httpServer    *httpapi.Server
}

func NewServerApp(kafkaConsumer *events.ProductEventConsumer, httpServer *httpapi.Server) *ServerApp {
	return &ServerApp{kafkaConsumer: kafkaConsumer, httpServer: httpServer}
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
	return eg.Wait()
}
