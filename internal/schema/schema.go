package schema

import "context"

// Reader is the interface for introspecting a database schema
type Reader interface {
	// ListTables returns all user tables in the given schema (e.g. "public")
	ListTables(ctx context.Context, schema string) ([]string, error)

	// TableExists checks whether a table exists
	TableExists(ctx context.Context, schema, table string) (bool, error)

	// InspectTable returns full column info for a table
	InspectTable(ctx context.Context, schema, table string) (*TableInfo, error)

	// InspectSchema returns the full schema (all tables + foreign keys)
	InspectSchema(ctx context.Context, schema string) (*SchemaInfo, error)
}
