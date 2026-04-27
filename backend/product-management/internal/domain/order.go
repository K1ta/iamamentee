package domain

import "errors"

type OrderStatus string

const (
	OrderStatusCreated      OrderStatus = "created"
	OrderStatusReserved     OrderStatus = "reserved"
	OrderStatusDone         OrderStatus = "done"
	OrderStatusCompensating OrderStatus = "compensating"
	OrderStatusCompensated  OrderStatus = "compensated"
	OrderStatusCanceling    OrderStatus = "canceling"
	OrderStatusCanceled     OrderStatus = "canceled"
	OrderStatusFailed       OrderStatus = "failed"
)

type Order struct {
	ID     int64
	Status OrderStatus
}

func NewOrder(id int64) (*Order, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	return &Order{
		ID:     id,
		Status: OrderStatusCreated,
	}, nil
}
