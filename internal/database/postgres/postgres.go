package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koustreak/DatRi/internal/database"
)

// DB implements database.DB for PostgreSQL using pgxpool
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
	return mapError(db.pool.Ping(ctx))
}

// Query executes a query returning multiple rows
func (db *DB) Query(ctx context.Context, sql string, args ...any) (database.Rows, error) {
	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return &pgRows{rows: rows}, nil
}

// QueryRow executes a query returning a single row
func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) database.Row {
	return &pgRow{row: db.pool.QueryRow(ctx, sql, args...)}
}

// Exec executes a statement returning the number of rows affected
func (db *DB) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := db.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, mapError(err)
	}
	return tag.RowsAffected(), nil
}

// Begin starts a transaction
func (db *DB) Begin(ctx context.Context) (database.Tx, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &pgTx{tx: tx}, nil
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

// --- pgTx wraps pgx.Tx ---

type pgTx struct{ tx pgx.Tx }

func (t *pgTx) Query(ctx context.Context, sql string, args ...any) (database.Rows, error) {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapError(err)
	}
	return &pgRows{rows: rows}, nil
}

func (t *pgTx) QueryRow(ctx context.Context, sql string, args ...any) database.Row {
	return &pgRow{row: t.tx.QueryRow(ctx, sql, args...)}
}

func (t *pgTx) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := t.tx.Exec(ctx, sql, args...)
	if err != nil {
		return 0, mapError(err)
	}
	return tag.RowsAffected(), nil
}

func (t *pgTx) Commit(ctx context.Context) error {
	return mapError(t.tx.Commit(ctx))
}

func (t *pgTx) Rollback(ctx context.Context) error {
	return mapError(t.tx.Rollback(ctx))
}
