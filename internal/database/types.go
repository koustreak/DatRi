package database

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
