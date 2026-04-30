package productmanagement

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

type cancelReservationRequest struct {
	OrderID int64 `json:"order_id"`
}

func (c *Client) CancelReservation(ctx context.Context, orderID int64) error {
	body, err := json.Marshal(cancelReservationRequest{OrderID: orderID})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/product/reservations/cancel"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// TODO в product-management нужен только для авторизации, в логике не участвует.
	// Убрать, когда перенесем ручку в internal
	req.Header.Set("X-User-ID", "1")
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
