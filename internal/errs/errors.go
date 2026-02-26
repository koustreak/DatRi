// Package errs provides the unified error type used across all of DatRi.
//
// Every subsystem (database, filestore, server, …) wraps its native errors
// into *errs.Error before returning them to callers. Callers use the Is*
// predicates to handle errors without importing driver-specific packages.
//
// Usage:
//
//	// In a driver — wrap native errors:
//	return errs.Wrap(errs.ErrKindTimeout, "query timed out", pgErr)
//
//	// In a handler — check error kind:
//	if errs.IsNotFound(err) {
//	    http.Error(w, "not found", http.StatusNotFound)
//	}
package errs

import (
	"errors"
	"fmt"
)

// ErrKind categorises an error without exposing subsystem-specific codes.
// All backends (Postgres, MySQL, MinIO, …) map their native errors to one
// of these kinds, giving callers a single consistent API.
type ErrKind int

const (
	ErrKindUnknown          ErrKind = iota
	ErrKindNotFound                 // no rows, no object, no bucket
	ErrKindConnectionFailed         // cannot reach the backend
	ErrKindTimeout                  // context deadline / cancellation
	ErrKindQueryFailed              // SQL or storage operation error
	ErrKindInvalidInput             // bad arguments from the caller
	ErrKindPermissionDenied         // access denied / auth failure
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
	case ErrKindPermissionDenied:
		return "permission_denied"
	default:
		return "unknown"
	}
}

// Error is the single error type returned by all DatRi subsystems.
// Drivers produce it; callers inspect it via the Is* predicates below.
type Error struct {
	Kind    ErrKind
	Message string
	Cause   error // original driver-level error, preserved for logging
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Kind, e.Message)
}

// Unwrap allows errors.Is / errors.As to traverse the cause chain.
func (e *Error) Unwrap() error {
	return e.Cause
}

// --- Constructors ---

// New creates an *Error with the given kind and message and no cause.
func New(kind ErrKind, msg string) *Error {
	return &Error{Kind: kind, Message: msg}
}

// Wrap creates an *Error with the given kind, message, and an underlying cause.
func Wrap(kind ErrKind, msg string, cause error) *Error {
	return &Error{Kind: kind, Message: msg, Cause: cause}
}

// --- Predicates ---

// IsNotFound reports whether err represents a "not found" result
// (no rows, missing object, unknown table/bucket, …).
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

// IsQueryFailed reports whether err is a backend operation failure
// (SQL execution error, storage I/O error, …).
func IsQueryFailed(err error) bool {
	return kindOf(err) == ErrKindQueryFailed
}

// IsInvalidInput reports whether err was caused by bad input from the caller.
func IsInvalidInput(err error) bool {
	return kindOf(err) == ErrKindInvalidInput
}

// IsPermissionDenied reports whether err is an access control failure.
func IsPermissionDenied(err error) bool {
	return kindOf(err) == ErrKindPermissionDenied
}

// kindOf extracts the ErrKind from any error in the chain.
func kindOf(err error) ErrKind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ErrKindUnknown
}
