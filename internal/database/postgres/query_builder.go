package postgres

import (
	"context"
	"fmt"

	"github.com/koustreak/DatRi/internal/database"
)

// BuildQuery builds a parameterized SELECT query for PostgreSQL ($1, $2, ...)
func BuildQuery(opts database.ListOptions) (database.Query, error) {
	return database.ExportBuildQuery(opts, func(n int) string {
		return fmt.Sprintf("$%d", n)
	})
}

// List executes a ListOptions query and returns all matching rows as maps
func (db *DB) List(ctx context.Context, opts database.ListOptions) ([]map[string]any, error) {
	q, err := BuildQuery(opts)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx, q.SQL, q.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// scanRows converts database.Rows into a slice of maps
func scanRows(rows database.Rows) ([]map[string]any, error) {
	// pgx exposes column names via FieldDescriptions — use raw pool for that
	// For now return raw rows; column mapping is handled at server layer
	var results []map[string]any
	_ = rows // placeholder until server layer maps columns
	return results, rows.Err()
}
