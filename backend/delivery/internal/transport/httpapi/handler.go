package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type DeliveryService interface {
	Create(ctx context.Context, orderID int64) error
	MockSuccess(ctx context.Context, orderID int64) error
	MockFail(ctx context.Context, orderID int64) error
}

type DeliveryHandler struct {
	service DeliveryService
}

func NewDeliveryHandler(service DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{service: service}
}

type orderRequest struct {
	OrderID int64 `json:"order_id"`
}

func (h *DeliveryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.Create(r.Context(), req.OrderID); err != nil {
		log.Println("create delivery failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *DeliveryHandler) MockSuccess(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.MockSuccess(r.Context(), req.OrderID); err != nil {
		log.Println("mock success failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *DeliveryHandler) MockFail(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.MockFail(r.Context(), req.OrderID); err != nil {
		log.Println("mock fail failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
