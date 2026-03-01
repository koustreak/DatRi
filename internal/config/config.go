// Package config defines the top-level configuration for DatRi.
//
// DatRi supports multiple independently-served resources in a single
// datri.yaml file. Each resource gets its own port, protocol list,
// database/filestore connection, and optionally its own auth and CORS rules.
//
// Typical usage:
//
//	cfg, err := config.Load("datri.yaml")
//	if err != nil { log.Fatal(err) }
//
//	for _, r := range cfg.Resources {
//	    dbCfg  := r.Database.ToDatabaseConfig()
//	    auth   := r.ResolvedAuth(cfg.Global)
//	    cors   := r.ResolvedCORS(cfg.Global)
//	}
package config

import "time"

// ─── Root ─────────────────────────────────────────────────────────────────────

// Config is the root configuration object parsed from datri.yaml.
type Config struct {
	// Global holds defaults shared by all resources.
	// Resource-level settings always override these.
	Global GlobalConfig `yaml:"global"`

	// Resources is the list of independently served API resources.
	// Each resource runs on its own port with its own server.
	Resources []ResourceConfig `yaml:"resources"`
}

// ─── Global ───────────────────────────────────────────────────────────────────

// GlobalConfig holds defaults that apply to every resource unless overridden.
type GlobalConfig struct {
	// Log controls the logging behaviour for the entire process.
	Log LogConfig `yaml:"log"`

	// Auth is the default authentication config applied to all resources
	// that do not declare their own auth block.
	Auth *AuthConfig `yaml:"auth"`

	// CORS is the default CORS config applied to all resources
	// that do not declare their own cors block.
	CORS *CORSConfig `yaml:"cors"`
}

// ─── Resource ─────────────────────────────────────────────────────────────────

// ResourceConfig represents a single independently-served API resource.
// Each resource binds to its own port and runs its own server goroutine.
type ResourceConfig struct {
	// Name is a unique human-readable identifier used in logs and error messages.
	// Example: "user-service", "analytics-api"
	Name string `yaml:"name"`

	// Type declares what backs this resource.
	// Allowed: "database", "filestore"
	Type string `yaml:"type"`

	// Server controls the network binding and enabled protocols for this resource.
	Server ServerConfig `yaml:"server"`

	// Database is required when Type is "database".
	// Must be nil when Type is "filestore".
	Database *DatabaseConfig `yaml:"database"`

	// Filestore is required when Type is "filestore".
	// Must be nil when Type is "database".
	Filestore *FilestoreConfig `yaml:"filestore"`

	// Auth overrides the global auth config for this resource only.
	// If nil, the global auth config is used.
	Auth *AuthConfig `yaml:"auth"`

	// CORS overrides the global CORS config for this resource only.
	// If nil, the global CORS config is used.
	CORS *CORSConfig `yaml:"cors"`
}

// ResolvedAuth returns the effective AuthConfig for this resource.
// Resource-level auth takes priority; falls back to global if not set.
// Returns a disabled AuthConfig if neither is configured.
func (r *ResourceConfig) ResolvedAuth(g GlobalConfig) AuthConfig {
	if r.Auth != nil {
		return *r.Auth
	}
	if g.Auth != nil {
		return *g.Auth
	}
	return AuthConfig{Enabled: false}
}

// ResolvedCORS returns the effective CORSConfig for this resource.
// Resource-level CORS takes priority; falls back to global if not set.
// Returns a permissive default if neither is configured.
func (r *ResourceConfig) ResolvedCORS(g GlobalConfig) CORSConfig {
	if r.CORS != nil {
		return *r.CORS
	}
	if g.CORS != nil {
		return *g.CORS
	}
	return defaultCORS()
}

// ─── Server ───────────────────────────────────────────────────────────────────

// ServerConfig controls the network binding and protocols for one resource.
type ServerConfig struct {
	// Host is the network address to bind to. Default: "0.0.0.0"
	Host string `yaml:"host"`

	// Port is the TCP port for this resource's API server.
	// Must be unique across all resources. Required.
	Port int `yaml:"port"`

	// Protocols is the list of API protocols to enable for this resource.
	// Allowed: rest, graphql, grpc, websocket
	// Default: [rest]
	Protocols []string `yaml:"protocols"`
}

// ─── Database ─────────────────────────────────────────────────────────────────

