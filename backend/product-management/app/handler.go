package app

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type (
	CreateProductRequest struct {
		Name  string `json:"name"`
		Price int64  `json:"price"`
	}
)

type ProductHandler struct {
	repo     *ProductRepository
	producer *KafkaProductProducer
}

func NewProductHandler(repo *ProductRepository, producer *KafkaProductProducer) *ProductHandler {
	return &ProductHandler{repo: repo, producer: producer}
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var req CreateProductRequest
	if err := dec.Decode(&req); err != nil {
		log.Println("decode CreateProductRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	userID, ok := r.Context().Value(UserIDContextKey{}).(int64)
	if !ok {
		log.Println("user id not found in context")
		http.Error(w, "Invalid user id", http.StatusUnauthorized)
		return
	}
	product, err := NewProduct(userID, req.Name, req.Price)
	if err != nil {
		log.Println("failed to create new product:", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err := h.repo.Create(r.Context(), product); err != nil {
		log.Println("failed to create product in db:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := h.producer.ProduceEvent(r.Context(), KafkaProductEventTypeCreated, product); err != nil {
		log.Println("failed to produce product event to kafka:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	respBody, err := json.Marshal(product)
	if err != nil {
		log.Println("failed to marshal product:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(respBody)
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		log.Println("failed to parse product id from url:", err)
		http.Error(w, "Invalid product id", http.StatusBadRequest)
		return
	}
	product, err := h.repo.GetByID(r.Context(), productID)
	if err != nil {
		log.Println("failed to get product from db:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	respBody, err := json.Marshal(product)
	if err != nil {
		log.Println("failed to marshal product:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(respBody)
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey{}).(int64)
	if !ok {
		log.Println("user id not found in context")
		http.Error(w, "Invalid user id", http.StatusUnauthorized)
		return
	}
	products, err := h.repo.List(r.Context(), userID)
	if err != nil {
		log.Println("failed to get products from db:", err)
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
