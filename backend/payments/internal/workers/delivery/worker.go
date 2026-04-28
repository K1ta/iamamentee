package delivery

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type OrderPaymentService interface {
	CreateDeliveryForNextOrder(ctx context.Context) (bool, error)
}

type DeliveryWorker struct {
	service         OrderPaymentService
	pauseWhenNoWork time.Duration
}

func NewDeliveryWorker(service OrderPaymentService, pauseWhenNoWork time.Duration) *DeliveryWorker {
	return &DeliveryWorker{service: service, pauseWhenNoWork: pauseWhenNoWork}
}

func (w *DeliveryWorker) Run(ctx context.Context) error {
	for {
		hadWork, err := w.service.CreateDeliveryForNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("delivery worker error:", err)
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
