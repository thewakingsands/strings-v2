package server

import (
	"encoding/json"
	"net/http"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeJSONWithMeta(w http.ResponseWriter, status int, data any, elapsed time.Duration) {
	response := map[string]any{
		"data": data,
		"meta": map[string]any{
			"elapsed": elapsed.String(),
		},
	}
	writeJSON(w, status, response)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error": msg,
	})
}
