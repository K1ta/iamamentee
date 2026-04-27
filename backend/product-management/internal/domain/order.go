package domain

import (
	"errors"
	"fmt"
)

var ErrNoOrderFound = errors.New("no order found")

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

func (o *Order) SetReserved() error {
	if o.Status != OrderStatusCreated {
		return fmt.Errorf("cannot set reserved from status %s", o.Status)
	}
	o.Status = OrderStatusReserved
	return nil
}

func (o *Order) SetDone() error {
	if o.Status != OrderStatusReserved {
		return fmt.Errorf("cannot set done from status %s", o.Status)
	}
	o.Status = OrderStatusDone
	return nil
}
