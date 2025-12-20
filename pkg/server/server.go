package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xivstrings/pkg/store"
)

// Server wraps the in-memory store and exposes HTTP handlers.
type Server struct {
	store *store.Store
}

// handleSearch implements:
//  1. Provided language code and an input, search all items that contain input,
//     return sheet name, rowId, and values from all languages.
//
// GET /search?lang=en&q=battle[&sheet=AchievementKind][&offset=0][&limit=100]
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()
	lang := strings.TrimSpace(query.Get("lang"))
	q := strings.TrimSpace(query.Get("q"))
	sheet := strings.TrimSpace(query.Get("sheet"))

	if lang == "" {
		writeError(w, http.StatusBadRequest, "missing lang query parameter")
		return
	}
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing q query parameter")
		return
	}

	offset, limit := parseOffsetLimit(query)
	results := s.store.Search(lang, q, sheet, offset, limit)
	writeJSONWithMeta(w, http.StatusOK, results, time.Since(start))
}

// handleItems implements:
// 2. Provided sheet, return all items related with pagination.
//
// GET /items?sheet=AchievementKind[&offset=0][&limit=100]
func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()
	sheet := strings.TrimSpace(query.Get("sheet"))

	if sheet == "" {
		writeError(w, http.StatusBadRequest, "missing sheet query parameter")
		return
	}

	offset, limit := parseOffsetLimit(query)
	items := s.store.GetBySheet(sheet, offset, limit)
	if len(items) == 0 {
		writeError(w, http.StatusNotFound, "no items found for given sheet")
		return
	}

	writeJSONWithMeta(w, http.StatusOK, items, time.Since(start))
}

type ServerConfig struct {
	Store *store.Store
	UiDir string
}

func CreateMux(config ServerConfig) *http.ServeMux {
	server := &Server{store: config.Store}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/search", server.handleSearch)
	mux.HandleFunc("/api/items", server.handleItems)

	// Serve static files from UI directory
	// For SPA routing: serve index.html for non-API routes that don't match files
	fs := http.FileServer(http.Dir(config.UiDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip API routes (they're already handled above)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Check if the requested file exists
		path := filepath.Join(config.UiDir, r.URL.Path)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// File exists, serve it
			fs.ServeHTTP(w, r)
			return
		}

		// File doesn't exist or is a directory, serve index.html for SPA routing
		indexPath := filepath.Join(config.UiDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
		} else {
			http.NotFound(w, r)
		}
	}))

	return mux
}
