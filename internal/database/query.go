package database

import "fmt"

// Operator defines the comparison type in a filter
type Operator string

const (
	OpEq   Operator = "eq"   // field = value
	OpNeq  Operator = "neq"  // field != value
	OpGt   Operator = "gt"   // field > value
	OpLt   Operator = "lt"   // field < value
	OpGte  Operator = "gte"  // field >= value
	OpLte  Operator = "lte"  // field <= value
	OpLike Operator = "like" // field LIKE %value%
	OpIn   Operator = "in"   // field IN (values...)
)

// SortDir defines sort direction
type SortDir string

const (
	SortAsc  SortDir = "ASC"
	SortDesc SortDir = "DESC"
)

// Filter represents a single WHERE condition
type Filter struct {
	Field    string
	Operator Operator
	Value    any
}

// SortField represents an ORDER BY clause
type SortField struct {
	Field     string
	Direction SortDir
}

// ListOptions holds all parameters for a GET (list) query
type ListOptions struct {
	Table   string      // required: table to query
	Fields  []string    // columns to SELECT, empty = *
	Filters []Filter    // WHERE conditions (ANDed together)
	Sort    []SortField // ORDER BY
	Limit   int         // 0 = no limit (not recommended in prod)
	Offset  int         // for pagination
}

// Query is the result of building a ListOptions into executable SQL
type Query struct {
	SQL  string
	Args []any
}

// Placeholder is a function that returns the param placeholder for position n (1-indexed)
// Postgres: $1, $2 ...   MySQL: ?, ? ...
type Placeholder func(n int) string

// ExportBuildQuery constructs a parameterized SELECT query from ListOptions.
// Called by each driver with its own placeholder style.
func ExportBuildQuery(opts ListOptions, ph Placeholder) (Query, error) {
	if opts.Table == "" {
		return Query{}, fmt.Errorf("ListOptions.Table is required")
	}

	var args []any
	argN := 0

	nextArg := func(v any) string {
		argN++
		args = append(args, v)
		return ph(argN)
	}

	// SELECT
	fields := "*"
	if len(opts.Fields) > 0 {
		fields = joinIdents(opts.Fields)
	}
	sql := fmt.Sprintf("SELECT %s FROM %s", fields, quoteIdent(opts.Table))

	// WHERE
	if len(opts.Filters) > 0 {
		sql += " WHERE "
		for i, f := range opts.Filters {
			if i > 0 {
				sql += " AND "
			}
			col := quoteIdent(f.Field)
			switch f.Operator {
			case OpEq:
				sql += fmt.Sprintf("%s = %s", col, nextArg(f.Value))
			case OpNeq:
				sql += fmt.Sprintf("%s != %s", col, nextArg(f.Value))
			case OpGt:
				sql += fmt.Sprintf("%s > %s", col, nextArg(f.Value))
			case OpLt:
				sql += fmt.Sprintf("%s < %s", col, nextArg(f.Value))
			case OpGte:
				sql += fmt.Sprintf("%s >= %s", col, nextArg(f.Value))
			case OpLte:
				sql += fmt.Sprintf("%s <= %s", col, nextArg(f.Value))
			case OpLike:
				sql += fmt.Sprintf("%s LIKE %s", col, nextArg(fmt.Sprintf("%%%v%%", f.Value)))
			case OpIn:
				sql += fmt.Sprintf("%s IN (%s)", col, nextArg(f.Value))
			default:
				return Query{}, fmt.Errorf("unsupported operator: %s", f.Operator)
			}
		}
	}

	// ORDER BY
	if len(opts.Sort) > 0 {
		sql += " ORDER BY "
		for i, s := range opts.Sort {
			if i > 0 {
				sql += ", "
			}
			dir := SortAsc
			if s.Direction == SortDesc {
				dir = SortDesc
			}
			sql += fmt.Sprintf("%s %s", quoteIdent(s.Field), dir)
		}
	}

	// LIMIT / OFFSET
	if opts.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	return Query{SQL: sql, Args: args}, nil
}

// quoteIdent wraps an identifier in double quotes to prevent SQL injection
func quoteIdent(s string) string {
	return `"` + s + `"`
}

// joinIdents quotes and joins a list of column names
func joinIdents(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ", "
		}
		result += quoteIdent(f)
	}
	return result
}
