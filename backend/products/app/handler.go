package app

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type (
	SearchRequest struct {
		Name      string
		PriceFrom int64
		PriceTo   int64
	}
)

type SearchHandler struct {
	repo *SearchRepository
}

func NewSearchHandler(repo *SearchRepository) *SearchHandler {
	return &SearchHandler{repo: repo}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	var err error

	var priceFrom int64
	if from := r.URL.Query().Get("from"); from != "" {
		priceFrom, err = strconv.ParseInt(from, 10, 64)
		if err != nil {
			log.Println("failed to parse from price from url:", err)
			http.Error(w, "Invalid param 'from'", http.StatusBadRequest)
			return
		}
	}

	var priceTo int64
	if to := r.URL.Query().Get("to"); to != "" {
		priceTo, err = strconv.ParseInt(to, 10, 64)
		if err != nil {
			log.Println("failed to parse to price from url:", err)
			http.Error(w, "Invalid param 'to'", http.StatusBadRequest)
			return
		}
	}

	req.PriceFrom = priceFrom
	req.PriceTo = priceTo
	req.Name = r.URL.Query().Get("name")

	log.Println("got search request:", req)

	products, err := h.repo.ListByFilter(r.Context(), &req)
	if err != nil {
		log.Println("ListByFilter failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	respBody, err := json.Marshal(products)
	if err != nil {
		log.Println("failed to marshal products:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(respBody)
}
