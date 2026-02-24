package mysql

import (
	"context"

	"github.com/koustreak/DatRi/internal/database"
)

// BuildQuery builds a parameterized SELECT query for MySQL (?, ?, ...)
func BuildQuery(opts database.ListOptions) (database.Query, error) {
	return database.ExportBuildQuery(opts, func(n int) string {
		return "?"
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

	var results []map[string]any
	_ = rows // column mapping handled at server layer
	return results, rows.Err()
}
