package app

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ggicci/httpin"
	"github.com/ggicci/httpin/core"
)

type (
	SearchRequest struct {
		Name      string `in:"query=name"`
		PriceFrom int64  `in:"query=from"`
		PriceTo   int64  `in:"query=to"`
	}
)

type SearchHandler struct {
	repo  *SearchRepository
	store *SearchStore
}

func NewSearchHandler(repo *SearchRepository, store *SearchStore) *SearchHandler {
	return &SearchHandler{repo: repo, store: store}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	req, err := httpin.Decode[SearchRequest](r)
	if err != nil {
		msg := "Invalid request"
		if invalidFielError, ok := errors.AsType[*core.InvalidFieldError](err); ok {
			msg = "Invalid param '" + invalidFielError.Key + "'"
		}
		log.Println("failed to parse request:", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Println("got search request:", req)

	productIDs, err := h.store.Search(r.Context(), req)
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
