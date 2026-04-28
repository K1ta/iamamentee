package orders

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type OrderService interface {
	ReserveNextOrder(ctx context.Context) (bool, error)
}

type ReservationWorker struct {
	service         OrderService
	pauseWhenNoWork time.Duration
}

func NewReservationWorker(service OrderService, pauseWhenNoWork time.Duration) *ReservationWorker {
	return &ReservationWorker{service: service, pauseWhenNoWork: pauseWhenNoWork}
}

func (w *ReservationWorker) Run(ctx context.Context) error {
	for {
		hadWork, err := w.service.ReserveNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("reservation worker error:", err)
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
