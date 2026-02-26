// Package minio provides a MinIO implementation of filestore.Store.
//
// Usage:
//
//	cfg := filestore.DefaultConfig("localhost:9000", "minioadmin", "minioadmin")
//	store, err := minio.New(ctx, cfg)
//	if err != nil { ... }
//	defer store.Close()
//
//	buckets, err := store.ListBuckets(ctx)
package minio

import (
	"context"
	"io"
	"time"

	"github.com/koustreak/DatRi/internal/errs"
	"github.com/koustreak/DatRi/internal/filestore"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Driver is a MinIO implementation of filestore.Store.
// It is safe for concurrent use by multiple goroutines.
type Driver struct {
	client *miniogo.Client
}

// New connects to MinIO using the provided Config and returns a Driver.
// It calls Ping to validate the connection before returning.
func New(ctx context.Context, cfg *filestore.Config) (*Driver, error) {
	client, err := miniogo.New(cfg.Endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, errs.Wrap(errs.ErrKindConnectionFailed, "failed to create minio client", err)
	}

	d := &Driver{client: client}

	if err := d.Ping(ctx); err != nil {
		return nil, err
	}

	return d, nil
}

// --- filestore.Store implementation ---

// Ping verifies the MinIO server is reachable by listing buckets.
func (d *Driver) Ping(ctx context.Context) error {
	_, err := d.client.ListBuckets(ctx)
	if err != nil {
		return mapError(err, "ping failed")
	}
	return nil
}

// Close is a no-op for MinIO — the SDK client holds no persistent connections.
func (d *Driver) Close() error {
	return nil
}

// ListBuckets returns all buckets accessible with the configured credentials.
func (d *Driver) ListBuckets(ctx context.Context) ([]filestore.BucketInfo, error) {
	raw, err := d.client.ListBuckets(ctx)
	if err != nil {
		return nil, mapError(err, "failed to list buckets")
	}

	buckets := make([]filestore.BucketInfo, len(raw))
	for i, b := range raw {
		buckets[i] = filestore.BucketInfo{
			Name:      b.Name,
			CreatedAt: b.CreationDate,
		}
	}
	return buckets, nil
}

// ListObjects returns objects in bucket that match opts.
func (d *Driver) ListObjects(ctx context.Context, bucket string, opts filestore.ListOptions) ([]filestore.ObjectInfo, error) {
	listOpts := miniogo.ListObjectsOptions{
		Prefix:    opts.Prefix,
		Recursive: opts.Recursive,
	}

	var results []filestore.ObjectInfo
	count := 0

	for obj := range d.client.ListObjects(ctx, bucket, listOpts) {
		if obj.Err != nil {
			return nil, mapError(obj.Err, "failed to list objects")
		}

		results = append(results, filestore.ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
			IsDir:        obj.Key[len(obj.Key)-1] == '/',
		})

		count++
		if opts.Limit > 0 && count >= opts.Limit {
			break
		}
	}

	return results, nil
}

// GetObject opens a streaming handle to the object at key inside bucket.
// The caller MUST call Object.Close() after reading.
func (d *Driver) GetObject(ctx context.Context, bucket, key string) (filestore.Object, error) {
	obj, err := d.client.GetObject(ctx, bucket, key, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, mapError(err, "failed to get object")
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, mapError(err, "failed to stat object after get")
	}

	return &object{
		ReadCloser: obj,
		info: &filestore.ObjectInfo{
			Key:          key,
			Size:         stat.Size,
			ContentType:  stat.ContentType,
			ETag:         stat.ETag,
			LastModified: stat.LastModified,
		},
	}, nil
}

// StatObject returns metadata for the object at key inside bucket
// without downloading its content.
func (d *Driver) StatObject(ctx context.Context, bucket, key string) (*filestore.ObjectInfo, error) {
	stat, err := d.client.StatObject(ctx, bucket, key, miniogo.StatObjectOptions{})
	if err != nil {
		return nil, mapError(err, "failed to stat object")
	}

	return &filestore.ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}

// PresignGetURL returns a time-limited public download URL for the object.
func (d *Driver) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	u, err := d.client.PresignedGetObject(ctx, bucket, key, ttl, nil)
	if err != nil {
		return "", mapError(err, "failed to generate presigned URL")
	}
	return u.String(), nil
}

// --- internal types ---

// object wraps a MinIO GetObject response and exposes filestore.Object.
type object struct {
	io.ReadCloser
	info *filestore.ObjectInfo
}

func (o *object) Info() *filestore.ObjectInfo {
	return o.info
}
