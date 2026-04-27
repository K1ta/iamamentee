package delivery

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

// CreateDelivery отправляет запрос на создание доставки в delivery.
//
// TODO: реализовать HTTP-запрос к delivery, когда сервис будет готов.
func (c *Client) CreateDelivery(_ context.Context, _ int64) error {
	return nil
}
