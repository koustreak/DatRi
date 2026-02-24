package database

import "time"

// Driver identifies the database engine.
type Driver string

const (
	DriverPostgres Driver = "postgres"
	DriverMySQL    Driver = "mysql"
)

// Config holds all settings needed to connect to and pool a database.
type Config struct {
	// Driver is the database engine (e.g. DriverPostgres).
	Driver Driver

	// DSN is the full data source name / connection string.
	// Example: "postgres://user:pass@localhost:5432/mydb"
	DSN string

	// Pool tuning
	MaxConns        int32         // maximum number of connections in the pool
	MinConns        int32         // minimum number of idle connections kept alive
	MaxConnLifetime time.Duration // maximum time a connection may be reused
	MaxConnIdleTime time.Duration // maximum time a connection may sit idle

	// Timeouts
	ConnectTimeout time.Duration // time limit for establishing a new connection
	QueryTimeout   time.Duration // default per-query deadline (applied by callers)
}

// DefaultConfig returns production-ready pool settings for the given DSN.
// These defaults are tuned for a high-throughput read-heavy workload.
func DefaultConfig(dsn string) *Config {
	return &Config{
		Driver:          DriverPostgres,
		DSN:             dsn,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		ConnectTimeout:  10 * time.Second,
		QueryTimeout:    30 * time.Second,
	}
}
