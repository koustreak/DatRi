package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/koustreak/DatRi/internal/config"
	"github.com/rs/zerolog"
)

// buildRouter constructs and returns the chi router with all routes and
// middleware attached. Called once per Server during Start().
func buildRouter(h *handlers, cfg config.ResourceConfig, log zerolog.Logger) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware stack (applied to every route) ─────────────────────

	// chi's built-in recoverer catches panics and returns a 500.
	r.Use(chiMiddleware.Recoverer)

	// Structured request logging using DatRi's zerolog logger.
	r.Use(loggingMiddleware(log))

	// CORS — uses the resource's resolved settings.
	cors := cfg.ResolvedCORS(config.GlobalConfig{})
	r.Use(corsMiddleware(cors.Origins, cors.Methods, cors.Headers, cors.Enabled))

	// chi's built-in request-ID stamping — helps trace logs.
	r.Use(chiMiddleware.RequestID)

	// ── Routes ───────────────────────────────────────────────────────────────

	// List all available tables for this resource.
	r.Get("/tables", h.handleListTables)

	// Table-level routes — parameterised by table name.
	r.Route("/{table}", func(r chi.Router) {
		// GET /users               → list rows (with optional filters/pagination)
		r.Get("/", h.handleListRows)

		// GET /users/42            → single row by primary key
		r.Get("/{id}", h.handleGetRow)
	})

	return r
}
