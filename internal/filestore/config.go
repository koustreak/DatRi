package filestore

// Provider identifies the file storage backend.
type Provider string

const (
	ProviderMinIO Provider = "minio"
)

// Config holds all settings needed to connect to a file storage backend.
type Config struct {
	// Provider is the storage backend (e.g. ProviderMinIO).
	Provider Provider

	// Endpoint is the host:port of the storage server.
	// Example: "localhost:9000" for local MinIO.
	Endpoint string

	// AccessKey is the access key ID (MinIO / S3 style).
	AccessKey string

	// SecretKey is the secret access key.
	SecretKey string

	// UseSSL controls whether TLS is used for the connection.
	UseSSL bool

	// Region is used by region-aware backends (e.g. AWS S3).
	// Leave empty for MinIO.
	Region string

	// DefaultBucket is an optional default bucket name.
	// Callers may override it per-request.
	DefaultBucket string
}

// DefaultConfig returns a sensible local-dev config for MinIO.
func DefaultConfig(endpoint, accessKey, secretKey string) *Config {
	return &Config{
		Provider:  ProviderMinIO,
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    false,
	}
}
