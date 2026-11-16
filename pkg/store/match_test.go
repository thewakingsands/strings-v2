package store

import (
	"slices"
	"testing"

	"github.com/blevesearch/bleve/v2"
)

func repeatsTestStore(t *testing.T) *Store {
	t.Helper()
	return newScorchTestStore(t, []*Item{
		{Sheet: "Item", RowID: "1", Index: 0, Values: map[string]string{"en": "onion", "chs": "洋葱"}},
		{Sheet: "Item", RowID: "2", Index: 1, Values: map[string]string{"en": "soup", "chs": "汤"}},
		{Sheet: "Item", RowID: "3", Index: 2, Values: map[string]string{"en": "onion soup", "chs": "洋葱汤"}},
		{Sheet: "Item", RowID: "4", Index: 3, Values: map[string]string{"en": "sardine", "chs": "沙丁鱼"}},
	})
}

// Searching a repeated term once changes how often its postings are read, not which rows
// match.
func TestSearchWithRepeatedTermsMatchesTheSameRows(t *testing.T) {
	st := repeatsTestStore(t)

	for _, tc := range []struct{ q, lang string }{
		{"onion soup onion onion", "en"},
		{"洋葱汤洋葱", "chs"},
	} {
		plain := bleve.NewMatchQuery(tc.q)
		plain.SetField(tc.lang)
		wantTotal, want := runQuery(t, st, plain)

		gotTotal, got := runQuery(t, st, parseSearchQuery(tc.q, []string{tc.lang}, SheetFilter{}))
		if gotTotal != wantTotal || !slices.Equal(got, want) {
			t.Errorf("%q in %s matched %v (total %d), want %v (total %d)",
				tc.q, tc.lang, got, gotTotal, want, wantTotal)
		}
	}
}

// A term typed twice still counts twice: onion outweighs soup in "onion onion soup".
func TestSearchWeighsARepeatedTermByItsRepeats(t *testing.T) {
	st := repeatsTestStore(t)

	q := parseSearchQuery("onion onion soup", []string{"en"}, SheetFilter{})
	onion, soup := scoreOf(t, st, q, "Item@1"), scoreOf(t, st, q, "Item@2")
	if soup <= 0 || onion <= soup {
		t.Errorf("onion scored %v and soup %v, want both matched and onion ahead", onion, soup)
	}
}

func TestCountTerms(t *testing.T) {
	m := codeMapping(t)
	tokens := m.AnalyzerNamed("cjk").Analyze([]byte("洋葱汤洋葱"))

	terms, repeats := countTerms(tokens)
	if want := []string{"洋葱", "葱汤", "汤洋"}; !slices.Equal(terms, want) {
		t.Errorf("terms = %q, want %q", terms, want)
	}
	if want := []int{2, 1, 1}; !slices.Equal(repeats, want) {
		t.Errorf("repeats = %v, want %v", repeats, want)
	}
}
