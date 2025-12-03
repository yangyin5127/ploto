package ploto

import (
	"context"
	"database/sql"

	"github.com/yangyin5127/sqlstring"
)

type Tx struct {
	*sql.Tx
	DB             *DB
	TransactionID  string
	TransactionCtx context.Context
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {

	if tx.DB.LogSql {
		tx.DB.logger.Info(tx.TransactionCtx, "Executing (%s): COMMIT;", tx.TransactionID)

	}
	return tx.Tx.Commit()
}

// Exec executes a query that doesn't return rows.
// For example: an INSERT and UPDATE.
func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {

	return tx.ExecContext(context.Background(), query, args...)
}

// ExecContext executes a query that doesn't return rows.
// For example: an INSERT and UPDATE.
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if tx.DB.LogSql {
		tx.DB.logger.Info(ctx, "Executing (%s):%s", tx.TransactionID, sqlstring.Format(query, args...))
	}
	return tx.Tx.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows, typically a SELECT.
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) *RowsResult {
	if tx.DB.LogSql {
		tx.DB.logger.Info(ctx, "Query (%s):%s", tx.TransactionID, sqlstring.Format(query, args...))
	}
	rs, err := tx.Tx.QueryContext(ctx, query, args...)
	return &RowsResult{rs, err}
}

// Query executes a query that returns rows, typically a SELECT.
func (tx *Tx) Query(query string, args ...interface{}) *RowsResult {
	return tx.QueryContext(context.Background(), query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (tx *Tx) QueryRow(query string, args ...interface{}) *RowResult {

	return tx.QueryRowContext(context.Background(), query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row.
// QueryRowContext always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *RowResult {
	if tx.DB.LogSql {
		tx.DB.logger.Info(ctx, "Query (%s):%s", tx.TransactionID, sqlstring.Format(query, args...))
	}
	rows, err := tx.Tx.QueryContext(ctx, query, args...)

	return &RowResult{rows: rows, LastError: err}
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback() error {
	err := tx.Tx.Rollback()
	if err != nil && err == sql.ErrTxDone {
		return err
	}
	if tx.DB.LogSql {
		tx.DB.logger.Info(tx.TransactionCtx, "Executing (%s): ROLLBACK", tx.TransactionID)
	}
	return nil
}
