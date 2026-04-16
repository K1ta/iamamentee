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

// productSearcher позволяет не зависеть от конкретной реализации поискового хранилища.
type productSearcher interface {
	Search(ctx context.Context, query domain.SearchQuery) ([]int64, error)
}

type SearchHandler struct {
	repo  domain.SearchRepository
	store productSearcher
}

func NewSearchHandler(repo domain.SearchRepository, store productSearcher) *SearchHandler {
	return &SearchHandler{repo: repo, store: store}
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

	query := domain.SearchQuery{
		Name:      req.Name,
		PriceFrom: req.PriceFrom,
		PriceTo:   req.PriceTo,
	}

	productIDs, err := h.store.Search(r.Context(), query)
	if err != nil {
		log.Println("Elastic failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(productIDs) == 0 {
		w.Write([]byte("[]"))
		return
	}
	products, err := h.repo.ListByIDs(r.Context(), productIDs)
	if err != nil {
		log.Println("ListByFilter failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(products); err != nil {
		log.Println("failed to write response:", err)
	}
}
