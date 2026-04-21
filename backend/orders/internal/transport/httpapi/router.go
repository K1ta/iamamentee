package httpapi

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(orderHandler *OrderHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.Default()}))
	r.Route("/orders", func(r chi.Router) {
		r.Post("/create", orderHandler.Create)
		r.Post("/cancel", orderHandler.Cancel)
		r.Post("/complete", orderHandler.Complete)
		r.Get("/", orderHandler.Get)
	})
	return r
}
