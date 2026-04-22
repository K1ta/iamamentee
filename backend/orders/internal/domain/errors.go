package domain

import "errors"

var ErrOrderNotFound = errors.New("order not found")

var ErrOrderConflict = errors.New("updating order concurrently")

var ErrNoOrderToProcess = errors.New("no order to process")
