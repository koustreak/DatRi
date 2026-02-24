package database

// Schema represents the entire introspected database schema.
// It is built once at startup and cached — never fetched per-request.
type Schema struct {
	// Tables maps table name to its metadata.
	Tables map[string]*TableInfo
}

// TableInfo describes a single table.
type TableInfo struct {
	// Name is the table name as it appears in the database.
	Name string

	// Columns is the ordered list of columns (by ordinal position).
	Columns []*ColumnInfo

	// PrimaryKey holds the column names that form the primary key.
	// Composite PKs are fully supported.
	PrimaryKey []string

	// ForeignKeys lists all outbound foreign key relationships.
	ForeignKeys []*ForeignKey
}

// ColumnInfo describes a single column within a table.
type ColumnInfo struct {
	// Name is the column name.
	Name string

	// DataType is the database-level type (e.g. "integer", "text", "timestamp").
	DataType string

	// Nullable reports whether the column accepts NULL values.
	Nullable bool

	// IsPrimary reports whether this column is part of the primary key.
	IsPrimary bool

	// IsUnique reports whether this column has a UNIQUE constraint.
	IsUnique bool

	// Default is the column's default expression, if any (e.g. "now()", "0").
	Default *string
}

// ForeignKey describes a single foreign key relationship on a column.
type ForeignKey struct {
	// Column is the local column that holds the foreign key.
	Column string

	// RefTable is the referenced table.
	RefTable string

	// RefColumn is the referenced column in the RefTable.
	RefColumn string
}
