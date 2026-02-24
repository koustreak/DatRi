package database

import "context"

// Introspector reads the structure of a database (tables, columns, foreign keys).
// Each driver implements the DB-specific queries; InspectSchema is shared.
type Introspector interface {
	ListTables(ctx context.Context, schema string) ([]string, error)
	TableExists(ctx context.Context, schema, table string) (bool, error)
	InspectTable(ctx context.Context, schema, table string) (*TableInfo, error)
	ListForeignKeys(ctx context.Context, schema string) ([]ForeignKey, error)
}

// InspectSchema builds the full SchemaInfo by orchestrating the Introspector.
// Shared across all DB drivers — no duplication in drivers.
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
