package database

import "context"

// DB is the central contract for all database operations.
// All layers above this package talk only to this interface —
// they never import the postgres or mysql packages directly.
type DB interface {
	// Ping verifies the database is reachable.
	Ping(ctx context.Context) error

	// Close releases all resources held by the connection pool.
	Close()

	// Query executes a SQL statement that returns multiple rows.
	Query(ctx context.Context, sql string, args ...any) (Rows, error)

	// QueryRow executes a SQL statement that returns at most one row.
	QueryRow(ctx context.Context, sql string, args ...any) (Row, error)

	// ListTables returns all user-defined table names in the public schema.
	ListTables(ctx context.Context) ([]string, error)

	// TableExists reports whether a table with the given name exists.
	TableExists(ctx context.Context, table string) (bool, error)

	// InspectSchema returns the full schema of the database.
	// This is an expensive operation — callers should cache the result.
	InspectSchema(ctx context.Context) (*Schema, error)
}

// Rows is an abstraction over a database result set.
// Callers must always call Close() when done, even on error.
type Rows interface {
	// Next advances to the next row.
	// Returns false when no more rows exist or on error.
	Next() bool

	// Scan copies the current row's columns into the provided destinations.
	Scan(dest ...any) error

	// Columns returns the column names of the result set.
	Columns() ([]string, error)

	// Close releases resources held by the result set.
	Close()

	// Err returns any error encountered during iteration.
	Err() error
}

// Row is an abstraction over a single database row.
type Row interface {
	Scan(dest ...any) error
}
