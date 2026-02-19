package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/koustreak/DatRi/internal/database"
)

// pgErrCode maps PostgreSQL error codes to DatRi error kinds
// Full list: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
	pgErrNotNullViolation    = "23502"
	pgErrConnectionFailure   = "08006"
	pgErrSyntaxError         = "42601"
)

// mapError converts a pgx error into a DatRi DBError
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// No rows
	if errors.Is(err, pgx.ErrNoRows) {
		return &database.DBError{
			Kind:    database.ErrKindNotFound,
			Message: "record not found",
			Cause:   err,
		}
	}

	// Postgres-specific errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrUniqueViolation:
			return &database.DBError{
				Kind:    database.ErrKindConflict,
				Message: fmt.Sprintf("conflict: %s", pgErr.Detail),
				Cause:   err,
			}
		case pgErrForeignKeyViolation:
			return &database.DBError{
				Kind:    database.ErrKindConflict,
				Message: fmt.Sprintf("foreign key violation: %s", pgErr.Detail),
				Cause:   err,
			}
		case pgErrNotNullViolation:
			return &database.DBError{
				Kind:    database.ErrKindInvalid,
				Message: fmt.Sprintf("not null violation: %s", pgErr.Detail),
				Cause:   err,
			}
		case pgErrConnectionFailure:
			return &database.DBError{
				Kind:    database.ErrKindConnection,
				Message: "database connection failed",
				Cause:   err,
			}
		case pgErrSyntaxError:
			return &database.DBError{
				Kind:    database.ErrKindQuery,
				Message: fmt.Sprintf("invalid query: %s", pgErr.Message),
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
