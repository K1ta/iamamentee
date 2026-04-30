package orders

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type CancellationService interface {
	CancelNextOrder(ctx context.Context) (bool, error)
}

type CancellationWorker struct {
	service         CancellationService
	pauseWhenNoWork time.Duration
}

func NewCancellationWorker(service CancellationService, pauseWhenNoWork time.Duration) *CancellationWorker {
	return &CancellationWorker{service: service, pauseWhenNoWork: pauseWhenNoWork}
}

func (w *CancellationWorker) Run(ctx context.Context) error {
	for {
		hadWork, err := w.service.CancelNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("cancellation worker error:", err)
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
