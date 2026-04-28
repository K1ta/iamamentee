package domain

import "errors"

var (
	ErrNoOrderDeliveryToProcess = errors.New("no order delivery to process")
	ErrOrderDeliveryNotFound    = errors.New("order delivery not found")
)
