package filestore

import (
	"io"
	"time"
)

// BucketInfo describes a storage bucket / container.
type BucketInfo struct {
	// Name is the bucket name.
	Name string

	// CreatedAt is when the bucket was created.
	// May be zero if the backend does not expose creation time.
	CreatedAt time.Time
}

// ObjectInfo describes a single object stored in a bucket.
type ObjectInfo struct {
	// Key is the full object path within the bucket (e.g. "images/photo.jpg").
	Key string

	// Size is the byte size of the object. -1 if unknown.
	Size int64

	// ContentType is the MIME type (e.g. "image/jpeg").
	ContentType string

	// ETag is the object's entity tag / hash, as returned by the backend.
	ETag string

	// LastModified is when the object was last written.
	LastModified time.Time

	// IsDir is true when the entry represents a virtual directory (prefix),
	// not an actual stored object.
	IsDir bool
}

// Object is a streaming handle to an object's content.
// The caller MUST call Close() after reading to avoid resource leaks.
type Object interface {
	io.ReadCloser

	// Info returns the metadata for this object.
	Info() *ObjectInfo
}

// ListOptions controls how ListObjects filters and paginates results.
type ListOptions struct {
	// Prefix restricts results to objects whose key starts with this string.
	// Use "" to list everything in the bucket.
	Prefix string

	// Recursive, when true, lists all objects under the prefix without
	// grouping by virtual directories. When false (default), common prefixes
	// (virtual "folders") are returned as IsDir entries.
	Recursive bool

	// Limit caps the number of results returned. 0 means use the backend default.
	Limit int

	// Marker is the pagination cursor — the last key seen in a previous page.
	// Pass "" to start from the beginning.
	Marker string
}
