package schema

import (
	"context"
	"fmt"

	"github.com/koustreak/DatRi/internal/database"
)

// PgIntrospector implements Reader for PostgreSQL using information_schema
type PgIntrospector struct {
	db database.DB
}

// NewPgIntrospector creates a new Postgres schema introspector
func NewPgIntrospector(db database.DB) *PgIntrospector {
	return &PgIntrospector{db: db}
}

// ListTables returns all user-defined table names in the given schema
func (p *PgIntrospector) ListTables(ctx context.Context, schema string) ([]string, error) {
	const q = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name`

	rows, err := p.db.Query(ctx, q, schema)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// TableExists checks whether a specific table exists
func (p *PgIntrospector) TableExists(ctx context.Context, schema, table string) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)`

	var exists bool
	if err := p.db.QueryRow(ctx, q, schema, table).Scan(&exists); err != nil {
		return false, fmt.Errorf("table exists check: %w", err)
	}
	return exists, nil
}

// InspectTable returns column details for a single table
func (p *PgIntrospector) InspectTable(ctx context.Context, schema, table string) (*TableInfo, error) {
	const q = `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES'              AS is_nullable,
			c.column_default,
			c.character_maximum_length,
			COALESCE(pk.is_pk, false)          AS is_primary_key,
			COALESCE(uq.is_unique, false)      AS is_unique
		FROM information_schema.columns c

		-- Primary key check
		LEFT JOIN (
			SELECT kcu.column_name, true AS is_pk
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
			  AND tc.table_schema = $1
			  AND tc.table_name   = $2
		) pk ON pk.column_name = c.column_name

		-- Unique constraint check
		LEFT JOIN (
			SELECT kcu.column_name, true AS is_unique
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'UNIQUE'
			  AND tc.table_schema = $1
			  AND tc.table_name   = $2
		) uq ON uq.column_name = c.column_name

		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`

	rows, err := p.db.Query(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("inspect table %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	info := &TableInfo{Schema: schema, Name: table}
	for rows.Next() {
		var col ColumnInfo
		var defaultVal *string
		var maxLen *int

		if err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&defaultVal,
			&maxLen,
			&col.IsPrimaryKey,
			&col.IsUnique,
		); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}

		col.DefaultValue = defaultVal
		col.MaxLength = maxLen
		info.Columns = append(info.Columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(info.Columns) == 0 {
		return nil, fmt.Errorf("table %s.%s not found or has no columns", schema, table)
	}
	return info, nil
}

// InspectSchema returns all tables and foreign keys in the schema
func (p *PgIntrospector) InspectSchema(ctx context.Context, schema string) (*SchemaInfo, error) {
	tables, err := p.ListTables(ctx, schema)
	if err != nil {
		return nil, err
	}

	info := &SchemaInfo{}
	for _, table := range tables {
		ti, err := p.InspectTable(ctx, schema, table)
		if err != nil {
			return nil, err
		}
		info.Tables = append(info.Tables, *ti)
	}

	// Foreign keys
	fks, err := p.listForeignKeys(ctx, schema)
	if err != nil {
		return nil, err
	}
	info.ForeignKeys = fks

	return info, nil
}

// listForeignKeys returns all FK relationships in the schema
func (p *PgIntrospector) listForeignKeys(ctx context.Context, schema string) ([]ForeignKey, error) {
	const q = `
		SELECT
			tc.constraint_name,
			kcu.table_name   AS from_table,
			kcu.column_name  AS from_column,
			ccu.table_name   AS to_table,
			ccu.column_name  AS to_column
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = $1
		ORDER BY tc.constraint_name`

	rows, err := p.db.Query(ctx, q, schema)
	if err != nil {
		return nil, fmt.Errorf("list foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Name, &fk.FromTable, &fk.FromColumn, &fk.ToTable, &fk.ToColumn); err != nil {
			return nil, fmt.Errorf("scan foreign key: %w", err)
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}
