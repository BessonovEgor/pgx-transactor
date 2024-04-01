package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Transactor struct{}

func NewTransactor() *Transactor {
	return &Transactor{}
}

func (t *Transactor) WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error {
	return txFunc(ctx)
}

func (t *Transactor) WithinTransactionWithOptions(
	ctx context.Context,
	txFunc func(ctx context.Context) error,
	_ pgx.TxOptions,
) error {
	return txFunc(ctx)
}
