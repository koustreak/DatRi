// Command datri is the DatRi CLI entrypoint.
// It loads datri.yaml, opens a DB connection for every resource, inspects
// their schemas, and starts one REST server goroutine per resource.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/koustreak/DatRi/internal/config"
	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/database/postgres"
	"github.com/koustreak/DatRi/internal/server/rest"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "datri:", err)
		os.Exit(1)
	}
}

func run() error {
	// ── 1. Load config ────────────────────────────────────────────────────────
	cfgPath := configPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config %q: %w", cfgPath, err)
	}

	// ── 2. Set up logger ──────────────────────────────────────────────────────
	logger := buildLogger(cfg.Global.Log)
	log.Logger = logger // set global zerolog logger

	logger.Info().Str("config", cfgPath).Msg("DatRi starting")

	// ── 3. Root context — cancelled on SIGINT / SIGTERM ───────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── 4. Start one server per resource ─────────────────────────────────────
	var wg sync.WaitGroup
	errs := make(chan error, len(cfg.Resources))

	for _, res := range cfg.Resources {
		res := res // loop capture

		if res.Type != "database" {
			logger.Warn().Str("resource", res.Name).Str("type", res.Type).
				Msg("non-database resources not yet supported — skipping")
			continue
		}

		if !hasProtocol(res.Server.Protocols, "rest") {
			logger.Warn().Str("resource", res.Name).
				Msg("no 'rest' protocol configured — skipping")
			continue
		}

		// Open DB connection.
		db, schema, err := connectAndInspect(ctx, res, logger)
		if err != nil {
			return fmt.Errorf("resource %q: %w", res.Name, err)
		}
		defer db.Close()

		// Start REST server.
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv := rest.New(res, db, schema, logger)
			if err := srv.Start(ctx); err != nil {
				errs <- fmt.Errorf("resource %q REST server: %w", res.Name, err)
			}
		}()
	}

	// Wait for ctx cancellation (OS signal), then wait for all servers to stop.
	<-ctx.Done()
	logger.Info().Msg("shutdown signal received — waiting for servers to stop")
	wg.Wait()

	// Check for fatal server errors.
	close(errs)
	var combined error
	for e := range errs {
		combined = errors.Join(combined, e)
	}
	return combined
}

// connectAndInspect opens a DB connection and introspects the schema once.
func connectAndInspect(ctx context.Context, res config.ResourceConfig, logger zerolog.Logger) (database.DB, *database.Schema, error) {
	dbCfg := res.Database.ToDatabaseConfig()

	var db database.DB
	var err error

	switch dbCfg.Driver {
	case database.DriverPostgres:
		db, err = postgres.New(ctx, dbCfg)
	default:
		return nil, nil, fmt.Errorf("unsupported driver: %q (mysql support coming soon)", dbCfg.Driver)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to database: %w", err)
	}

	logger.Info().Str("resource", res.Name).Str("driver", string(dbCfg.Driver)).Msg("database connected")

	schema, err := db.InspectSchema(ctx)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("inspecting schema: %w", err)
	}

	tableNames := make([]string, 0, len(schema.Tables))
	for name := range schema.Tables {
		tableNames = append(tableNames, name)
	}
	logger.Info().Str("resource", res.Name).Strs("tables", tableNames).Msg("schema loaded")

	return db, schema, nil
}

// buildLogger creates a zerolog.Logger from the global log config.
func buildLogger(cfg config.LogConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.Format == "json" {
		return zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
}

// configPath returns the path to datri.yaml, from --config flag or default.
func configPath() string {
	for i, arg := range os.Args[1:] {
		if (arg == "--config" || arg == "-config") && i+1 < len(os.Args[1:]) {
			return os.Args[i+2]
		}
	}
	return "datri.yaml"
}

// hasProtocol reports whether the given protocol is in the list.
func hasProtocol(protocols []string, target string) bool {
	for _, p := range protocols {
		if p == target {
			return true
		}
	}
	return false
}