// DatabaseConfig holds all settings needed to connect to a SQL database.
type DatabaseConfig struct {
	// Driver is the database engine. Allowed: "postgres", "mysql"
	Driver string `yaml:"driver"`

	// DSN is the full data source name / connection string.
	// Postgres: "postgres://user:pass@localhost:5432/mydb"
	// MySQL:    "user:pass@tcp(localhost:3306)/mydb"
	DSN string `yaml:"dsn"`

	// Pool controls the connection pool behaviour.
	Pool PoolConfig `yaml:"pool"`

	// Timeouts controls per-operation deadlines.
	Timeouts TimeoutConfig `yaml:"timeouts"`
}

// PoolConfig controls the database connection pool behaviour.
type PoolConfig struct {
	// MaxConns is the maximum number of connections in the pool. Default: 25
	MaxConns int32 `yaml:"max_conns"`

	// MinConns is the minimum number of idle connections kept alive. Default: 5
	MinConns int32 `yaml:"min_conns"`

	// MaxConnLifetime is how long a connection may be reused. Default: 30m
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`

	// MaxConnIdleTime is how long a connection may sit idle. Default: 5m
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
}

// TimeoutConfig controls per-operation deadlines.
type TimeoutConfig struct {
	// Connect is the time limit to establish a new connection. Default: 10s
	Connect time.Duration `yaml:"connect"`

	// Query is the default per-query deadline. Default: 30s
	Query time.Duration `yaml:"query"`
}

// ─── Filestore ────────────────────────────────────────────────────────────────

// FilestoreConfig holds all settings needed to connect to an object storage backend.
type FilestoreConfig struct {
	// Provider is the storage backend. Allowed: "minio", "s3"
	Provider string `yaml:"provider"`

	// Endpoint is the host:port of the storage server.
	// Example: "localhost:9000" for local MinIO.
	Endpoint string `yaml:"endpoint"`

	// AccessKey is the access key ID (MinIO / S3 style).
	// Use env var DATRI_RESOURCE_<NAME>_ACCESS_KEY to avoid storing in yaml.
	AccessKey string `yaml:"access_key"`

	// SecretKey is the secret access key.
	// Use env var DATRI_RESOURCE_<NAME>_SECRET_KEY to avoid storing in yaml.
	SecretKey string `yaml:"secret_key"`

	// UseSSL controls whether TLS is used for the connection. Default: false
	UseSSL bool `yaml:"use_ssl"`

	// Region is used by region-aware backends (e.g. AWS S3).
	// Leave empty for MinIO.
	Region string `yaml:"region"`

	// DefaultBucket is an optional default bucket name.
	DefaultBucket string `yaml:"default_bucket"`
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

// AuthConfig controls authentication for a resource or globally.
// Declared under global.auth or resource.auth.
type AuthConfig struct {
	// Enabled turns authentication on or off. Default: false
	Enabled bool `yaml:"enabled"`

	// Type is the authentication mechanism. Allowed: "jwt", "apikey"
	Type string `yaml:"type"`

	// Secret is the signing secret for JWT tokens.
	// Required when Type is "jwt".
	// Prefer DATRI_AUTH_SECRET env var over storing here.
	Secret string `yaml:"secret"`
}

// ─── CORS ─────────────────────────────────────────────────────────────────────

// CORSConfig controls cross-origin resource sharing for a resource or globally.
// Declared under global.cors or resource.cors.
type CORSConfig struct {
	// Enabled turns CORS middleware on or off. Default: true
	Enabled bool `yaml:"enabled"`

	// Origins is the list of allowed origins. Use ["*"] to allow all.
	// Default: ["*"]
	Origins []string `yaml:"origins"`

	// Methods is the list of allowed HTTP methods.
	// Default: [GET, POST, PUT, DELETE, OPTIONS]
	Methods []string `yaml:"methods"`

	// Headers is the list of allowed request headers.
	// Default: [Content-Type, Authorization]
	Headers []string `yaml:"headers"`
}

// ─── Log ──────────────────────────────────────────────────────────────────────

// LogConfig controls logging for the entire DatRi process.
// Declared only under global.log (not per-resource).
type LogConfig struct {
	// Level is the minimum log level to emit.
	// Allowed: "debug", "info", "warn", "error". Default: "info"
	Level string `yaml:"level"`

	// Format controls the output format.
	// Allowed: "json", "console". Default: "console"
	Format string `yaml:"format"`
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func defaultCORS() CORSConfig {
	return CORSConfig{
		Enabled: true,
		Origins: []string{"*"},
		Methods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		Headers: []string{"Content-Type", "Authorization"},
	}
}
