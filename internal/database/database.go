package database

import (
	"context"
)

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

// DB is the common interface all database drivers must implement
type DB interface {
	// Lifecycle
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
	Ping(ctx context.Context) error

	// Query execution
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Exec(ctx context.Context, sql string, args ...any) (int64, error)

	// Transactions
	Begin(ctx context.Context) (Tx, error)
}

// Tx represents a database transaction
type Tx interface {
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Config holds common database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	// Pool settings
	MaxConns     int32
	MinConns     int32
	MaxIdleConns int32
}

// ErrKind categorizes database errors
type ErrKind int

const (
	ErrKindNotFound   ErrKind = iota // row not found
	ErrKindConflict                  // unique constraint violation
	ErrKindInvalid                   // not null / missing required field
	ErrKindConnection                // connection failure
	ErrKindQuery                     // bad query / syntax
	ErrKindUnknown                   // uncategorized
)

// DBError is DatRi's unified database error
type DBError struct {
	Kind    ErrKind
	Message string
	Cause   error
}

func (e *DBError) Error() string { return e.Message }
func (e *DBError) Unwrap() error { return e.Cause }
