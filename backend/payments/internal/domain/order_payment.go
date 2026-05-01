package domain

import "fmt"

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

func (p *OrderPayment) SetPaid() error {
	if p.Status != PaymentStatusCreated {
		return fmt.Errorf("cannot set paid from status %s", p.Status)
	}
	p.Status = PaymentStatusPaid
	return nil
}

func (p *OrderPayment) SetFailing() error {
	if p.Status != PaymentStatusCreated {
		return fmt.Errorf("cannot set failing from status %s", p.Status)
	}
	p.Status = PaymentStatusFailing
	return nil
}

func (p *OrderPayment) SetDone() error {
	if p.Status != PaymentStatusPaid {
		return fmt.Errorf("cannot set done from status %s", p.Status)
	}
	p.Status = PaymentStatusDone
	return nil
}

func (p *OrderPayment) SetFailed() error {
	if p.Status != PaymentStatusFailing {
		return fmt.Errorf("cannot set failed from status %s", p.Status)
	}
	p.Status = PaymentStatusFailed
	return nil
}

func (p *OrderPayment) SetCompensating() error {
	if p.Status != PaymentStatusDone {
		return fmt.Errorf("cannot set compensating from status %s", p.Status)
	}
	p.Status = PaymentStatusCompensating
	return nil
}

func (p *OrderPayment) SetCompensated() error {
	if p.Status != PaymentStatusCompensating {
		return fmt.Errorf("cannot set compensated from status %s", p.Status)
	}
	p.Status = PaymentStatusCompensated
	return nil
}

func (p *OrderPayment) SetCanceled() error {
	if p.Status != PaymentStatusCompensated {
		return fmt.Errorf("cannot set canceled from status %s", p.Status)
	}
	p.Status = PaymentStatusCanceled
	return nil
}
