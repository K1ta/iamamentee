package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
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

type cancelRequest struct {
	OrderID int64 `json:"order_id"`
}

func (c *Client) CancelPayment(ctx context.Context, orderID int64) error {
	body, err := json.Marshal(cancelRequest{OrderID: orderID})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/payments/cancel"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.RequestIDHeader, middleware.GetReqID(ctx))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}
