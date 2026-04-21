package storage

import (
	"context"
	"net/http"
	"orders/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// CreateReservation резервирует товары на складе для заказа.
//
// TODO: реализовать HTTP-запрос к storage, когда ручка будет готова.
func (c *Client) CreateReservation(_ context.Context, order *domain.Order) error {
	return nil
}
