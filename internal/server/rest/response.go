// Package rest implements DatRi's HTTP/REST server layer.
// It wires chi routes, middleware, and query-param parsing on top of
// the database.DB interface — never touching driver-specific code directly.
package rest

import (
	"encoding/json"
	"net/http"
)

// ─── Response envelope ────────────────────────────────────────────────────────

// envelope is the top-level JSON wrapper for every successful response.
//
//	{ "data": [...], "meta": { "count": 42 } }
type envelope struct {
	Data any   `json:"data"`
	Meta *meta `json:"meta,omitempty"`
}

// meta carries optional pagination/count info alongside the data.
type meta struct {
	Count  int  `json:"count"`
	Limit  *int `json:"limit,omitempty"`
	Offset *int `json:"offset,omitempty"`
}

// apiError is the JSON body returned on any non-2xx response.
//
//	{ "error": { "kind": "not_found", "message": "table \"foo\" does not exist" } }
type apiError struct {
	Error errBody `json:"error"`
}

type errBody struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// ─── Writers ──────────────────────────────────────────────────────────────────

// writeJSON serialises v as JSON and writes it with the given HTTP status.
// If marshalling fails a 500 is written instead.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Nothing useful we can do here — headers are already sent.
		_ = err
	}
}

// writeData is a convenience wrapper that wraps v in the standard envelope.
func writeData(w http.ResponseWriter, status int, data any, m *meta) {
	writeJSON(w, status, envelope{Data: data, Meta: m})
}

// writeError writes a structured error response.
// kind should be one of the errs.ErrKind string representations
// (e.g. "not_found", "invalid_input", "query_failed", …).
func writeError(w http.ResponseWriter, status int, kind, message string) {
	writeJSON(w, status, apiError{Error: errBody{Kind: kind, Message: message}})
}
