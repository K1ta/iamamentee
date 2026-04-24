package domain

import (
	"errors"
	"fmt"
)

type Status string

const (
	StatusCreated    Status = "created"
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

func NewOrder(userID int64, items []Item) (*Order, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	if len(items) == 0 {
		return nil, errors.New("items are empty")
	}
	return &Order{
		UserID: userID,
		Status: StatusCreated,
		Items:  items,
	}, nil
}

// RestoreOrder восстанавливает Order из хранилища.
// Валидирует поля, чтобы гарантировать инварианты агрегата при чтении.
func RestoreOrder(id, userID int64, status Status, items []Item) (*Order, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	if !isValidStatus(status) {
		return nil, fmt.Errorf("unknown status: %s", status)
	}
	if len(items) == 0 {
		return nil, errors.New("items are empty")
	}
	return &Order{
		ID:     id,
		UserID: userID,
		Status: status,
		Items:  items,
	}, nil
}

func isValidStatus(s Status) bool {
	switch s {
	case StatusCreated, StatusProcessing, StatusCompleted, StatusCanceled, StatusFailed:
		return true
	}
	return false
}

// SetProcessing резервирует товары и переводит заказ в статус processing.
// prices — маппинг product_id -> price, полученный от product-management при резервации.
func (o *Order) SetProcessing(prices map[int64]int64) error {
	if o.Status != StatusCreated {
		return fmt.Errorf("cannot set processing from status %s", o.Status)
	}
	for i, item := range o.Items {
		price, ok := prices[item.ProductID]
		if !ok {
			return fmt.Errorf("price for product %d not found", item.ProductID)
		}
		o.Items[i].Price = price
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
	if o.Status != StatusCreated {
		return fmt.Errorf("cannot fail from status %s", o.Status)
	}
	o.Status = StatusFailed
	return nil
}
