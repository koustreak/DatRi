package minio

import (
	"context"
	"errors"
	"net/http"

	"github.com/koustreak/DatRi/internal/errs"
	minioErr "github.com/minio/minio-go/v7"
)

// mapError translates a MinIO SDK error into a *errs.Error.
// It mirrors the mapError pattern used in the postgres and mysql drivers.
func mapError(err error, msg string) *errs.Error {
	if err == nil {
		return nil
	}

	// Context cancellation / deadline
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errs.Wrap(errs.ErrKindTimeout, msg, err)
	}

	// MinIO SDK exposes a typed ErrorResponse for S3-protocol errors
	var resp minioErr.ErrorResponse
	if errors.As(err, &resp) {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return errs.Wrap(errs.ErrKindNotFound, msg, err)
		case http.StatusForbidden, http.StatusUnauthorized:
			return errs.Wrap(errs.ErrKindPermissionDenied, msg, err)
		case http.StatusBadRequest:
			return errs.Wrap(errs.ErrKindInvalidInput, msg, err)
		}

		// S3 error codes for "not found" that may arrive with 200-range status
		switch resp.Code {
		case "NoSuchBucket", "NoSuchKey", "NoSuchUpload":
			return errs.Wrap(errs.ErrKindNotFound, msg, err)
		case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
			return errs.Wrap(errs.ErrKindPermissionDenied, msg, err)
		case "InvalidBucketName", "InvalidObjectName", "KeyTooLongError":
			return errs.Wrap(errs.ErrKindInvalidInput, msg, err)
		case "RequestTimeout", "SlowDown":
			return errs.Wrap(errs.ErrKindTimeout, msg, err)
		}
	}

	// Anything else — treat as a generic connection / I/O failure
	return errs.Wrap(errs.ErrKindConnectionFailed, msg, err)
}
