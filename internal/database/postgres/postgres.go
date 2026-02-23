package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koustreak/DatRi/internal/database"
)

// DB implements database.Reader for PostgreSQL using pgxpool
type DB struct {
	pool *pgxpool.Pool
	cfg  *database.Config
}

// New creates a new Postgres DB instance (does not connect yet)
func New(cfg *database.Config) *DB {
	return &DB{cfg: cfg}
}

// Connect establishes the connection pool
func (db *DB) Connect(ctx context.Context) error {
	pool, err := buildPool(ctx, db.cfg)
	if err != nil {
		return err
	}
	db.pool = pool
	return nil
}

// Close shuts down the connection pool
func (db *DB) Close(_ context.Context) error {
	if db.pool != nil {
		db.pool.Close()
	}
	return nil
}

// Ping verifies the connection is alive
func (db *DB) Ping(ctx context.Context) error {
	if db.pool == nil {
		return &database.DBError{Kind: database.ErrKindConnection, Message: "not connected"}
	}
	return mapError(db.pool.Ping(ctx))
}

// Query executes a SELECT query returning multiple rows
func (db *DB) Query(ctx context.Context, sql string, args ...any) (database.Rows, error) {
	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return &pgRows{rows: rows}, nil
}

// QueryRow executes a SELECT query returning a single row
func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) database.Row {
	return &pgRow{row: db.pool.QueryRow(ctx, sql, args...)}
}

// Pool returns the underlying pgxpool (for advanced use)
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// --- pgRows wraps pgx.Rows ---

type pgRows struct{ rows pgx.Rows }

func (r *pgRows) Next() bool             { return r.rows.Next() }
func (r *pgRows) Scan(dest ...any) error { return mapError(r.rows.Scan(dest...)) }
func (r *pgRows) Close()                 { r.rows.Close() }
func (r *pgRows) Err() error             { return mapError(r.rows.Err()) }

// --- pgRow wraps pgx.Row ---

type pgRow struct{ row pgx.Row }

func (r *pgRow) Scan(dest ...any) error { return mapError(r.row.Scan(dest...)) }
