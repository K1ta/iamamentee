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

// ReserveProducts резервирует товары в product-management и возвращает их актуальные цены.
//
// TODO: реализовать HTTP-запрос к product-management, когда ручка будет готова.
func (c *Client) ReserveProducts(_ context.Context, order *domain.Order) (map[int64]int64, error) {
	prices := make(map[int64]int64, len(order.Items))
	for _, item := range order.Items {
		prices[item.ProductID] = rand.Int63n(10000) + 1
	}
	return prices, nil
}
