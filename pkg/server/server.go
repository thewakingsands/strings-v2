package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"xivstrings/pkg/store"
	"xivstrings/pkg/version"
)

// Server wraps the in-memory store and exposes HTTP handlers.
// Store can be replaced when data is updated via POST /api/version with token.
type Server struct {
	mu          sync.RWMutex
	store       *store.Store
	baseDir     string
	updateToken string // from XIVSTRINGS_UPDATE_TOKEN; empty means update not allowed
}

// handleSearch implements:
//  1. Provided language code and an input, search all items that contain input,
//     return sheet name, rowId, and values from all languages.
//
// GET /search?lang=en&q=battle[&sheet=AchievementKind][&offset=0][&limit=100]
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()
	lang := strings.TrimSpace(query.Get("lang"))
	q := strings.TrimSpace(query.Get("q"))

	if lang == "" {
		writeError(w, http.StatusBadRequest, "missing lang query parameter")
		return
	}
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing q query parameter")
		return
	}

	offset, limit := parseOffsetLimit(query)
	fields, err := parseFields(query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.mu.RLock()
	st := s.store
	s.mu.RUnlock()
	if st == nil {
		writeError(w, http.StatusServiceUnavailable, "store not loaded")
		return
	}
	results, err := st.Search(lang, q, offset, limit, fields)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]any{
		"data": results.Items,
		"meta": map[string]any{
			"elapsed": results.Elapsed.String(),
			"total":   results.Total,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// handleItems implements:
// 2. Provided sheet, return all items related with pagination.
//
// GET /items?sheet=AchievementKind[&offset=0][&limit=100]
func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
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
	fields, err := parseFields(query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.mu.RLock()
	st := s.store
	s.mu.RUnlock()
	if st == nil {
		writeError(w, http.StatusServiceUnavailable, "store not loaded")
		return
	}
	results, err := st.GetBySheet(sheet, offset, limit, fields)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]any{
		"data": results.Items,
		"meta": map[string]any{
			"elapsed": results.Elapsed.String(),
			"total":   results.Total,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// handleVersion: GET returns current data version; POST triggers update (requires token).
// GET /api/version -> { "version": "publish-20260303-8b409c8" }
// POST /api/version?token=... -> { "version": "...", "updated": true|false }
// Token is set via environment variable XIVSTRINGS_UPDATE_TOKEN. If not set, POST returns 403.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		v, err := version.GetLocalVersion(s.baseDir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"version": v})
		return
	case http.MethodPost:
		if s.updateToken == "" {
			writeError(w, http.StatusForbidden, "update not allowed: XIVSTRINGS_UPDATE_TOKEN is not set")
			return
		}
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			writeError(w, http.StatusBadRequest, "missing token parameter")
			return
		}
		if token != s.updateToken {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		result, err := version.EnsureVersion(s.baseDir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if result.Updated {
			newStore, err := store.LoadStore(result.StringDir, result.IndexDir)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "reload store after update: "+err.Error())
				return
			}
			s.SetStore(newStore)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"version": result.Version,
			"updated": result.Updated,
		})
		return
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type ServerConfig struct {
	Store       *store.Store
	UiDir       string
	BaseDir     string // root dir for data/, index/, version file
	UpdateToken string // from XIVSTRINGS_UPDATE_TOKEN; required for POST /api/version
}

// SetStore replaces the current store (closes the old one). Caller must not use the old store after this.
func (s *Server) SetStore(newStore *store.Store) {
	s.mu.Lock()
	old := s.store
	s.store = newStore
	s.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
}

func CreateMux(config ServerConfig) *http.ServeMux {
	server := &Server{
		store:       config.Store,
		baseDir:     config.BaseDir,
		updateToken: config.UpdateToken,
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/search", server.handleSearch)
	mux.HandleFunc("/api/items", server.handleItems)
	mux.HandleFunc("/api/version", server.handleVersion)

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

// filterItemsByFields filters the values map of each item to only include the specified field languages.
func filterItemsByFields(items []*store.Item, fields []string) []*store.Item {
	// Create a set of valid fields for quick lookup
	fieldSet := make(map[string]bool)
	for _, field := range fields {
		fieldSet[strings.TrimSpace(field)] = true
	}

	filtered := make([]*store.Item, len(items))
	for i, item := range items {
		filteredItem := &store.Item{
			Sheet:  item.Sheet,
			RowID:  item.RowID,
			Index:  item.Index,
			Values: make(map[string]string),
		}
		// Only include the specified field languages in values
		for field := range fieldSet {
			if val, ok := item.Values[field]; ok {
				filteredItem.Values[field] = val
			}
		}
		filtered[i] = filteredItem
	}
	return filtered
}
