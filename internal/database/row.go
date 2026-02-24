package database

// ScanRows reads all rows from the result set and returns them as a slice
// of maps, where each key is the column name and each value is the Go-native
// representation of the DB value.
//
// The returned slice is always non-nil (empty slice on zero rows).
// ScanRows always closes the Rows — callers do not need to call Close().
func ScanRows(rows Rows) ([]map[string]any, error) {
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, errQuery("failed to read column names", err)
	}

	result := make([]map[string]any, 0)

	for rows.Next() {
		// Allocate scan targets as *any so the driver can write any type.
		dest := make([]any, len(columns))
		destPtrs := make([]any, len(columns))
		for i := range dest {
			destPtrs[i] = &dest[i]
		}

		if err := rows.Scan(destPtrs...); err != nil {
			return nil, errQuery("failed to scan row", err)
		}

		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = dest[i]
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, errQuery("error during row iteration", err)
	}

	return result, nil
}

// ScanRow reads a single row and returns it as a map.
// Returns ErrKindNotFound if the row does not exist.
func ScanRow(row Row, columns []string) (map[string]any, error) {
	dest := make([]any, len(columns))
	destPtrs := make([]any, len(columns))
	for i := range dest {
		destPtrs[i] = &dest[i]
	}

	if err := row.Scan(destPtrs...); err != nil {
		return nil, errQuery("failed to scan single row", err)
	}

	result := make(map[string]any, len(columns))
	for i, col := range columns {
		result[col] = dest[i]
	}
	return result, nil
}
