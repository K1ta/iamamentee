package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"product-management/internal/app/domain"
	"product-management/internal/pkg/sharding"
	"strconv"
)

type UnitOfWorkFactory struct {
	shardsPool        *sharding.Pool[*sql.DB]
	outboxMaxAttempts int
}

func NewUnitOfWorkFactory(shards *sharding.Pool[*sql.DB], outboxMaxAttempts int) *UnitOfWorkFactory {
	return &UnitOfWorkFactory{shardsPool: shards, outboxMaxAttempts: outboxMaxAttempts}
}

func (m *UnitOfWorkFactory) ForUser(ctx context.Context, userID int64) (domain.UnitOfWork, error) {
	db := m.shardsPool.Get(strconv.FormatInt(userID, 10))
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	return &UnitOfWork{tx: tx, outboxMaxAttempts: m.outboxMaxAttempts}, nil
}

type UnitOfWork struct {
	tx *sql.Tx
	// UOW знает про параметр для создания OutboxRepository. В идеале он должен отвечать только за
	// проведение транзакции и создавать такие репозитории через фабрики, но здесь это избыточно
	outboxMaxAttempts int
}

func (uow *UnitOfWork) ProductRepository() domain.ProductRepository {
	return NewProductRepository(uow.tx)
}

func (uow *UnitOfWork) OutboxRepository() domain.OutboxRepository {
	return NewOutboxRepository(uow.tx, uow.outboxMaxAttempts)
}

func (uow *UnitOfWork) Commit() error {
	return uow.tx.Commit()
}

func (uow *UnitOfWork) Rollback() error {
	return uow.tx.Rollback()
}
