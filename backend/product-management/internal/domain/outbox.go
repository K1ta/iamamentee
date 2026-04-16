package domain

import "context"

type OutboxEventType = string

const (
	ProductCreated OutboxEventType = "product.created"
)

type OutboxEvent struct {
	ID      int64
	Type    OutboxEventType
	Key     string
	Payload string
}

type OutboxRepository interface {
	Create(ctx context.Context, event *OutboxEvent) error
}
