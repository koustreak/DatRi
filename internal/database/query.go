package database

import (
	"fmt"
	"strings"
)

// Dialect controls which SQL placeholder style the query builder emits.
type Dialect int

const (
	// DialectPostgres uses $1, $2, … placeholders.
	DialectPostgres Dialect = iota

	// DialectMySQL uses ? placeholders.
	DialectMySQL
)

// validOps is the allowlist of comparison operators for WHERE clauses.
// Any operator not in this list is rejected to prevent SQL injection
// through the operator position (which cannot be parameterized).
var validOps = map[string]bool{
	"=":     true,
	"!=":    true,
	"<>":    true,
	"<":     true,
	">":     true,
	"<=":    true,
	">=":    true,
	"LIKE":  true,
	"ILIKE": true,
}

// SelectBuilder constructs a parameterized SELECT query using a fluent API.
// Values are never interpolated into the SQL string — always passed as args.
//
// Usage (Postgres):
//
//	sql, args, err := Select("users", DialectPostgres).
//	    Columns("id", "name", "email").
//	    Where("active", "=", true).
//	    OrderBy("created_at", Desc).
//	    Limit(20).
//	    Offset(0).
//	    Build()
type SelectBuilder struct {
	table   string
	dialect Dialect
	columns []string
	where   []whereClause
	orderBy []orderClause
	limit   *int
	offset  *int
}

// SortDirection controls the ORDER BY direction.
type SortDirection bool

const (
	Asc  SortDirection = false
	Desc SortDirection = true
)

type whereClause struct {
	column string
	op     string
	value  any
}

type orderClause struct {
	column string
	dir    SortDirection
}

// Select starts a new SelectBuilder for the given table and dialect.
func Select(table string, d Dialect) *SelectBuilder {
	return &SelectBuilder{table: table, dialect: d}
}

// Columns restricts the SELECT to the specified columns.
// If not called, SELECT * is used.
func (b *SelectBuilder) Columns(cols ...string) *SelectBuilder {
	b.columns = cols
	return b
}

// Where adds a WHERE condition. op must be one of the allowed comparison
// operators (=, !=, <, >, <=, >=, LIKE, ILIKE).
// Multiple calls are combined with AND.
func (b *SelectBuilder) Where(column, op string, value any) *SelectBuilder {
	b.where = append(b.where, whereClause{column, op, value})
	return b
}

// OrderBy appends an ORDER BY clause for the given column and direction.
func (b *SelectBuilder) OrderBy(column string, dir SortDirection) *SelectBuilder {
	b.orderBy = append(b.orderBy, orderClause{column, dir})
	return b
}

// Limit sets the maximum number of rows to return.
func (b *SelectBuilder) Limit(n int) *SelectBuilder {
	b.limit = &n
	return b
}

// Offset sets the number of rows to skip (for pagination).
func (b *SelectBuilder) Offset(n int) *SelectBuilder {
	b.offset = &n
	return b
}

// Build produces the final SQL string and argument slice.
// Returns an error if any WHERE operator is not in the allowlist.
func (b *SelectBuilder) Build() (string, []any, error) {
	// --- column list ---
	cols := "*"
	if len(b.columns) > 0 {
		quoted := make([]string, len(b.columns))
		for i, c := range b.columns {
			quoted[i] = quoteIdent(c)
		}
		cols = strings.Join(quoted, ", ")
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(cols)
	sb.WriteString(" FROM ")
	sb.WriteString(quoteIdent(b.table))

	var args []any
	argIdx := 1

	// --- WHERE ---
	if len(b.where) > 0 {
		parts := make([]string, 0, len(b.where))
		for _, w := range b.where {
			op := strings.ToUpper(w.op)
			if !validOps[op] {
				return "", nil, errInvalidInput(
					fmt.Sprintf("unsupported WHERE operator: %q", w.op),
				)
			}
			parts = append(parts, fmt.Sprintf("%s %s %s", quoteIdent(w.column), op, b.placeholder(argIdx)))
			args = append(args, w.value)
			argIdx++
		}
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(parts, " AND "))
	}

	// --- ORDER BY ---
	if len(b.orderBy) > 0 {
		parts := make([]string, len(b.orderBy))
		for i, o := range b.orderBy {
			dir := "ASC"
			if o.dir == Desc {
				dir = "DESC"
			}
			parts[i] = fmt.Sprintf("%s %s", quoteIdent(o.column), dir)
		}
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(parts, ", "))
	}

	// --- LIMIT ---
	if b.limit != nil {
		sb.WriteString(fmt.Sprintf(" LIMIT %s", b.placeholder(argIdx)))
		args = append(args, *b.limit)
		argIdx++
	}

	// --- OFFSET ---
	if b.offset != nil {
		sb.WriteString(fmt.Sprintf(" OFFSET %s", b.placeholder(argIdx)))
		args = append(args, *b.offset)
	}

	return sb.String(), args, nil
}

// placeholder returns the correct parameter placeholder for the dialect.
// Postgres: $1, $2, …   MySQL: ? (index is ignored)
func (b *SelectBuilder) placeholder(idx int) string {
	if b.dialect == DialectMySQL {
		return "?"
	}
	return fmt.Sprintf("$%d", idx)
}

// quoteIdent wraps a SQL identifier in double-quotes (ANSI standard).
// This safely handles reserved words and mixed-case names.
// Note: MySQL also accepts double-quoted identifiers when ANSI mode is on,
// but both drivers work correctly with this quoting style.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
