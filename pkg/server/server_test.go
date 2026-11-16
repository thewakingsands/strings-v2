package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"xivstrings/pkg/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	dataDir := t.TempDir()
	rows := `[{"sheet": "Item", "rowId": "1", "index": 0, "values": {"en": "onion", "chs": "洋葱"}}]`
	if err := os.WriteFile(filepath.Join(dataDir, "strings-0000.json"), []byte(rows), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.LoadStore(dataDir, filepath.Join(t.TempDir(), "index"))
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	return &Server{store: st, searchTimeout: searchTimeout}
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body.Error
}

func TestHandleSearchAnswers(t *testing.T) {
	s := newTestServer(t)

	rec := httptest.NewRecorder()
	s.handleSearch(rec, httptest.NewRequest(http.MethodGet, "/api/search?lang=en&q=onion", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"rowId": "1"`) {
		t.Errorf("the row is missing from %s", rec.Body.String())
	}
}

func TestHandleSearchPastTheDeadlineIsAGatewayTimeout(t *testing.T) {
	for _, path := range []string{
		"/api/search?lang=en&q=onion",
		"/api/search?lang=en&q=onion&mode=advanced",
		"/api/items?sheet=Item",
	} {
		t.Run(path, func(t *testing.T) {
			s := newTestServer(t)
			s.searchTimeout = time.Nanosecond

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if strings.HasPrefix(path, "/api/items") {
				s.handleItems(rec, req)
			} else {
				s.handleSearch(rec, req)
			}

			if rec.Code != http.StatusGatewayTimeout {
				t.Fatalf("status = %d, want 504: %s", rec.Code, rec.Body.String())
			}
			if got := decodeError(t, rec); !strings.Contains(got, "longer than 1ns") {
				t.Errorf("error = %q, want it to name the limit", got)
			}
		})
	}
}

func TestHandleSearchWritesNothingOnceTheClientHasGone(t *testing.T) {
	s := newTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/search?lang=en&q=onion", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	s.handleSearch(rec, req)

	if rec.Body.Len() != 0 {
		t.Errorf("wrote %q to a client that has gone", rec.Body.String())
	}
}

func TestHandleSearchRejectsAQueryOfTooManyTermsAsUnprocessable(t *testing.T) {
	s := newTestServer(t)

	words := make([]string, 10000)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	path := "/api/search?lang=en&q=" + url.QueryEscape(strings.Join(words, " "))

	rec := httptest.NewRecorder()
	s.handleSearch(rec, httptest.NewRequest(http.MethodGet, path, nil))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422: %s", rec.Code, rec.Body.String())
	}
	if got := decodeError(t, rec); !strings.Contains(got, "more than") {
		t.Errorf("error = %q, want it to name the limit", got)
	}
}

func TestHandleSearchRejectsAnInvalidAdvancedQuery(t *testing.T) {
	s := newTestServer(t)

	rec := httptest.NewRecorder()
	s.handleSearch(rec, httptest.NewRequest(http.MethodGet, `/api/search?lang=en&mode=advanced&q=%22unclosed`, nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}
