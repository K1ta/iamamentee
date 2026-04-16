package domain

import "context"

type UnitOfWork interface {
	ProductRepository() ProductRepository
	OutboxRepository() OutboxRepository
	Commit() error
	Rollback() error
}

type UnitOfWorkFactory interface {
	ForUser(ctx context.Context, userID int64) (UnitOfWork, error)
}
