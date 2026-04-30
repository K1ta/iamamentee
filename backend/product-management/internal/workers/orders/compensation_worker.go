package orders

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	uuid "github.com/satori/go.uuid"
)

type CompensationService interface {
	CompensateNextOrder(ctx context.Context) (bool, error)
}

type CompensationWorker struct {
	service         CompensationService
	pauseWhenNoWork time.Duration
}

func NewCompensationWorker(service CompensationService, pauseWhenNoWork time.Duration) *CompensationWorker {
	return &CompensationWorker{service: service, pauseWhenNoWork: pauseWhenNoWork}
}

func (w *CompensationWorker) Run(ctx context.Context) error {
	for {
		hadWork, err := w.service.CompensateNextOrder(context.WithValue(ctx, middleware.RequestIDKey, uuid.NewV4().String()))
		if err != nil {
			log.Println("compensation worker error:", err)
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
