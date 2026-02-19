package schema

// ColumnInfo describes a single column in a table
type ColumnInfo struct {
	Name         string
	DataType     string // postgres type: text, int4, timestamptz, etc.
	IsNullable   bool
	IsPrimaryKey bool
	IsUnique     bool
	DefaultValue *string // nil if no default
	MaxLength    *int    // nil for non-char types
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
