package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

type DeliveryHandler struct{}

func NewDeliveryHandler() *DeliveryHandler {
	return &DeliveryHandler{}
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
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (h *DeliveryHandler) MockSuccess(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (h *DeliveryHandler) MockFail(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode orderRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}
