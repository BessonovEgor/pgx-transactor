package pgxtransactor

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PgxQueryRunner represents query runner behavior.
//
//go:generate mockery --name=PgxQueryRunner --case underscore --output=./mocks
type PgxQueryRunner interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, ident pgx.Identifier, cls []string, src pgx.CopyFromSource) (int64, error)
}

// Runner extracts query runner from context, if pgx.Tx is injected into context
// it is returned and pgxpool.Pool otherwise.
func (e *PgxPool) Runner(ctx context.Context) PgxQueryRunner {
	tx := extractTx(ctx)
	if tx != nil {
		return tx
	}
	return e.pool
}
