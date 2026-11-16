package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/searcher"
)

// distinctWords returns n different en words, all matching w*.
func distinctWords(n int) string {
	words := make([]string, n)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	return strings.Join(words, " ")
}

// newScorchTestStore indexes items the way the server does. An in-memory index will
// not do here: it lacks scorch's regexp automaton, so bleve counts a wildcard's terms
// while listing them, even in a filter, and fails where the server would not.
func newScorchTestStore(t *testing.T, items []*Item) *Store {
	t.Helper()

	idx, err := bleve.New(filepath.Join(t.TempDir(), "index"), buildItemIndexMapping())
	if err != nil {
		t.Fatalf("create index: %v", err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Errorf("close index: %v", err)
		}
	})

	if err := indexItems(idx, items); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

func newWordsTestStore(t *testing.T, words int) *Store {
	t.Helper()
	return newScorchTestStore(t, []*Item{{Sheet: "Item", RowID: "1", Values: map[string]string{"en": distinctWords(words)}}})
}

func TestAdvancedSearchExpandsAClauseUpToTheLimit(t *testing.T) {
	st := newWordsTestStore(t, maxClauseCount)

	result, err := st.AdvancedSearch(t.Context(), "en:w*", []string{"en"}, SheetFilter{}, 0, 10, []string{"en"})
	if err != nil {
		t.Fatalf("advanced search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
}

func TestAdvancedSearchRejectsAClauseExpandingPastTheLimit(t *testing.T) {
	st := newWordsTestStore(t, maxClauseCount+1)

	_, err := st.AdvancedSearch(t.Context(), "en:w*", []string{"en"}, SheetFilter{}, 0, 10, []string{"en"})
	var tooBroad *QueryTooBroadError
	if !errors.As(err, &tooBroad) {
		t.Fatalf("err = %v, want a *QueryTooBroadError", err)
	}
	if !strings.Contains(err.Error(), "terms in en") {
		t.Errorf("error = %q, want it to name the field", err)
	}
}

func TestSearchRejectsAQueryOfMoreWordsThanTheLimit(t *testing.T) {
	st := newWordsTestStore(t, 1)

	_, err := st.Search(t.Context(), distinctWords(maxClauseCount+1), []string{"en"}, SheetFilter{}, 0, 10, []string{"en"})
	var tooBroad *QueryTooBroadError
	if !errors.As(err, &tooBroad) {
		t.Fatalf("err = %v, want a *QueryTooBroadError", err)
	}
	if strings.Contains(err.Error(), " in ") {
		t.Errorf("error = %q names a field, but bleve counted the alternatives of a disjunction", err)
	}
}

func TestSheetExclusionIgnoresTheClauseLimit(t *testing.T) {
	limit := searcher.DisjunctionMaxClauseCount
	// Bleve merges a filter's terms in batches the size of the limit, and will not merge
	// a batch of one, so a filter works at 2 and up.
	searcher.DisjunctionMaxClauseCount = 2
	t.Cleanup(func() { searcher.DisjunctionMaxClauseCount = limit })

	st := newScorchTestStore(t, sheetTestItems())
	langs := []string{"en"}

	// The same exclusion as a scored clause shows that the limit is in force.
	if _, err := st.AdvancedSearch(t.Context(), "onion -sheet:*/*", langs, SheetFilter{}, 0, 100, langs); err == nil {
		t.Fatal("fixture: */* matches too few sheets to test the limit")
	}

	all, err := st.Search(t.Context(), "onion", langs, SheetFilter{}, 0, 100, langs)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, filter := range []SheetFilter{
		{Exclude: []string{"*/*"}},
		{Include: []string{"*"}, Exclude: []string{"*/*"}},
	} {
		filtered, err := st.Search(t.Context(), "onion", langs, filter, 0, 100, langs)
		if err != nil {
			t.Fatalf("search with %+v: %v", filter, err)
		}
		assertFilteredOrder(t, filter, all, filtered)
	}
}
