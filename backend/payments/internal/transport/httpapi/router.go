package httpapi

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(paymentHandler *PaymentHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.Default()}))
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
