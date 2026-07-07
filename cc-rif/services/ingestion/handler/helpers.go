// Package handler provides HTTP handler constructors for the Ingestion Service
// REST API. Every handler accepts and propagates context.Context from the
// incoming request. All error responses are structured JSON objects.
package handler

import (
	"encoding/json"
	"net/http"
)

const maxJSONBodyBytes = 1 << 20 // 1 MiB

// writeJSON sets Content-Type, writes the HTTP status code, and encodes v as
// JSON. If encoding fails after the header is written the error is silently
// dropped — the response header is already committed at that point.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// errResponse returns a map suitable for use as a JSON error body.
func errResponse(msg string) map[string]string {
	return map[string]string{"error": msg}
}

// decodeBody decodes the JSON request body into dst. Returns false and writes
// a 400 response if decoding fails.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse("invalid JSON body: "+err.Error()))
		return false
	}
	return true
}
