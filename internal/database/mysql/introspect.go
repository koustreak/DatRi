package mysql

import (
	"context"
	"fmt"

	"github.com/koustreak/DatRi/internal/database"
)

// Introspector implements database.Introspector for MySQL
type Introspector struct {
	db database.Reader
}

// NewIntrospector creates a new MySQL schema introspector
func NewIntrospector(db database.Reader) *Introspector {
	return &Introspector{db: db}
}

// ListTables returns all user-defined table names in the given database
func (m *Introspector) ListTables(ctx context.Context, schema string) ([]string, error) {
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
func (m *Introspector) TableExists(ctx context.Context, schema, table string) (bool, error) {
	const q = `
		SELECT COUNT(*) > 0
		FROM information_schema.tables
		WHERE table_schema = ? AND table_name = ?`

	var exists bool
	if err := m.db.QueryRow(ctx, q, schema, table).Scan(&exists); err != nil {
		return false, fmt.Errorf("table exists: %w", err)
	}
	return exists, nil
}

// InspectTable returns column details for a single table
func (m *Introspector) InspectTable(ctx context.Context, schema, table string) (*database.TableInfo, error) {
	const q = `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES'  AS is_nullable,
			c.column_default,
			c.character_maximum_length,
			(c.column_key = 'PRI') AS is_primary_key,
			(c.column_key = 'UNI') AS is_unique
		FROM information_schema.columns c
		WHERE c.table_schema = ? AND c.table_name = ?
		ORDER BY c.ordinal_position`

	rows, err := m.db.Query(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("inspect table %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	info := &database.TableInfo{Schema: schema, Name: table}
	for rows.Next() {
		var col database.ColumnInfo
		var defaultVal *string
		var maxLen *int

		if err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable,
			&defaultVal, &maxLen, &col.IsPrimaryKey, &col.IsUnique); err != nil {
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

// ListForeignKeys returns all FK relationships in the database
func (m *Introspector) ListForeignKeys(ctx context.Context, schema string) ([]database.ForeignKey, error) {
	const q = `
		SELECT
			rc.constraint_name,
			kcu.table_name,
			kcu.column_name,
			kcu.referenced_table_name,
			kcu.referenced_column_name
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

	var fks []database.ForeignKey
	for rows.Next() {
		var fk database.ForeignKey
		if err := rows.Scan(&fk.Name, &fk.FromTable, &fk.FromColumn, &fk.ToTable, &fk.ToColumn); err != nil {
			return nil, fmt.Errorf("scan fk: %w", err)
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}
