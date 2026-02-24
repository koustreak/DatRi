package database

import (
	"errors"
	"fmt"
)

// ErrKind categorises a database error without exposing driver-specific codes.
type ErrKind int

const (
	ErrKindUnknown          ErrKind = iota
	ErrKindNotFound                 // no rows matched the query
	ErrKindConnectionFailed         // could not reach or authenticate to the DB
	ErrKindTimeout                  // query or connection exceeded its deadline
	ErrKindQueryFailed              // SQL syntax or runtime execution error
	ErrKindInvalidInput             // caller passed bad arguments (e.g. unknown table)
)

func (k ErrKind) String() string {
	switch k {
	case ErrKindNotFound:
		return "not_found"
	case ErrKindConnectionFailed:
		return "connection_failed"
	case ErrKindTimeout:
		return "timeout"
	case ErrKindQueryFailed:
		return "query_failed"
	case ErrKindInvalidInput:
		return "invalid_input"
	default:
		return "unknown"
	}
}

// DBError is the single error type returned by all database operations.
// Drivers translate their native errors into DBError before returning them.
type DBError struct {
	Kind    ErrKind
	Message string
	Cause   error // original driver-level error, for logging/debugging
}

func (e *DBError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Kind, e.Message)
}

func (e *DBError) Unwrap() error {
	return e.Cause
}

// --- Constructor helpers used by drivers ---

func errNotFound(msg string) *DBError {
	return &DBError{Kind: ErrKindNotFound, Message: msg}
}

func errConnection(msg string, cause error) *DBError {
	return &DBError{Kind: ErrKindConnectionFailed, Message: msg, Cause: cause}
}

func errTimeout(msg string, cause error) *DBError {
	return &DBError{Kind: ErrKindTimeout, Message: msg, Cause: cause}
}

func errQuery(msg string, cause error) *DBError {
	return &DBError{Kind: ErrKindQueryFailed, Message: msg, Cause: cause}
}

func errInvalidInput(msg string) *DBError {
	return &DBError{Kind: ErrKindInvalidInput, Message: msg}
}

// --- Public predicates for callers ---

// IsNotFound reports whether err represents a "no rows" result.
func IsNotFound(err error) bool {
	return kindOf(err) == ErrKindNotFound
}

// IsTimeout reports whether err was caused by a deadline or context cancellation.
func IsTimeout(err error) bool {
	return kindOf(err) == ErrKindTimeout
}

// IsConnectionFailed reports whether err is a connectivity or auth failure.
func IsConnectionFailed(err error) bool {
	return kindOf(err) == ErrKindConnectionFailed
}

// IsQueryFailed reports whether err is a SQL execution error.
func IsQueryFailed(err error) bool {
	return kindOf(err) == ErrKindQueryFailed
}

// IsInvalidInput reports whether err was caused by bad input from the caller.
func IsInvalidInput(err error) bool {
	return kindOf(err) == ErrKindInvalidInput
}

func kindOf(err error) ErrKind {
	var dbErr *DBError
	if errors.As(err, &dbErr) {
		return dbErr.Kind
	}
	return ErrKindUnknown
}
