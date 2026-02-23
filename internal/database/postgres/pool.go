package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koustreak/DatRi/internal/database"
)

const (
	defaultMaxConns    = 10
	defaultMinConns    = 2
	defaultConnTimeout = 5 * time.Second
)

// buildPool creates a pgxpool from the given config
func buildPool(ctx context.Context, cfg *database.Config) (*pgxpool.Pool, error) {
	dsn := buildDSN(cfg)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid postgres config: %w", err)
	}

	// Apply pool settings with defaults
	poolCfg.MaxConns = withDefault(cfg.MaxConns, defaultMaxConns)
	poolCfg.MinConns = withDefault(cfg.MinConns, defaultMinConns)
	poolCfg.MaxConnIdleTime = defaultConnTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, mapError(err)
	}

	return pool, nil
}

// buildDSN constructs the postgres connection string
func buildDSN(cfg *database.Config) string {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	port := cfg.Port
	if port == 0 {
		port = 5432
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, port, cfg.User, cfg.Password, cfg.Database, sslMode,
	)
}

// withDefault returns val if non-zero, otherwise returns def
func withDefault(val, def int32) int32 {
	if val == 0 {
		return def
	}
	return val
}
