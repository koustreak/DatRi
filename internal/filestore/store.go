// Package filestore defines the unified interface for file/object storage backends.
//
// All providers (MinIO, S3, Azure Blob, …) implement the Store interface.
// Callers depend only on this package — never on a specific provider package.
//
// Usage:
//
//	cfg := filestore.DefaultConfig("localhost:9000", "minioadmin", "minioadmin")
//	store, err := minio.New(ctx, cfg)
//	if err != nil { ... }
//	defer store.Close()
//
//	buckets, err := store.ListBuckets(ctx)
package filestore

import (
	"context"
	"time"
)

// Store is the single interface all file storage providers must implement.
// Currently scoped to GET (read) operations only.
type Store interface {
	// Ping verifies the storage backend is reachable.
	Ping(ctx context.Context) error

	// Close releases any held resources (connections, goroutines, etc.).
	Close() error

	// ListBuckets returns all buckets / containers accessible with the configured credentials.
	ListBuckets(ctx context.Context) ([]BucketInfo, error)

	// ListObjects returns the objects in bucket that match opts.
	// Virtual directory entries (common prefixes) are included when opts.Recursive is false.
	ListObjects(ctx context.Context, bucket string, opts ListOptions) ([]ObjectInfo, error)

	// GetObject opens a streaming handle to the object at key inside bucket.
	// The caller MUST call Object.Close() after reading.
	GetObject(ctx context.Context, bucket, key string) (Object, error)

	// StatObject returns metadata for the object at key inside bucket
	// without downloading its content.
	StatObject(ctx context.Context, bucket, key string) (*ObjectInfo, error)

	// PresignGetURL returns a time-limited URL that allows anyone to download
	// the object at key inside bucket without credentials.
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
}
