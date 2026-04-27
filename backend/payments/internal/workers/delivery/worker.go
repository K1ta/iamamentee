package delivery

import (
	"context"
	"log"
	"time"
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
		hadWork, err := w.service.CreateDeliveryForNextOrder(ctx)
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
