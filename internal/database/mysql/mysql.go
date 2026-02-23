package mysql

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/go-sql-driver/mysql" // register driver
	"github.com/koustreak/DatRi/internal/database"
)

// DB implements database.Reader for MySQL using database/sql
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
	if db.sqlDB == nil {
		return &database.DBError{Kind: database.ErrKindConnection, Message: "not connected"}
	}
	return mapError(db.sqlDB.PingContext(ctx))
}

// Query executes a SELECT query returning multiple rows
func (db *DB) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	rows, err := db.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return &mysqlRows{rows: rows}, nil
}

// QueryRow executes a SELECT query returning a single row
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return &mysqlRow{row: db.sqlDB.QueryRowContext(ctx, query, args...)}
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
