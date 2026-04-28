package orders

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type PaymentService interface {
	RequestPaymentForNextOrder(ctx context.Context) (bool, error)
}

type PaymentWorker struct {
	service         PaymentService
	pauseWhenNoWork time.Duration
}

func NewPaymentWorker(service PaymentService, pauseWhenNoWork time.Duration) *PaymentWorker {
	return &PaymentWorker{service: service, pauseWhenNoWork: pauseWhenNoWork}
}

func (w *PaymentWorker) Run(ctx context.Context) error {
	for {
		hadWork, err := w.service.RequestPaymentForNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("payment worker error:", err)
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
