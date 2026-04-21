package productmanagement

import (
	"context"
	"math/rand"
	"net/http"
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

// GetProductPrices возвращает цены для переданных product_id.
// Ошибка, если какой-то продукт невалиден.
//
// TODO: реализовать HTTP-запрос к product-management, когда ручка будет готова.
func (c *Client) GetProductPrices(_ context.Context, productIDs []int64) (map[int64]int64, error) {
	prices := make(map[int64]int64, len(productIDs))
	for _, id := range productIDs {
		prices[id] = rand.Int63n(10000) + 1
	}
	return prices, nil
}
