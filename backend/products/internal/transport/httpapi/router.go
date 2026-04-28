package httpapi

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *SearchHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo)}))
	r.Get("/products/search", handler.Search)
	return r
}
