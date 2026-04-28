package domain

import "fmt"

type DeliveryStatus string

const (
	DeliveryStatusCreated   DeliveryStatus = "created"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusDone      DeliveryStatus = "done"
	DeliveryStatusFailing   DeliveryStatus = "failing"
	DeliveryStatusFailed    DeliveryStatus = "failed"
)

type OrderDelivery struct {
	OrderID int64
	Status  DeliveryStatus
}

func NewOrderDelivery(orderID int64) *OrderDelivery {
	return &OrderDelivery{
		OrderID: orderID,
		Status:  DeliveryStatusCreated,
	}
}

func (d *OrderDelivery) SetDelivered() error {
	if d.Status != DeliveryStatusCreated {
		return fmt.Errorf("cannot set delivered from status %s", d.Status)
	}
	d.Status = DeliveryStatusDelivered
	return nil
}

func (d *OrderDelivery) SetFailing() error {
	if d.Status != DeliveryStatusCreated {
		return fmt.Errorf("cannot set failing from status %s", d.Status)
	}
	d.Status = DeliveryStatusFailing
	return nil
}

func (d *OrderDelivery) SetDone() error {
	if d.Status != DeliveryStatusDelivered {
		return fmt.Errorf("cannot set done from status %s", d.Status)
	}
	d.Status = DeliveryStatusDone
	return nil
}
