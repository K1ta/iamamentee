package app

import (
	"context"
	"log"
	"orders/internal/transport/workers"

	"golang.org/x/sync/errgroup"
)

type WorkerApp struct {
	worker *workers.OrderWorker
}

func NewWorkerApp(worker *workers.OrderWorker) *WorkerApp {
	return &WorkerApp{worker: worker}
}

func (a *WorkerApp) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return a.worker.RunStartOrders(egCtx)
	})
	eg.Go(func() error {
		return a.worker.RunFailOrders(egCtx)
	})
	log.Println("workers are running")
	return eg.Wait()
}
