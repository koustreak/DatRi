package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koustreak/DatRi/internal/database"
)

// Driver is a PostgreSQL implementation of database.DB backed by pgxpool.
// It is safe for concurrent use by multiple goroutines.
type Driver struct {
	pool *pgxpool.Pool
}

// New connects to PostgreSQL using the provided Config and returns a Driver.
// It calls Ping to validate the connection before returning.
func New(ctx context.Context, cfg *database.Config) (*Driver, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, &database.DBError{
			Kind:    database.ErrKindConnectionFailed,
			Message: "invalid DSN",
			Cause:   err,
		}
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, &database.DBError{
			Kind:    database.ErrKindConnectionFailed,
			Message: "failed to create connection pool",
			Cause:   err,
		}
	}

	d := &Driver{pool: pool}

	if err := d.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return d, nil
}

// --- database.DB implementation ---

// Ping verifies the database is reachable by acquiring and releasing a connection.
func (d *Driver) Ping(ctx context.Context) error {
	if err := d.pool.Ping(ctx); err != nil {
		return mapError(err, "ping failed")
	}
	return nil
}

// Close drains the connection pool. Call when the application shuts down.
func (d *Driver) Close() {
	d.pool.Close()
}

// Query executes a SQL statement that returns multiple rows.
func (d *Driver) Query(ctx context.Context, sql string, args ...any) (database.Rows, error) {
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapError(err, "query failed")
	}
	return &pgxRows{rows: rows}, nil
}

// QueryRow executes a SQL statement expected to return at most one row.
func (d *Driver) QueryRow(ctx context.Context, sql string, args ...any) (database.Row, error) {
	row := d.pool.QueryRow(ctx, sql, args...)
	return &pgxRow{row: row}, nil
}

