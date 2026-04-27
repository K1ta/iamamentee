package payments

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

// RequestPayment отправляет запрос на оплату в payments.
//
// TODO: реализовать HTTP-запрос к payments, когда сервис будет готов.
func (c *Client) RequestPayment(_ context.Context, _ int64) error {
	return nil
}
