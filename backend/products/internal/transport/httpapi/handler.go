package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"products/internal/domain"

	"github.com/ggicci/httpin"
	"github.com/ggicci/httpin/core"
)

type searchRequest struct {
	Name      string `in:"query=name"`
	PriceFrom int64  `in:"query=from"`
	PriceTo   int64  `in:"query=to"`
}

type productService interface {
	Search(ctx context.Context, query domain.SearchQuery) ([]domain.Product, error)
}

type SearchHandler struct {
	svc productService
}

func NewSearchHandler(svc productService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	req, err := httpin.Decode[searchRequest](r)
	if err != nil {
		msg := "Invalid request"
		if invalidFieldError, ok := errors.AsType[*core.InvalidFieldError](err); ok {
			msg = "Invalid param '" + invalidFieldError.Key + "'"
		}
		log.Println("failed to parse request:", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Println("got search request:", req)

	products, err := h.svc.Search(r.Context(), domain.SearchQuery{
		Name:      req.Name,
		PriceFrom: req.PriceFrom,
		PriceTo:   req.PriceTo,
	})
	if err != nil {
		log.Println("search failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(products); err != nil {
		log.Println("failed to write response:", err)
	}
}
