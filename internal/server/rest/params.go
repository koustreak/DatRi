package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/errs"
)

// QueryParams holds the parsed, validated values from the URL query string.
// Option-A style: ?limit=20&offset=0&order_by=id&order_dir=desc&name=alice&age=30
type QueryParams struct {
	Limit    *int
	Offset   *int
	OrderBy  string
	OrderDir database.SortDirection // Asc | Desc

	// Filters holds col→value pairs extracted from all remaining query params.
	// Each becomes a WHERE col = value clause (equality only for now).
	Filters []filterClause
}

type filterClause struct {
	Column string
	Value  string
}

// reservedParams is the set of query keys consumed by the framework.
// Everything else is treated as a column filter.
var reservedParams = map[string]bool{
	"limit":     true,
	"offset":    true,
	"order_by":  true,
	"order_dir": true,
}

// parseQueryParams reads the URL query string and returns a validated
// QueryParams. It never returns a hard error — invalid values are silently
// ignored and defaults are applied, matching the principle of least surprise
// for an auto-generated API.
func parseQueryParams(r *http.Request, table *database.TableInfo) (*QueryParams, error) {
	q := r.URL.Query()
	p := &QueryParams{}

	// --- limit ---
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return nil, errs.New(errs.ErrKindInvalidInput,
				fmt.Sprintf("limit must be a non-negative integer, got %q", raw))
		}
		if n > maxLimit {
			n = maxLimit
		}
		p.Limit = &n
	}

	// --- offset ---
	if raw := q.Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return nil, errs.New(errs.ErrKindInvalidInput,
				fmt.Sprintf("offset must be a non-negative integer, got %q", raw))
		}
		p.Offset = &n
	}

	// --- order_by ---
	if col := q.Get("order_by"); col != "" {
		if table != nil && !columnExists(table, col) {
			return nil, errs.New(errs.ErrKindInvalidInput,
				fmt.Sprintf("unknown column for order_by: %q", col))
		}
		p.OrderBy = col
	}

	// --- order_dir ---
	switch strings.ToLower(q.Get("order_dir")) {
	case "desc":
		p.OrderDir = database.Desc
	default:
		p.OrderDir = database.Asc
	}

	// --- column filters (everything that is not a reserved key) ---
	for key, vals := range q {
		if reservedParams[key] {
			continue
		}
		// Validate that the column actually exists in the schema to prevent
		// arbitrary SQL injection through the column name position.
		if table != nil && !columnExists(table, key) {
			return nil, errs.New(errs.ErrKindInvalidInput,
				fmt.Sprintf("unknown filter column: %q", key))
		}
		// Use only the first value for simplicity (multi-value = future work).
		p.Filters = append(p.Filters, filterClause{Column: key, Value: vals[0]})
	}

	return p, nil
}

// columnExists reports whether the table has a column with the given name.
func columnExists(t *database.TableInfo, name string) bool {
	for _, c := range t.Columns {
		if c.Name == name {
			return true
		}
	}
	return false
}

// maxLimit caps the number of rows a single request may fetch.
// Prevents huge accidental queries while still being generous.
const maxLimit = 1000

// applyToBuilder applies the parsed query params onto a SelectBuilder.
func (p *QueryParams) applyToBuilder(b *database.SelectBuilder) *database.SelectBuilder {
	for _, f := range p.Filters {
		b = b.Where(f.Column, "=", f.Value)
	}
	if p.OrderBy != "" {
		b = b.OrderBy(p.OrderBy, p.OrderDir)
	}
	if p.Limit != nil {
		b = b.Limit(*p.Limit)
	}
	if p.Offset != nil {
		b = b.Offset(*p.Offset)
	}
	return b
}
