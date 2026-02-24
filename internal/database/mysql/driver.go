package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/koustreak/DatRi/internal/database"

	_ "github.com/go-sql-driver/mysql" // register "mysql" driver
)

// Driver is a MySQL implementation of database.DB backed by database/sql.
// It is safe for concurrent use by multiple goroutines.
type Driver struct {
	db *sql.DB
}

// New opens a MySQL connection pool using the provided Config and returns a Driver.
// It calls Ping to validate the connection before returning.
func New(ctx context.Context, cfg *database.Config) (*Driver, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, &database.DBError{
			Kind:    database.ErrKindConnectionFailed,
			Message: "invalid DSN",
			Cause:   err,
		}
	}

	// Pool tuning — mirrors pgxpool settings as closely as sql.DB allows.
	db.SetMaxOpenConns(int(cfg.MaxConns))
	db.SetMaxIdleConns(int(cfg.MinConns))
	db.SetConnMaxLifetime(cfg.MaxConnLifetime)
	db.SetConnMaxIdleTime(cfg.MaxConnIdleTime)

	d := &Driver{db: db}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := d.Ping(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return d, nil
}

// --- database.DB implementation ---

// Ping verifies the database is reachable.
func (d *Driver) Ping(ctx context.Context) error {
	if err := d.db.PingContext(ctx); err != nil {
		return mapError(err, "ping failed")
	}
	return nil
}

// Close releases all resources held by the connection pool.
func (d *Driver) Close() {
	_ = d.db.Close()
}

// Query executes a SQL statement that returns multiple rows.
func (d *Driver) Query(ctx context.Context, sql string, args ...any) (database.Rows, error) {
	rows, err := d.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, mapError(err, "query failed")
	}
	return &mysqlRows{rows: rows}, nil
}

// QueryRow executes a SQL statement expected to return at most one row.
func (d *Driver) QueryRow(ctx context.Context, query string, args ...any) (database.Row, error) {
	row := d.db.QueryRowContext(ctx, query, args...)
	return &mysqlRow{row: row}, nil
}

// ListTables returns all user-defined table names in the current schema.
func (d *Driver) ListTables(ctx context.Context) ([]string, error) {
	const q = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		  AND table_type   = 'BASE TABLE'
		ORDER BY table_name`

	rows, err := d.db.QueryContext(ctx, q)
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

// TableExists reports whether a table with the given name exists in the current schema.
func (d *Driver) TableExists(ctx context.Context, table string) (bool, error) {
	const q = `
		SELECT 1
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		  AND table_type   = 'BASE TABLE'
		  AND table_name   = ?`

	var exists int
	err := d.db.QueryRowContext(ctx, q, table).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, mapError(err, "failed to check table existence")
	}
	return true, nil
}

// InspectSchema introspects the full schema and returns a *database.Schema.
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
// MySQL conveniently encodes PK and unique info directly in information_schema.columns
// via the column_key field ('PRI', 'UNI', 'MUL').
func (d *Driver) inspectTable(ctx context.Context, table string) (*database.TableInfo, error) {
	columns, pks, err := d.fetchColumns(ctx, table)
	if err != nil {
		return nil, err
	}

	fks, err := d.fetchForeignKeys(ctx, table)
	if err != nil {
		return nil, err
	}

	return &database.TableInfo{
		Name:        table,
		Columns:     columns,
		PrimaryKey:  pks,
		ForeignKeys: fks,
	}, nil
}

// fetchColumns retrieves column metadata for a table.
// MySQL's information_schema.columns has a column_key field that encodes
// 'PRI' (primary key) and 'UNI' (unique constraint) directly — no extra join needed.
func (d *Driver) fetchColumns(ctx context.Context, table string) ([]*database.ColumnInfo, []string, error) {
	const q = `
		SELECT column_name,
		       data_type,
		       is_nullable = 'YES',
		       column_default,
		       column_key
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name   = ?
		ORDER BY ordinal_position`

	rows, err := d.db.QueryContext(ctx, q, table)
	if err != nil {
		return nil, nil, mapError(err, "failed to fetch columns")
	}
	defer rows.Close()

	var cols []*database.ColumnInfo
	var pks []string

	for rows.Next() {
		var c database.ColumnInfo
		var columnKey string
		if err := rows.Scan(&c.Name, &c.DataType, &c.Nullable, &c.Default, &columnKey); err != nil {
			return nil, nil, mapError(err, "failed to scan column info")
		}
		c.IsPrimary = columnKey == "PRI"
		c.IsUnique = columnKey == "UNI"
		if c.IsPrimary {
			pks = append(pks, c.Name)
		}
		cols = append(cols, &c)
	}

	return cols, pks, rows.Err()
}

// fetchForeignKeys retrieves foreign key relationships for a table.
// MySQL stores these in information_schema.key_column_usage with
// non-null referenced_table_name for FK entries.
func (d *Driver) fetchForeignKeys(ctx context.Context, table string) ([]*database.ForeignKey, error) {
	const q = `
		SELECT column_name,
		       referenced_table_name,
		       referenced_column_name
		FROM information_schema.key_column_usage
		WHERE table_schema              = DATABASE()
		  AND table_name                = ?
		  AND referenced_table_name    IS NOT NULL`

	rows, err := d.db.QueryContext(ctx, q, table)
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

// --- sql.DB type wrappers ---

// mysqlRows wraps *sql.Rows to satisfy database.Rows.
// The only mismatch is Close() — sql.Rows.Close() returns error,
// our interface discards it (errors are captured by Err() after iteration).
type mysqlRows struct {
	rows *sql.Rows
}

func (r *mysqlRows) Next() bool                 { return r.rows.Next() }
func (r *mysqlRows) Scan(dest ...any) error     { return r.rows.Scan(dest...) }
func (r *mysqlRows) Columns() ([]string, error) { return r.rows.Columns() }
func (r *mysqlRows) Close()                     { _ = r.rows.Close() }
func (r *mysqlRows) Err() error                 { return r.rows.Err() }

// mysqlRow wraps *sql.Row to satisfy database.Row.
type mysqlRow struct {
	row *sql.Row
}

func (r *mysqlRow) Scan(dest ...any) error { return r.row.Scan(dest...) }

// --- error mapping ---

// mapError translates go-sql-driver/mysql errors into *database.DBError.
func mapError(err error, msg string) *database.DBError {
	if err == nil {
		return nil
	}

	// Context deadline / cancellation
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &database.DBError{Kind: database.ErrKindTimeout, Message: msg, Cause: err}
	}

	// No rows
	if errors.Is(err, sql.ErrNoRows) {
		return &database.DBError{Kind: database.ErrKindNotFound, Message: msg, Cause: err}
	}

	// MySQL server-side error
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		kind := classifyMySQLCode(mysqlErr.Number)
		return &database.DBError{
			Kind:    kind,
			Message: fmt.Sprintf("%s: %s", msg, mysqlErr.Message),
			Cause:   err,
		}
	}

	// Fallthrough: assume connection-level failure
	return &database.DBError{Kind: database.ErrKindConnectionFailed, Message: msg, Cause: err}
}

// classifyMySQLCode maps MySQL error numbers to ErrKind.
// Reference: https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
func classifyMySQLCode(code uint16) database.ErrKind {
	switch code {
	case 1044, 1045, 1046, 1049: // access denied, unknown db
		return database.ErrKindConnectionFailed
	case 1040, 1203: // too many connections
		return database.ErrKindConnectionFailed
	case 1054, 1064, 1146: // unknown column, syntax error, table doesn't exist
		return database.ErrKindQueryFailed
	default:
		return database.ErrKindQueryFailed
	}
}