// ListTables returns all user-defined table names in the public schema.
func (d *Driver) ListTables(ctx context.Context) ([]string, error) {
	const q = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type   = 'BASE TABLE'
		ORDER BY table_name`

	rows, err := d.pool.Query(ctx, q)
	if err != nil {
		return nil, mapError(err, "failed to list tables")
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, mapError(err, "failed to scan table name")
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err, "error iterating tables")
	}
	return tables, nil
}

// TableExists reports whether a table with the given name exists in the public schema.
func (d *Driver) TableExists(ctx context.Context, table string) (bool, error) {
	const q = `
		SELECT 1
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type   = 'BASE TABLE'
		  AND table_name   = $1`

	var exists int
	err := d.pool.QueryRow(ctx, q, table).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, mapError(err, "failed to check table existence")
	}
	return true, nil
}

// InspectSchema introspects the full public schema and returns a *database.Schema.
// This is intentionally expensive — callers must cache the result.
func (d *Driver) InspectSchema(ctx context.Context) (*database.Schema, error) {
	tables, err := d.ListTables(ctx)
	if err != nil {
		return nil, err
	}

	schema := &database.Schema{
		Tables: make(map[string]*database.TableInfo, len(tables)),
	}

	for _, tableName := range tables {
		info, err := d.inspectTable(ctx, tableName)
		if err != nil {
			return nil, fmt.Errorf("inspecting table %q: %w", tableName, err)
		}
		schema.Tables[tableName] = info
	}

	return schema, nil
}

// inspectTable fetches column, primary key, unique, and foreign key info for one table.
func (d *Driver) inspectTable(ctx context.Context, table string) (*database.TableInfo, error) {
	columns, err := d.fetchColumns(ctx, table)
	if err != nil {
		return nil, err
	}

	pks, err := d.fetchPrimaryKeys(ctx, table)
	if err != nil {
		return nil, err
	}

	uniqueCols, err := d.fetchUniqueColumns(ctx, table)
	if err != nil {
		return nil, err
	}

	fks, err := d.fetchForeignKeys(ctx, table)
	if err != nil {
		return nil, err
	}

	// Mark columns that are primary or unique
	pkSet := toSet(pks)
	uqSet := toSet(uniqueCols)
	for _, col := range columns {
		col.IsPrimary = pkSet[col.Name]
		col.IsUnique = uqSet[col.Name]
	}

	return &database.TableInfo{
		Name:        table,
		Columns:     columns,
		PrimaryKey:  pks,
		ForeignKeys: fks,
	}, nil
}

func (d *Driver) fetchColumns(ctx context.Context, table string) ([]*database.ColumnInfo, error) {
	const q = `
		SELECT column_name,
		       data_type,
		       is_nullable = 'YES',
		       column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name   = $1
		ORDER BY ordinal_position`

	rows, err := d.pool.Query(ctx, q, table)
	if err != nil {
		return nil, mapError(err, "failed to fetch columns")
	}
	defer rows.Close()

	var cols []*database.ColumnInfo
	for rows.Next() {
		var c database.ColumnInfo
		if err := rows.Scan(&c.Name, &c.DataType, &c.Nullable, &c.Default); err != nil {
			return nil, mapError(err, "failed to scan column info")
		}
		cols = append(cols, &c)
	}
	return cols, rows.Err()
}

func (d *Driver) fetchPrimaryKeys(ctx context.Context, table string) ([]string, error) {
	const q = `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema    = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema    = 'public'
		  AND tc.table_name      = $1
		ORDER BY kcu.ordinal_position`

	return d.fetchStringList(ctx, q, table, "failed to fetch primary keys")
}

func (d *Driver) fetchUniqueColumns(ctx context.Context, table string) ([]string, error) {
	const q = `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema    = kcu.table_schema
		WHERE tc.constraint_type = 'UNIQUE'
		  AND tc.table_schema    = 'public'
		  AND tc.table_name      = $1`

	return d.fetchStringList(ctx, q, table, "failed to fetch unique columns")
}

func (d *Driver) fetchForeignKeys(ctx context.Context, table string) ([]*database.ForeignKey, error) {
	const q = `
		SELECT kcu.column_name,
		       ccu.table_name  AS ref_table,
		       ccu.column_name AS ref_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema    = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON tc.constraint_name = ccu.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema    = 'public'
		  AND tc.table_name      = $1`

	rows, err := d.pool.Query(ctx, q, table)
	if err != nil {
		return nil, mapError(err, "failed to fetch foreign keys")
	}
	defer rows.Close()

	var fks []*database.ForeignKey
	for rows.Next() {
		fk := &database.ForeignKey{}
		if err := rows.Scan(&fk.Column, &fk.RefTable, &fk.RefColumn); err != nil {
			return nil, mapError(err, "failed to scan foreign key")
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

// fetchStringList is a helper for queries that return a single text column.
func (d *Driver) fetchStringList(ctx context.Context, q, table, errMsg string) ([]string, error) {
	rows, err := d.pool.Query(ctx, q, table)
	if err != nil {
		return nil, mapError(err, errMsg)
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, mapError(err, errMsg)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- pgx type wrappers ---

// pgxRows wraps pgx.Rows to satisfy database.Rows.
type pgxRows struct {
	rows pgx.Rows
}

func (r *pgxRows) Next() bool             { return r.rows.Next() }
func (r *pgxRows) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r *pgxRows) Close()                 { r.rows.Close() }
func (r *pgxRows) Err() error             { return r.rows.Err() }

func (r *pgxRows) Columns() ([]string, error) {
	descs := r.rows.FieldDescriptions()
	cols := make([]string, len(descs))
	for i, d := range descs {
		cols[i] = d.Name
	}
	return cols, nil
}

// pgxRow wraps pgx.Row to satisfy database.Row.
type pgxRow struct {
	row pgx.Row
}

func (r *pgxRow) Scan(dest ...any) error { return r.row.Scan(dest...) }

// --- error mapping ---

// mapError translates pgx / pgconn native errors into *database.DBError.
func mapError(err error, msg string) *database.DBError {
	if err == nil {
		return nil
	}

	// Context cancellation / deadline exceeded
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &database.DBError{Kind: database.ErrKindTimeout, Message: msg, Cause: err}
	}

	// No rows
	if errors.Is(err, pgx.ErrNoRows) {
		return &database.DBError{Kind: database.ErrKindNotFound, Message: msg, Cause: err}
	}

	// Postgres server-side error (SQLSTATE codes)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		kind := database.ErrKindQueryFailed
		// Class 08 — connection errors
		if len(pgErr.Code) >= 2 && pgErr.Code[:2] == "08" {
			kind = database.ErrKindConnectionFailed
		}
		return &database.DBError{
			Kind:    kind,
			Message: fmt.Sprintf("%s: %s", msg, pgErr.Message),
			Cause:   err,
		}
	}

	// Fallthrough: connection-level errors (TLS, network, auth)
	return &database.DBError{Kind: database.ErrKindConnectionFailed, Message: msg, Cause: err}
}

// --- helpers ---

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
