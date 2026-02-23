package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/koustreak/DatRi/internal/database"
)

// PostgreSQL SQLSTATE error codes (read-relevant only)
// Full list: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgErrConnectionFailure = "08006"
	pgErrSyntaxError       = "42601"
	pgErrUndefinedTable    = "42P01"
	pgErrUndefinedColumn   = "42703"
)

// mapError converts a pgx error into a DatRi DBError
func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &database.DBError{
			Kind:    database.ErrKindNotFound,
			Message: "record not found",
			Cause:   err,
		}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrConnectionFailure:
			return &database.DBError{
				Kind:    database.ErrKindConnection,
				Message: "database connection failed",
				Cause:   err,
			}
		case pgErrSyntaxError, pgErrUndefinedTable, pgErrUndefinedColumn:
			return &database.DBError{
				Kind:    database.ErrKindQuery,
				Message: fmt.Sprintf("query error: %s", pgErr.Message),
				Cause:   err,
			}
		}
	}

	return &database.DBError{
		Kind:    database.ErrKindUnknown,
		Message: err.Error(),
		Cause:   err,
	}
}
