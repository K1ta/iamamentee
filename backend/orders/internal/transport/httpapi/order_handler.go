package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"orders/internal/domain"
	"strconv"
)

type OrderService interface {
	Create(ctx context.Context, userID int64, items []domain.Item) (*domain.Order, error)
	Cancel(ctx context.Context, orderID int64) error
	Complete(ctx context.Context, orderID int64) error
	GetByID(ctx context.Context, orderID int64) (*domain.Order, error)
}

type OrderHandler struct {
	service OrderService
}

func NewOrderHandler(service OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

type createOrderRequest struct {
	Items []createOrderItem `json:"items"`
}

type createOrderItem struct {
	ProductID int64 `json:"product_id"`
	Amount    int   `json:"amount"`
}

type cancelOrderRequest struct {
	OrderID int64 `json:"order_id"`
}

type completeOrderRequest struct {
	OrderID int64 `json:"order_id"`
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode createOrderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	items := make([]domain.Item, len(req.Items))
	for i, item := range req.Items {
		items[i] = domain.Item{ProductID: item.ProductID, Amount: item.Amount}
	}

	order, err := h.service.Create(r.Context(), MustGetUserID(r.Context()), items)
	if err != nil {
		log.Println("failed to create order:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Println("failed to write response:", err)
	}
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req cancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode cancelOrderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.Cancel(r.Context(), req.OrderID); err != nil {
		log.Println("failed to cancel order:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *OrderHandler) Complete(w http.ResponseWriter, r *http.Request) {
	var req completeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode completeOrderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.Complete(r.Context(), req.OrderID); err != nil {
		log.Println("failed to complete order:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.URL.Query().Get("order_id"), 10, 64)
	if err != nil {
		log.Println("failed to parse order_id from query:", err)
		http.Error(w, "invalid order_id", http.StatusBadRequest)
		return
	}

	order, err := h.service.GetByID(r.Context(), orderID)
	if errors.Is(err, domain.ErrOrderNotFound) {
		log.Println("order not found")
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("failed to get order:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Println("failed to write response:", err)
	}
}
