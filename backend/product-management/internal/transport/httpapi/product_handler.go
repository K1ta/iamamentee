package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"product-management/internal/domain"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	service ProductService
}

type (
	ProductService interface {
		Create(ctx context.Context, userID int64, name string, price int64) (*domain.Product, error)
		GetByID(ctx context.Context, id, userID int64) (*domain.Product, error)
		List(ctx context.Context, userID int64) ([]domain.Product, error)
		GetPrices(ctx context.Context, ids []int64) (map[int64]int64, error)
	}
)

func NewProductHandler(service ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

type CreateProductRequest struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var req CreateProductRequest
	if err := dec.Decode(&req); err != nil {
		log.Println("decode CreateProductRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	product, err := h.service.Create(r.Context(), MustGetUserID(r.Context()), req.Name, req.Price)
	if err != nil {
		log.Println("failed to create product:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(product); err != nil {
		log.Println("failed to write response:", err)
	}
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		log.Println("failed to parse product id from url:", err)
		http.Error(w, "Invalid product id", http.StatusBadRequest)
		return
	}

	product, err := h.service.GetByID(r.Context(), MustGetUserID(r.Context()), productID)
	if err != nil {
		log.Println("failed to get product by id:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(product); err != nil {
		log.Println("failed to write response:", err)
	}
}

func (h *ProductHandler) Prices(w http.ResponseWriter, r *http.Request) {
	idsStr := r.URL.Query().Get("ids")
	if idsStr == "" {
		http.Error(w, "ids is required", http.StatusBadRequest)
		return
	}

	parts := strings.Split(idsStr, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			http.Error(w, "invalid id: "+part, http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
	}

	prices, err := h.service.GetPrices(r.Context(), ids)
	if err != nil {
		log.Println("failed to get prices:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(prices); err != nil {
		log.Println("failed to write response:", err)
	}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.List(r.Context(), MustGetUserID(r.Context()))
	if err != nil {
		log.Println("failed to list products:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(products); err != nil {
		log.Println("failed to write response:", err)
	}
}
