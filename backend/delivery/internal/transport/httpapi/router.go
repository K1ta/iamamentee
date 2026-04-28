package httpapi

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(deliveryHandler *DeliveryHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo)}))
	r.Route("/delivery", func(r chi.Router) {
		r.Post("/create", deliveryHandler.Create)
		r.Route("/mock", func(r chi.Router) {
			r.Post("/success", deliveryHandler.MockSuccess)
			r.Post("/fail", deliveryHandler.MockFail)
		})
	})
	return r
}
