package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Item represents one string entry exported from ixion (per sheet/row/field).
type Item struct {
	Sheet  string            `json:"sheet"`
	RowID  string            `json:"rowId"`
	Field  string            `json:"field"`
	Values map[string]string `json:"values"`
}

// Store keeps all items in memory and provides simple lookup helpers.
type Store struct {
	items      []*Item
	bySheetRow map[string]map[string][]*Item // sheet -> rowId -> items
}

// LoadStore loads all JSON files from dataDir into memory.
func LoadStore(dataDir string) (*Store, error) {
	s := &Store{
		bySheetRow: make(map[string]map[string][]*Item),
	}

	var files []string
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk data files: %w", err)
	}
	if len(files) == 0 {
		log.Printf("no data files found in %s", dataDir)
	}

	for _, path := range files {
		if err := s.loadFile(path); err != nil {
			return nil, err
		}
	}

	log.Printf("loaded %d items from %d files", len(s.items), len(files))
	return s, nil
}

func (s *Store) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var arr []*Item
	if err := decoder.Decode(&arr); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	for _, it := range arr {
		if it == nil {
			continue
		}
		s.items = append(s.items, it)

		rowMap, ok := s.bySheetRow[it.Sheet]
		if !ok {
			rowMap = make(map[string][]*Item)
			s.bySheetRow[it.Sheet] = rowMap
		}
		rowMap[it.RowID] = append(rowMap[it.RowID], it)
	}

	return nil
}

// Search finds items whose value in the given language contains the query substring.
// If sheetFilter is non-empty, only items from that sheet are considered.
func (s *Store) Search(lang, query, sheetFilter string, limit int) []*Item {
	if limit <= 0 {
		limit = 100
	}

	lang = strings.ToLower(strings.TrimSpace(lang))
	q := strings.ToLower(query)

	results := make([]*Item, 0, limit)
	for _, it := range s.items {
		if len(results) >= limit {
			break
		}
		if sheetFilter != "" && it.Sheet != sheetFilter {
			continue
		}
		if it.Values == nil {
			continue
		}

		value, ok := it.Values[lang]
		if !ok {
			continue
		}

		if strings.Contains(strings.ToLower(value), q) {
			results = append(results, it)
		}
	}

	return results
}

// GetBySheetRow returns all items for a given sheet and rowId.
// This corresponds to "all items related with pagination" for that row.
func (s *Store) GetBySheetRow(sheet, rowID string) ([]*Item, bool) {
	sheetMap, ok := s.bySheetRow[sheet]
	if !ok {
		return nil, false
	}
	items, ok := sheetMap[rowID]
	return items, ok
}

// Server wraps the in-memory store and exposes HTTP handlers.
type Server struct {
	store *Store
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error": msg,
	})
}

// handleSearch implements:
//  1. Provided language code and an input, search all items that contain input,
//     return sheet name, rowId, field, and values from all languages.
//
// GET /search?lang=en&q=battle[&sheet=AchievementKind][&limit=100]
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
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

	limit := 100
	if limitStr := query.Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			if v > 1000 {
				v = 1000
			}
			limit = v
		}
	}

	results := s.store.Search(lang, q, sheet, limit)
	writeJSON(w, http.StatusOK, results)
}

// handleItems implements:
// 2. Provided sheet and rowId, return all items related with pagination.
//
// GET /items?sheet=AchievementKind&rowId=1
func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()
	sheet := strings.TrimSpace(query.Get("sheet"))
	rowID := strings.TrimSpace(query.Get("rowId"))

	if sheet == "" {
		writeError(w, http.StatusBadRequest, "missing sheet query parameter")
		return
	}
	if rowID == "" {
		writeError(w, http.StatusBadRequest, "missing rowId query parameter")
		return
	}

	items, ok := s.store.GetBySheetRow(sheet, rowID)
	if !ok || len(items) == 0 {
		writeError(w, http.StatusNotFound, "no items found for given sheet and rowId")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func main() {
	addr := flag.String("addr", ":8080", "listen address (e.g. :8080)")
	dataDir := flag.String("data", "data", "directory containing JSON data files")
	flag.Parse()

	store, err := LoadStore(*dataDir)
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}

	server := &Server{store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("/search", server.handleSearch)
	mux.HandleFunc("/items", server.handleItems)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("xivstrings API listening on %s (data dir: %s)", *addr, *dataDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
