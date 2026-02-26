package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/errs"

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
		return nil, errs.Wrap(errs.ErrKindConnectionFailed, "invalid DSN", err)
	}

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

func (d *Driver) Ping(ctx context.Context) error {
	if err := d.db.PingContext(ctx); err != nil {
		return mapError(err, "ping failed")
	}
	return nil
}

func (d *Driver) Close() {
	_ = d.db.Close()
}

func (d *Driver) Query(ctx context.Context, sql string, args ...any) (database.Rows, error) {
	rows, err := d.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, mapError(err, "query failed")
	}
	return &mysqlRows{rows: rows}, nil
}

func (d *Driver) QueryRow(ctx context.Context, query string, args ...any) (database.Row, error) {
	row := d.db.QueryRowContext(ctx, query, args...)
	return &mysqlRow{row: row}, nil
}

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

type mysqlRows struct {
	rows *sql.Rows
}

func (r *mysqlRows) Next() bool                 { return r.rows.Next() }
func (r *mysqlRows) Scan(dest ...any) error     { return r.rows.Scan(dest...) }
func (r *mysqlRows) Columns() ([]string, error) { return r.rows.Columns() }
func (r *mysqlRows) Close()                     { _ = r.rows.Close() }
func (r *mysqlRows) Err() error                 { return r.rows.Err() }

type mysqlRow struct {
	row *sql.Row
}

func (r *mysqlRow) Scan(dest ...any) error { return r.row.Scan(dest...) }

// --- error mapping ---

// mapError translates go-sql-driver/mysql errors into *errs.Error.
func mapError(err error, msg string) *errs.Error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errs.Wrap(errs.ErrKindTimeout, msg, err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		return errs.Wrap(errs.ErrKindNotFound, msg, err)
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return errs.Wrap(
			classifyMySQLCode(mysqlErr.Number),
			fmt.Sprintf("%s: %s", msg, mysqlErr.Message),
			err,
		)
	}

	return errs.Wrap(errs.ErrKindConnectionFailed, msg, err)
}

// classifyMySQLCode maps MySQL error numbers to ErrKind.
func classifyMySQLCode(code uint16) errs.ErrKind {
	switch code {
	case 1044, 1045, 1046, 1049:
		return errs.ErrKindConnectionFailed
	case 1040, 1203:
		return errs.ErrKindConnectionFailed
	case 1054, 1064, 1146:
		return errs.ErrKindQueryFailed
	default:
		return errs.ErrKindQueryFailed
	}
}
