package domain

type PaymentStatus string

const (
	PaymentStatusCreated      PaymentStatus = "created"
	PaymentStatusPaid         PaymentStatus = "paid"
	PaymentStatusDone         PaymentStatus = "done"
	PaymentStatusCompensating PaymentStatus = "compensating"
	PaymentStatusCompensated  PaymentStatus = "compensated"
	PaymentStatusCanceled     PaymentStatus = "canceled"
	PaymentStatusFailing      PaymentStatus = "failing"
	PaymentStatusFailed       PaymentStatus = "failed"
)

type OrderPayment struct {
	OrderID int64
	Status  PaymentStatus
}

func NewOrderPayment(orderID int64) *OrderPayment {
	return &OrderPayment{
		OrderID: orderID,
		Status:  PaymentStatusCreated,
	}
}
