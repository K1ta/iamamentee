package postgres

import (
	"context"
	"database/sql"
	"errors"
	"product-management/internal/app/models"
	"product-management/internal/infra/storage"
	"product-management/internal/pkg/tx"
	"strconv"
)

type UnitOfWorkManager struct {
	shards storage.Shards[*sql.DB]
}

func NewUnitOfWorkManager(shards storage.Shards[*sql.DB]) (*UnitOfWorkManager, error) {
	if len(shards) == 0 {
		return nil, errors.New("empty shards")
	}
	return &UnitOfWorkManager{shards: shards}, nil
}

func (m *UnitOfWorkManager) RunForUser(ctx context.Context, userID int64, f func(uow *UnitOfWork) error) error {
	_, db := m.shards.Get(strconv.FormatInt(userID, 10))
	return tx.Run(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		return f(&UnitOfWork{tx: tx})
	})
}

type UnitOfWork struct {
	tx *sql.Tx
}

func (uow *UnitOfWork) CreateProduct(ctx context.Context, product *models.Product) error {
	return NewProductRepository(uow.tx).Create(ctx, product)
}

func (uow *UnitOfWork) CreateOutboxEvent(ctx context.Context, event *models.OutboxEvent, maxAttempts int) error {
	return NewOutboxRepository(uow.tx).Create(ctx, event, maxAttempts)
}
