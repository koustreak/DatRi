package schema

import (
	"context"
	"fmt"

	"github.com/koustreak/DatRi/internal/database"
)

// MySQLIntrospector implements Reader for MySQL using information_schema
type MySQLIntrospector struct {
	db database.DB
}

// NewMySQLIntrospector creates a new MySQL schema introspector
func NewMySQLIntrospector(db database.DB) *MySQLIntrospector {
	return &MySQLIntrospector{db: db}
}

// ListTables returns all user-defined table names in the given database (schema = database in MySQL)
func (m *MySQLIntrospector) ListTables(ctx context.Context, schema string) ([]string, error) {
	const q = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name`

	rows, err := m.db.Query(ctx, q, schema)
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
func (m *MySQLIntrospector) TableExists(ctx context.Context, schema, table string) (bool, error) {
	const q = `
		SELECT COUNT(*) > 0
		FROM information_schema.tables
		WHERE table_schema = ? AND table_name = ?`

	var exists bool
	if err := m.db.QueryRow(ctx, q, schema, table).Scan(&exists); err != nil {
		return false, fmt.Errorf("table exists check: %w", err)
	}
	return exists, nil
}

// InspectTable returns column details for a single table
func (m *MySQLIntrospector) InspectTable(ctx context.Context, schema, table string) (*TableInfo, error) {
	const q = `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES'                         AS is_nullable,
			c.column_default,
			c.character_maximum_length,
			(c.column_key = 'PRI')                        AS is_primary_key,
			(c.column_key = 'UNI')                        AS is_unique
		FROM information_schema.columns c
		WHERE c.table_schema = ?
		  AND c.table_name   = ?
		ORDER BY c.ordinal_position`

	rows, err := m.db.Query(ctx, q, schema, table)
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

// InspectSchema returns all tables and foreign keys for the given database
func (m *MySQLIntrospector) InspectSchema(ctx context.Context, schema string) (*SchemaInfo, error) {
	tables, err := m.ListTables(ctx, schema)
	if err != nil {
		return nil, err
	}

	info := &SchemaInfo{}
	for _, table := range tables {
		ti, err := m.InspectTable(ctx, schema, table)
		if err != nil {
			return nil, err
		}
		info.Tables = append(info.Tables, *ti)
	}

	fks, err := m.listForeignKeys(ctx, schema)
	if err != nil {
		return nil, err
	}
	info.ForeignKeys = fks

	return info, nil
}

// listForeignKeys returns all FK relationships in the database
func (m *MySQLIntrospector) listForeignKeys(ctx context.Context, schema string) ([]ForeignKey, error) {
	const q = `
		SELECT
			rc.constraint_name,
			kcu.table_name       AS from_table,
			kcu.column_name      AS from_column,
			kcu.referenced_table_name  AS to_table,
			kcu.referenced_column_name AS to_column
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
			ON rc.constraint_name = kcu.constraint_name
			AND rc.constraint_schema = kcu.table_schema
		WHERE rc.constraint_schema = ?
		ORDER BY rc.constraint_name`

	rows, err := m.db.Query(ctx, q, schema)
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
