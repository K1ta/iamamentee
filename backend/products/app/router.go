package app

import (
	"github.com/go-chi/chi"
)

func NewRouter(handler *SearchHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/products/search", handler.Search)
	return r
}
