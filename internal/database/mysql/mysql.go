package mysql

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/go-sql-driver/mysql" // register driver
	"github.com/koustreak/DatRi/internal/database"
)

// DB implements database.DB for MySQL using database/sql
type DB struct {
	sqlDB *sql.DB
	cfg   *database.Config
}

// New creates a new MySQL DB instance (does not connect yet)
func New(cfg *database.Config) *DB {
	return &DB{cfg: cfg}
}

// Connect opens and verifies the MySQL connection pool
func (db *DB) Connect(ctx context.Context) error {
	sqlDB, err := buildPool(db.cfg)
	if err != nil {
		return err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return mapError(err)
	}
	db.sqlDB = sqlDB
	return nil
}

// Close shuts down the connection pool
func (db *DB) Close(_ context.Context) error {
	if db.sqlDB != nil {
		return db.sqlDB.Close()
	}
	return nil
}

// Ping verifies the connection is alive
func (db *DB) Ping(ctx context.Context) error {
	return mapError(db.sqlDB.PingContext(ctx))
}

// Query executes a query returning multiple rows
func (db *DB) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	rows, err := db.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return &mysqlRows{rows: rows}, nil
}

// QueryRow executes a query returning a single row
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return &mysqlRow{row: db.sqlDB.QueryRowContext(ctx, query, args...)}
}

// Exec executes a statement returning rows affected
func (db *DB) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := db.sqlDB.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, mapError(err)
	}
	n, err := res.RowsAffected()
	return n, mapError(err)
}

// Begin starts a transaction
func (db *DB) Begin(ctx context.Context) (database.Tx, error) {
	tx, err := db.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, mapError(err)
	}
	return &mysqlTx{tx: tx}, nil
}

// SqlDB returns the underlying *sql.DB (for advanced use)
func (db *DB) SqlDB() *sql.DB {
	return db.sqlDB
}

// --- mysqlRows wraps *sql.Rows ---

type mysqlRows struct{ rows *sql.Rows }

func (r *mysqlRows) Next() bool             { return r.rows.Next() }
func (r *mysqlRows) Scan(dest ...any) error { return mapError(r.rows.Scan(dest...)) }
func (r *mysqlRows) Close()                 { r.rows.Close() }
func (r *mysqlRows) Err() error             { return mapError(r.rows.Err()) }

// --- mysqlRow wraps *sql.Row ---

type mysqlRow struct{ row *sql.Row }

func (r *mysqlRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return &database.DBError{
			Kind:    database.ErrKindNotFound,
			Message: "record not found",
			Cause:   err,
		}
	}
	return mapError(err)
}

// --- mysqlTx wraps *sql.Tx ---

type mysqlTx struct{ tx *sql.Tx }

func (t *mysqlTx) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return &mysqlRows{rows: rows}, nil
}

func (t *mysqlTx) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return &mysqlRow{row: t.tx.QueryRowContext(ctx, query, args...)}
}

func (t *mysqlTx) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, mapError(err)
	}
	n, err := res.RowsAffected()
	return n, mapError(err)
}

func (t *mysqlTx) Commit(_ context.Context) error   { return mapError(t.tx.Commit()) }
func (t *mysqlTx) Rollback(_ context.Context) error { return mapError(t.tx.Rollback()) }
