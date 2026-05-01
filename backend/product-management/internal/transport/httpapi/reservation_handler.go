package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"product-management/internal/service"
)

type ReservationService interface {
	Create(ctx context.Context, orderID int64, items []service.ReservationItem) error
	Cancel(ctx context.Context, orderID int64) error
}

type ReservationHandler struct {
	service ReservationService
}

func NewReservationHandler(svc ReservationService) *ReservationHandler {
	return &ReservationHandler{service: svc}
}

type createReservationRequest struct {
	OrderID int64                   `json:"order_id"`
	Items   []createReservationItem `json:"items"`
}

type createReservationItem struct {
	ProductID int64 `json:"product_id"`
	Amount    int   `json:"amount"`
}

type cancelReservationRequest struct {
	OrderID int64 `json:"order_id"`
}

func (h *ReservationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req cancelReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode cancelReservationRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.Cancel(r.Context(), req.OrderID); err != nil {
		log.Println("failed to cancel reservation:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ReservationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode createReservationRequest failed:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	items := make([]service.ReservationItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = service.ReservationItem{ProductID: item.ProductID, Amount: item.Amount}
	}

	if err := h.service.Create(r.Context(), req.OrderID, items); err != nil {
		log.Println("failed to create reservation:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
