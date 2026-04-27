package domain

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
