package mysql

import (
	"errors"
	"fmt"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/koustreak/DatRi/internal/database"
)

// MySQL error numbers
// Full list: https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
const (
	errDuplicateEntry  = 1062
	errNoReferencedRow = 1452
	errRowIsReferenced = 1451
	errBadFieldError   = 1054
	errAccessDenied    = 1045
	errConnRefused     = 2003
	errUnknownDatabase = 1049
)

// mapError converts a MySQL driver error into a DatRi DBError
func mapError(err error) error {
	if err == nil {
		return nil
	}

	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case errDuplicateEntry:
			return &database.DBError{
				Kind:    database.ErrKindConflict,
				Message: fmt.Sprintf("conflict: %s", mysqlErr.Message),
				Cause:   err,
			}
		case errNoReferencedRow, errRowIsReferenced:
			return &database.DBError{
				Kind:    database.ErrKindConflict,
				Message: fmt.Sprintf("foreign key violation: %s", mysqlErr.Message),
				Cause:   err,
			}
		case errAccessDenied, errConnRefused, errUnknownDatabase:
			return &database.DBError{
				Kind:    database.ErrKindConnection,
				Message: fmt.Sprintf("connection error: %s", mysqlErr.Message),
				Cause:   err,
			}
		case errBadFieldError:
			return &database.DBError{
				Kind:    database.ErrKindQuery,
				Message: fmt.Sprintf("invalid query: %s", mysqlErr.Message),
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
