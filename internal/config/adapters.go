package config

import (
	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/filestore"
)

// ToDatabaseConfig converts this resource's DatabaseConfig into the type
// expected by the internal/database package.
// Only call this when ResourceConfig.Type == "database".
func (d *DatabaseConfig) ToDatabaseConfig() *database.Config {
	return &database.Config{
		Driver:          database.Driver(d.Driver),
		DSN:             d.DSN,
		MaxConns:        d.Pool.MaxConns,
		MinConns:        d.Pool.MinConns,
		MaxConnLifetime: d.Pool.MaxConnLifetime,
		MaxConnIdleTime: d.Pool.MaxConnIdleTime,
		ConnectTimeout:  d.Timeouts.Connect,
		QueryTimeout:    d.Timeouts.Query,
	}
}

// ToFilestoreConfig converts this resource's FilestoreConfig into the type
// expected by the internal/filestore package.
// Only call this when ResourceConfig.Type == "filestore".
func (f *FilestoreConfig) ToFilestoreConfig() *filestore.Config {
	return &filestore.Config{
		Provider:      filestore.Provider(f.Provider),
		Endpoint:      f.Endpoint,
		AccessKey:     f.AccessKey,
		SecretKey:     f.SecretKey,
		UseSSL:        f.UseSSL,
		Region:        f.Region,
		DefaultBucket: f.DefaultBucket,
	}
}
