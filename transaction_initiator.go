package pgxtransactor

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// PgxTransactionInitiator represents transaction initiator.
//
//go:generate mockery --name=PgxTransactionInitiator --case underscore --output=./mocks
type PgxTransactionInitiator interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}
