package httpapi

import "net/http"

type OrderHandler struct{}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *OrderHandler) Complete(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
