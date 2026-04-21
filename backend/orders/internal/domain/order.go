package domain

import (
	"errors"
	"fmt"
)

type Status string

const (
	StatusCreated    Status = "created"
	StatusConfirmed  Status = "confirmed"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusCanceled   Status = "canceled"
	StatusFailed     Status = "failed"
)

type Item struct {
	ProductID int64
	Amount    int
	Price     int64
}

type Order struct {
	ID     int64
	UserID int64
	Status Status
	Items  []Item
}

func NewOrder(id int64, userID int64, items []Item) (*Order, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	if len(items) == 0 {
		return nil, errors.New("items are empty")
	}
	return &Order{
		ID:     id,
		UserID: userID,
		Status: StatusCreated,
		Items:  items,
	}, nil
}

// Confirm фиксирует цены продуктов в заказе.
// prices - маппинг product_id -> price.
// Возвращает ошибку, если есть продукт без цены или цена для продукта, которого нет в заказе.
func (o *Order) Confirm(prices map[int64]int64) error {
	if o.Status != StatusCreated {
		return fmt.Errorf("cannot confirm order in status %s", o.Status)
	}

	itemIDs := make(map[int64]struct{}, len(o.Items))
	for _, item := range o.Items {
		itemIDs[item.ProductID] = struct{}{}
	}
	for productID := range prices {
		if _, ok := itemIDs[productID]; !ok {
			return fmt.Errorf("price provided for unknown product %d", productID)
		}
	}

	for i, item := range o.Items {
		price, ok := prices[item.ProductID]
		if !ok {
			return fmt.Errorf("price for product %d not found", item.ProductID)
		}
		o.Items[i].Price = price
	}

	o.Status = StatusConfirmed
	return nil
}

func (o *Order) SetProcessing() error {
	if o.Status != StatusConfirmed {
		return fmt.Errorf("cannot set processing from status %s", o.Status)
	}
	o.Status = StatusProcessing
	return nil
}

func (o *Order) SetCanceled() error {
	if o.Status != StatusProcessing {
		return fmt.Errorf("cannot cancel from status %s", o.Status)
	}
	o.Status = StatusCanceled
	return nil
}

func (o *Order) SetCompleted() error {
	if o.Status != StatusProcessing {
		return fmt.Errorf("cannot complete from status %s", o.Status)
	}
	o.Status = StatusCompleted
	return nil
}

func (o *Order) SetFailed() error {
	if o.Status != StatusCreated && o.Status != StatusConfirmed {
		return fmt.Errorf("cannot fail from status %s", o.Status)
	}
	o.Status = StatusFailed
	return nil
}
