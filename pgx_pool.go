package pgxtransactor

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxPool struct {
	pool   *pgxpool.Pool
	tracer Tracer
}

// Transactor represents pgx transactor behavior.
//
//go:generate mockery --name=Transactor --case underscore --output=./mocks
type Transactor interface {
	WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error
	WithinTransactionWithOptions(ctx context.Context, txFunc func(ctx context.Context) error,
		opts pgx.TxOptions) error
}

// NewPgxPool builds new PgxPool.
func NewPgxPool(p *pgxpool.Pool) *PgxPool {
	return &PgxPool{pool: p, tracer: NewNilTracer()}
}

// WithTracer builds new PgxPool.
func (e *PgxPool) WithTracer(tracer Tracer) *PgxPool {
	e.tracer = tracer
	return e
}

// Pool extracts *pgxpool.Pool from transaction runner.
func (e *PgxPool) Pool() *pgxpool.Pool {
	return e.pool
}

// Exec calls pgxpool.Pool.Exec or pgx.Tx.Exec depending on execution context.
func (e *PgxPool) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	spanFinish := e.tracer.TraceData(sql)
	defer spanFinish()
	return e.Runner(ctx).Exec(ctx, sql, args...)
}

// Query calls pgxpool.Pool.Query or pgx.Tx.Query depending on execution context.
func (e *PgxPool) Query(
	ctx context.Context,
	sql string,
	optionsAndArgs ...interface{},
) (pgx.Rows, error) {
	spanFinish := e.tracer.TraceData(sql)
	defer spanFinish()

	return e.Runner(ctx).Query(ctx, sql, optionsAndArgs...)
}

// Begin calls pgxpool.Pool.Begin or pgx.Tx.Begin depending on execution context.
func (e *PgxPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return e.Runner(ctx).Begin(ctx)
}

// SendBatch calls pgxpool.Pool.SendBatch or pgx.Tx.SendBatch depending on execution context.
func (e *PgxPool) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	spanFinish := e.tracer.TraceData("batch")
	defer spanFinish()

	return e.Runner(ctx).SendBatch(ctx, b)
}

// CopyFrom calls pgxpool.Pool.CopyFrom or pgx.Tx.CopyFrom depending on execution context.
func (e *PgxPool) CopyFrom(
	ctx context.Context,
	ident pgx.Identifier,
	cls []string,
	src pgx.CopyFromSource,
) (int64, error) {
	return e.Runner(ctx).CopyFrom(ctx, ident, cls, src)
}

// QueryRow calls pgxpool.Pool.QueryRow or pgx.Tx.QueryRow depending on execution context.
func (e *PgxPool) QueryRow(ctx context.Context,
	sql string, optionsAndArgs ...interface{},
) pgx.Row {
	spanFinish := e.tracer.TraceData(sql)
	defer spanFinish()

	return e.Runner(ctx).QueryRow(ctx, sql, optionsAndArgs...)
}

// WithinTransaction runs WithinTransactionWithOptions with default tx options.
func (e PgxPool) WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error {
	return withinTransaction(ctx, e.pool, txFunc)
}

// WithinTransactionWithOptions runs logic within transaction passing context with pgx.Tx injected into it,
// so you can retrieve it via Executor function Runner.
func (e PgxPool) WithinTransactionWithOptions(ctx context.Context, txFunc func(ctx context.Context) error,
	opts pgx.TxOptions,
) error {
	return withinTransactionWithOptions(ctx, e.pool, txFunc, opts)
}

// withinTransaction runs WithinTransactionWithOptions with default tx options.
func withinTransaction(ctx context.Context, txInit PgxTransactionInitiator,
	txFunc func(txCtx context.Context) error,
) error {
	return withinTransactionWithOptions(ctx, txInit, txFunc, pgx.TxOptions{})
}

// withinTransactionWithOptions runs logic within transaction passing context with pgx.Tx injected into it.
func withinTransactionWithOptions(ctx context.Context, txInit PgxTransactionInitiator,
	txFunc func(txCtx context.Context) error, opts pgx.TxOptions,
) error {
	tx, err := txInit.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		var txErr error
		if err != nil {
			txErr = tx.Rollback(ctx)
		} else {
			txErr = tx.Commit(ctx)
		}

		if txErr != nil && !errors.Is(txErr, pgx.ErrTxClosed) {
			err = txErr
		}
	}()

	err = txFunc(injectTx(ctx, tx))
	return err
}
