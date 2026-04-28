package orders

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type OrderDeliveryService interface {
	CompleteNextOrder(ctx context.Context) (bool, error)
}

type OrdersWorker struct {
	service         OrderDeliveryService
	pauseWhenNoWork time.Duration
}

func NewOrdersWorker(service OrderDeliveryService, pauseWhenNoWork time.Duration) *OrdersWorker {
	return &OrdersWorker{service: service, pauseWhenNoWork: pauseWhenNoWork}
}

func (w *OrdersWorker) Run(ctx context.Context) error {
	for {
		hadWork, err := w.service.CompleteNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("orders worker error:", err)
		} else if hadWork {
			continue
		}

		select {
		case <-time.After(w.pauseWhenNoWork):
		case <-ctx.Done():
			return nil
		}
	}
}
