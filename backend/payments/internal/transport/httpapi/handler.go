package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

type PaymentHandler struct{}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{}
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
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
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
