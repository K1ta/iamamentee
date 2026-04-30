package failing

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type OrderPaymentService interface {
	FailNextOrder(ctx context.Context) (bool, error)
}

type FailingWorker struct {
	service         OrderPaymentService
	pauseWhenNoWork time.Duration
}

func NewFailingWorker(service OrderPaymentService, pauseWhenNoWork time.Duration) *FailingWorker {
	return &FailingWorker{service: service, pauseWhenNoWork: pauseWhenNoWork}
}

func (w *FailingWorker) Run(ctx context.Context) error {
	for {
		hadWork, err := w.service.FailNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("failing worker error:", err)
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
