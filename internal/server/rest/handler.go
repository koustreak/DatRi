package rest

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/koustreak/DatRi/internal/database"
	"github.com/koustreak/DatRi/internal/errs"
)

// handlers holds the shared state needed by every HTTP handler in this package.
// It is initialised once per Server and passed into the router.
type handlers struct {
	db      database.DB
	schema  *database.Schema
	dialect database.Dialect
}

// ─── GET /tables ──────────────────────────────────────────────────────────────

// handleListTables returns the names of all tables exposed by this resource.
//
//	GET /tables
//	→ 200 { "data": ["users", "orders", ...], "meta": { "count": 2 } }
func (h *handlers) handleListTables(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tables, err := h.db.ListTables(ctx)
	if err != nil {
		httpErrFromDatri(w, err)
		return
	}

	count := len(tables)
	writeData(w, http.StatusOK, tables, &meta{Count: count})
}

// ─── GET /{table} ─────────────────────────────────────────────────────────────

// handleListRows returns all rows from a table, optionally filtered/paginated.
//
//	GET /users
//	GET /users?limit=20&offset=0&order_by=created_at&order_dir=desc&name=alice
//	→ 200 { "data": [...], "meta": { "count": 20, "limit": 20, "offset": 0 } }
func (h *handlers) handleListRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tableName := chi.URLParam(r, "table")

	// ── validate table exists in cached schema ────────────────────────────────
	tableInfo, ok := h.schema.Tables[tableName]
	if !ok {
		writeError(w, http.StatusNotFound, "not_found",
			fmt.Sprintf("table %q does not exist", tableName))
		return
	}

	// ── parse query params ────────────────────────────────────────────────────
	params, err := parseQueryParams(r, tableInfo)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}

	// ── build SELECT ──────────────────────────────────────────────────────────
	builder := database.Select(tableName, h.dialect)
	builder = params.applyToBuilder(builder)

	sql, args, err := builder.Build()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}

	// ── execute ───────────────────────────────────────────────────────────────
	rows, err := h.db.Query(ctx, sql, args...)
	if err != nil {
		httpErrFromDatri(w, err)
		return
	}

	result, err := database.ScanRows(rows)
	if err != nil {
		httpErrFromDatri(w, err)
		return
	}

	count := len(result)
	writeData(w, http.StatusOK, result, &meta{
		Count:  count,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
}

// ─── GET /{table}/{id} ────────────────────────────────────────────────────────

// handleGetRow returns a single row identified by its primary key value.
// Only single-column primary keys are supported in this version.
//
//	GET /users/42
//	→ 200 { "data": { "id": 42, "name": "Alice", ... } }
func (h *handlers) handleGetRow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tableName := chi.URLParam(r, "table")
	id := chi.URLParam(r, "id")

	// ── validate table ────────────────────────────────────────────────────────
	tableInfo, ok := h.schema.Tables[tableName]
	if !ok {
		writeError(w, http.StatusNotFound, "not_found",
			fmt.Sprintf("table %q does not exist", tableName))
		return
	}

	// ── resolve primary key column ────────────────────────────────────────────
	if len(tableInfo.PrimaryKey) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no_primary_key",
			fmt.Sprintf("table %q has no primary key — use GET /%s with filters", tableName, tableName))
		return
	}
	if len(tableInfo.PrimaryKey) > 1 {
		writeError(w, http.StatusUnprocessableEntity, "composite_primary_key",
			fmt.Sprintf("table %q has a composite primary key — use GET /%s with filters", tableName, tableName))
		return
	}
	pkCol := tableInfo.PrimaryKey[0]

	// ── build SELECT … WHERE pk = $1 LIMIT 1 ─────────────────────────────────
	n := 1
	sql, args, err := database.Select(tableName, h.dialect).
		Where(pkCol, "=", id).
		Limit(n).
		Build()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// ── execute ───────────────────────────────────────────────────────────────
	dbRows, err := h.db.Query(ctx, sql, args...)
	if err != nil {
		httpErrFromDatri(w, err)
		return
	}

	rows, err := database.ScanRows(dbRows)
	if err != nil {
		httpErrFromDatri(w, err)
		return
	}

	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "not_found",
			fmt.Sprintf("no row in %q with %s = %q", tableName, pkCol, id))
		return
	}

	writeData(w, http.StatusOK, rows[0], nil)
}

// ─── error mapping ────────────────────────────────────────────────────────────

// httpErrFromDatri maps an *errs.Error to the appropriate HTTP status code
// and writes a structured JSON error response.
func httpErrFromDatri(w http.ResponseWriter, err error) {
	switch {
	case errs.IsNotFound(err):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errs.IsInvalidInput(err):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errs.IsTimeout(err):
		writeError(w, http.StatusGatewayTimeout, "timeout", err.Error())
	case errs.IsPermissionDenied(err):
		writeError(w, http.StatusForbidden, "permission_denied", err.Error())
	case errs.IsConnectionFailed(err):
		writeError(w, http.StatusServiceUnavailable, "connection_failed", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}
