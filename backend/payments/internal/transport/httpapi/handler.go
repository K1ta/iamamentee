package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type PaymentService interface {
	Create(ctx context.Context, orderID int64, amount float64) error
}

type PaymentHandler struct {
	service PaymentService
}

func NewPaymentHandler(service PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

type paymentRequest struct {
	OrderID int64   `json:"order_id"`
	Amount  float64 `json:"amount"`
}

type orderRequest struct {
	OrderID int64 `json:"order_id"`
}

func (h *PaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req paymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode paymentRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.Create(r.Context(), req.OrderID, req.Amount); err != nil {
		log.Println("create payment failed:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PaymentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (h *PaymentHandler) MockSuccess(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (h *PaymentHandler) MockFail(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}
