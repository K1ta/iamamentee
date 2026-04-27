package orders

import (
	"context"
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

// CompleteOrder отправляет запрос на завершение заказа в orders.
//
// TODO: реализовать HTTP-запрос к orders.
func (c *Client) CompleteOrder(_ context.Context, _ int64) error {
	return nil
}
