package mysql

import (
	"errors"
	"fmt"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/koustreak/DatRi/internal/database"
)

// MySQL error numbers (read-relevant only)
// Full list: https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
const (
	errBadFieldError   = 1054 // unknown column
	errNoSuchTable     = 1146 // table doesn't exist
	errAccessDenied    = 1045 // bad credentials
	errConnRefused     = 2003 // can't connect
	errUnknownDatabase = 1049 // database doesn't exist
)

// mapError converts a MySQL driver error into a DatRi DBError
func mapError(err error) error {
	if err == nil {
		return nil
	}

	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case errAccessDenied, errConnRefused, errUnknownDatabase:
			return &database.DBError{
				Kind:    database.ErrKindConnection,
				Message: fmt.Sprintf("connection error: %s", mysqlErr.Message),
				Cause:   err,
			}
		case errBadFieldError, errNoSuchTable:
			return &database.DBError{
				Kind:    database.ErrKindQuery,
				Message: fmt.Sprintf("query error: %s", mysqlErr.Message),
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
