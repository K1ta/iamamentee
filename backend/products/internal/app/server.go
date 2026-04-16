package app

import (
	"context"
	"log"
	"products/internal/infra/messaging/kafka"
	"products/internal/transport/httpapi"

	"golang.org/x/sync/errgroup"
)

type ServerApp struct {
	kafkaConsumer *kafka.ProductEventConsumer
	httpServer    *httpapi.Server
}

func NewServerApp(kafkaConsumer *kafka.ProductEventConsumer, httpServer *httpapi.Server) *ServerApp {
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
