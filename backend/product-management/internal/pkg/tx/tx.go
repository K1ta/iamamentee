package tx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func Run(ctx context.Context, db *sql.DB, f func(context.Context, *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	err = f(ctx, tx)
	if err == nil {
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
		return nil
	}

	err = fmt.Errorf("do in tx: %w", err)
	rollbackErr := tx.Rollback()
	if rollbackErr != nil {
		rollbackErr = fmt.Errorf("rollback tx: %w", rollbackErr)
		return errors.Join(err, rollbackErr)
	}
	return err
}
