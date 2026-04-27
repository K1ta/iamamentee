package domain

import "errors"

var ErrOrderPaymentNotFound = errors.New("order payment not found")

var ErrNoOrderPaymentToProcess = errors.New("no order payment to process")
