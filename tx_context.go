package pgxtransactor

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type pgxTxKey struct{}

// injectTx injects pgx.Tx into context.
func injectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, pgxTxKey{}, tx)
}

// extractTx retrieves pgx.Tx from context.
func extractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(pgxTxKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}
