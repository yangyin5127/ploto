package ploto

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/yangyin5127/sqlstring"
)

type DB struct {
	*sql.DB
	LogSql bool
	logger LoggerInterface
}

type RowsResult struct {
	*sql.Rows
	LastError error
}

type RowResult struct {
	rows      *sql.Rows
	LastError error
}

// RawDB return the *sql.DB
func (db *DB) RawDB() *sql.DB {
	return db.DB
}

// Query executes a query that returns RowsResult, typically a SELECT.
// The args are for any placeholder parameters in the query.
func (db *DB) Query(query string, args ...interface{}) *RowsResult {
	ctx := context.Background()
	return db.QueryContext(ctx, query, args...)

}

// QueryContext executes a query that returns RowsResult, typically a SELECT.
// The args are for any placeholder parameters in the query.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) *RowsResult {
	if db.LogSql {
		db.logger.Info(ctx, "QueryContext sql:%s", sqlstring.Format(query, args...))
	}
	rs, err := db.DB.QueryContext(ctx, query, args...)
	return &RowsResult{rs, err}
}

// QueryRowContext executes a query that is expected to return at most one row.
// QueryRowContext always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *RowResult {
	if db.LogSql {
		db.logger.Info(ctx, "QueryRowContext sql:%s", sqlstring.Format(query, args...))
	}
	rows, err := db.DB.QueryContext(ctx, query, args...)
	return &RowResult{rows: rows, LastError: err}
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (db *DB) QueryRow(query string, args ...interface{}) *RowResult {
	return db.QueryRowContext(context.Background(), query, args...)
}

// Exec executes a query without returning any rows.
// The args are for any placeholder parameters in the query
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {

	return db.ExecContext(context.Background(), query, args...)
}

// ExecContext executes a query without returning any rows.
// The args are for any placeholder parameters in the query.
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if db.LogSql {
		db.logger.Info(ctx, "ExecContext sql:%s", sqlstring.Format(query, args...))
	}

	return db.DB.ExecContext(ctx, query, args...)
}

// Close returns the connection to the connection pool
func (r RowsResult) Close() error {
	return r.Rows.Close()
}

// Scan
func (r *RowsResult) Scan(dest interface{}) error {

	if r.LastError != nil {
		return r.LastError
	}

	if r.Err() != nil {
		return r.Err()
	}
	err := ScanResult(r.Rows, dest)
	return err
}

// Raw
func (r *RowsResult) Raw() (*sql.Rows, error) {
	return r.Rows, r.LastError
}

// RowResult return the error of RowResult
func (r *RowResult) Err() error {
	return r.LastError
}

// Scan RowResult's scan
func (r *RowResult) Scan(dest interface{}) error {

	if r.LastError != nil {
		return r.LastError
	}

	if r.Err() != nil {
		return r.Err()
	}

	if r.rows.Err() != nil {
		return r.rows.Err()
	}
	defer r.rows.Close()

	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	err := Scan(r.rows, dest)

	if err != nil {
		return err
	}

	// Make sure the query can be processed to completion with no errors.
	return r.rows.Close()
}

// Init init all the database clients
func Open(configs DialectConfig, log LoggerInterface) (*Dialect, error) {
	dialect := &Dialect{}
	dialect.Clients = make(map[string]*DB)
	dialect.Configs = configs

	if log == nil {
		dialect.logger = DefaultLogger{}
	} else {
		dialect.logger = log
	}

	for k := range configs.Clients {

		db, err := dialect.CreateClient(k)
		if err != nil {
			return nil, err
		}
		db.logger = dialect.logger
		dialect.Clients[k] = db
	}

	return dialect, nil
}

// Begin starts a transaction. The default isolation level is dependent on
// the driver.
func (db *DB) Begin() (*Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

// BeginTx starts a transaction.
//
// The provided context is used until the transaction is committed or rolled back.
// If the context is canceled, the sql package will roll back
// the transaction. Tx.Commit will return an error if the context provided to
// BeginTx is canceled.
//
// The provided TxOptions is optional and may be nil if defaults should be used.
// If a non-default isolation level is used that the driver doesn't support,
// an error will be returned.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {

	transactionID := ""
	if db.LogSql {

		transactionID = uuid.New().String()
		if db.LogSql {
			db.logger.Info(ctx, "Executing (%s): START TRANSACTION;", transactionID)
		}
	}

	rawTx, err := db.DB.BeginTx(ctx, opts)
	return &Tx{Tx: rawTx, TransactionID: transactionID, DB: db, TransactionCtx: ctx}, err
}
