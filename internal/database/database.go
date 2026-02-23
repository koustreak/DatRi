package database

import (
	"context"
)

// ColumnInfo describes a single column in a table
type ColumnInfo struct {
	Name         string
	DataType     string
	IsNullable   bool
	IsPrimaryKey bool
	IsUnique     bool
	DefaultValue *string
	MaxLength    *int
}

// TableInfo describes a table and its columns
type TableInfo struct {
	Schema  string
	Name    string
	Columns []ColumnInfo
}

// ForeignKey describes a relationship between two tables
type ForeignKey struct {
	Name       string
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
}

// SchemaInfo is the full introspected database schema
type SchemaInfo struct {
	Tables      []TableInfo
	ForeignKeys []ForeignKey
}

// Introspector reads the structure of a database (tables, columns, keys).
// Each driver implements the DB-specific queries; InspectSchema is shared.
type Introspector interface {
	ListTables(ctx context.Context, schema string) ([]string, error)
	TableExists(ctx context.Context, schema, table string) (bool, error)
	InspectTable(ctx context.Context, schema, table string) (*TableInfo, error)
	ListForeignKeys(ctx context.Context, schema string) ([]ForeignKey, error)
}

// InspectSchema builds the full SchemaInfo by orchestrating the Introspector.
// This is shared across all DB drivers — no duplication.
func InspectSchema(ctx context.Context, i Introspector, schema string) (*SchemaInfo, error) {
	tables, err := i.ListTables(ctx, schema)
	if err != nil {
		return nil, err
	}

	info := &SchemaInfo{}
	for _, table := range tables {
		ti, err := i.InspectTable(ctx, schema, table)
		if err != nil {
			return nil, err
		}
		info.Tables = append(info.Tables, *ti)
	}

	fks, err := i.ListForeignKeys(ctx, schema)
	if err != nil {
		return nil, err
	}
	info.ForeignKeys = fks
	return info, nil
}

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

// Reader is the read-only interface for all database drivers.
// DatRi v1 is strictly read-only — only SELECT queries are supported.
type Reader interface {
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
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
