package database

import "context"

// Row represents a single result row
type Row interface {
	Scan(dest ...any) error
}

// Rows represents multiple result rows
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}

// Reader is the read-only connection interface all DB drivers must implement.
// DatRi v1 supports only SELECT queries.
type Reader interface {
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
}
