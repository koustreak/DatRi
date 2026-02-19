package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/koustreak/DatRi/internal/database"
)

const (
	defaultMaxOpenConns    = 10
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 10 * time.Minute
	defaultPort            = 3306
)

// buildPool configures and returns a *sql.DB with pool settings
func buildPool(cfg *database.Config) (*sql.DB, error) {
	dsn := buildDSN(cfg)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql: %w", err)
	}

	// Pool settings
	maxOpen := int(cfg.MaxConns)
	if maxOpen == 0 {
		maxOpen = defaultMaxOpenConns
	}
	maxIdle := int(cfg.MaxIdleConns)
	if maxIdle == 0 {
		maxIdle = defaultMaxIdleConns
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)
	db.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	return db, nil
}

// buildDSN constructs the MySQL DSN string
func buildDSN(cfg *database.Config) string {
	port := cfg.Port
	if port == 0 {
		port = defaultPort
	}
	// format: user:pass@tcp(host:port)/dbname?parseTime=true
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		cfg.User, cfg.Password, cfg.Host, port, cfg.Database,
	)
}
