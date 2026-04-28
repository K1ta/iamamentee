package workers

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type OrderService interface {
	StartNextOrder(ctx context.Context) (bool, error)
	FailNextExhaustedOrder(ctx context.Context) (bool, error)
}

type OrderWorker struct {
	svc          OrderService
	pollInterval time.Duration
}

func NewOrderWorker(svc OrderService, pollInterval time.Duration) *OrderWorker {
	return &OrderWorker{svc: svc, pollInterval: pollInterval}
}

func (w *OrderWorker) RunStartOrders(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		worked, err := w.svc.StartNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("start next order:", err)
		}
		if !worked {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(w.pollInterval):
			}
		}
	}
}

func (w *OrderWorker) RunFailOrders(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		worked, err := w.svc.FailNextExhaustedOrder(ctx)
		if err != nil {
			log.Println("fail next exhausted order:", err)
		}
		if !worked {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(w.pollInterval):
			}
		}
	}
}
