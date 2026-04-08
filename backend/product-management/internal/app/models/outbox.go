package models

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
