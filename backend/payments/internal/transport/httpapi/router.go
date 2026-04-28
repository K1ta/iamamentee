package httpapi

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(paymentHandler *PaymentHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo)}))
	r.Route("/payments", func(r chi.Router) {
		r.Post("/create", paymentHandler.Create)
		r.Post("/cancel", paymentHandler.Cancel)
		r.Route("/mock", func(r chi.Router) {
			r.Post("/success", paymentHandler.MockSuccess)
			r.Post("/fail", paymentHandler.MockFail)
		})
	})
	return r
}
