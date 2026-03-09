package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/koustreak/DatRi/internal/config"
	"github.com/koustreak/DatRi/internal/database"
	"github.com/rs/zerolog"
)

// Server is a single REST API server bound to one port, backed by one
// database.DB. Create one Server per resource that includes "rest" in its
// protocol list.
type Server struct {
	cfg    config.ResourceConfig
	db     database.DB
	schema *database.Schema
	log    zerolog.Logger
}

// New creates a Server. schema must already be fetched and cached by the
// caller (InspectSchema is intentionally expensive and called once at startup).
func New(cfg config.ResourceConfig, db database.DB, schema *database.Schema, log zerolog.Logger) *Server {
	return &Server{
		cfg:    cfg,
		db:     db,
		schema: schema,
		log:    log.With().Str("resource", cfg.Name).Logger(),
	}
}

// Start builds the router, binds the HTTP listener, and serves until ctx is
// cancelled or a fatal listener error occurs. It performs a graceful shutdown
// (waiting up to 10 s for in-flight requests to finish) before returning.
//
// Start blocks — run it in a goroutine when serving multiple resources.
func (s *Server) Start(ctx context.Context) error {
	dialect := dialectFromDriver(s.cfg.Database)

	h := &handlers{
		db:      s.db,
		schema:  s.schema,
		dialect: dialect,
	}

	router := buildRouter(h, s.cfg, s.log)

	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start listening in a goroutine so we can also wait on ctx.
	listenErr := make(chan error, 1)
	go func() {
		s.log.Info().
			Str("addr", addr).
			Strs("protocols", s.cfg.Server.Protocols).
			Msg("REST server starting")

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenErr <- err
		}
		close(listenErr)
	}()

	// Block until context is cancelled (shutdown signal) or listener dies.
	select {
	case err := <-listenErr:
		return err // fatal listener error

	case <-ctx.Done():
		s.log.Info().Str("addr", addr).Msg("REST server shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		s.log.Info().Str("addr", addr).Msg("REST server stopped cleanly")
		return nil
	}
}

// dialectFromDriver maps the config driver string to a database.Dialect.
func dialectFromDriver(dbCfg *config.DatabaseConfig) database.Dialect {
	if dbCfg != nil && dbCfg.Driver == "mysql" {
		return database.DialectMySQL
	}
	return database.DialectPostgres
}
