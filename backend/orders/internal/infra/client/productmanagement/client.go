package productmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"orders/internal/domain"
	"strconv"
	"strings"

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

func (c *Client) GetProductPrices(ctx context.Context, items []domain.Item) (map[int64]int64, error) {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = strconv.FormatInt(item.ProductID, 10)
	}

	url := c.baseURL + "/product/prices?ids=" + strings.Join(ids, ",")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	// TODO в product-management нужен только для авторизации, в логике не участвует.
	// Убрать, когда перенесем ручку в internal
	req.Header.Set("X-User-ID", "1")
	req.Header.Set(middleware.RequestIDHeader, middleware.GetReqID(ctx))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var prices map[int64]int64
	if err := json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return prices, nil
}

type createReservationRequest struct {
	OrderID int64                   `json:"order_id"`
	Items   []createReservationItem `json:"items"`
}

type createReservationItem struct {
	ProductID int64 `json:"product_id"`
	Amount    int   `json:"amount"`
}

func (c *Client) CreateReservation(ctx context.Context, order *domain.Order) error {
	reqBody := createReservationRequest{
		OrderID: order.ID,
		Items:   make([]createReservationItem, len(order.Items)),
	}
	for i, item := range order.Items {
		reqBody.Items[i] = createReservationItem{
			ProductID: item.ProductID,
			Amount:    item.Amount,
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/product/reservations/create"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// TODO в product-management нужен только для авторизации, в логике не участвует.
	// Убрать, когда перенесем ручку в internal
	req.Header.Set("X-User-ID", strconv.FormatInt(order.UserID, 10))

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
