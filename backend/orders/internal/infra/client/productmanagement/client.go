package productmanagement

import (
	"context"
	"math/rand"
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

// GetProductPrices возвращает актуальные цены на товары из product-management.
//
// TODO: реализовать HTTP-запрос к product-management, когда ручка будет готова.
func (c *Client) GetProductPrices(_ context.Context, items []domain.Item) (map[int64]int64, error) {
	prices := make(map[int64]int64, len(items))
	for _, item := range items {
		prices[item.ProductID] = rand.Int63n(10000) + 1
	}
	return prices, nil
}

// CreateReservation создаёт резервацию товаров в product-management.
//
// TODO: реализовать HTTP-запрос к product-management, когда ручка будет готова.
func (c *Client) CreateReservation(_ context.Context, _ *domain.Order) error {
	return nil
}
