package store

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

func advancedTestItems() []*Item {
	return []*Item{
		{Sheet: "Item", RowID: "7", Index: 5, Values: map[string]string{"chs": "暗鲈", "en": "dark bass"}},
		{Sheet: "Item", RowID: "8", Index: 6, Values: map[string]string{"chs": "黄金鲈鱼", "en": "golden perch"}},
	}
}

func newAdvancedTestStore(t *testing.T) *Store {
	t.Helper()

	idx, err := bleve.NewMemOnly(buildItemIndexMapping())
	if err != nil {
		t.Fatalf("create in-memory index: %v", err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Errorf("close index: %v", err)
		}
	})

	if err := indexItems(idx, slices.Concat(searchTestItems(), advancedTestItems())); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

func advancedSearch(t *testing.T, st *Store, q string, langs []string) *SearchResult {
	t.Helper()

	result, err := st.AdvancedSearch(q, langs, "", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("advanced search %q: %v", q, err)
	}
	return result
}

func sortedRowIDsOf(items []*Item) []string {
	ids := rowIDsOf(items)
	slices.Sort(ids)
	return ids
}

// The reason this exists: a single CJK character is never a term of its own in a
// bigram index, so only a wildcard over the bigrams can find it.
func TestAdvancedSearchWildcardFindsSingleCJKCharacter(t *testing.T) {
	st := newAdvancedTestStore(t)

	plain, err := st.Search("鲈", []string{"chs"}, "", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if plain.Total != 0 {
		t.Fatalf("plain search for 鲈 found %v; the bigram index is expected to miss it", rowIDsOf(plain.Items))
	}

	result := advancedSearch(t, st, "*鲈*", []string{"chs", "en"})

	if got, want := sortedRowIDsOf(result.Items), []string{"Item#7", "Item#8"}; !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if highlight := itemAt(t, result, "Item", "7").Highlights["chs"]; !strings.Contains(highlight, "<mark>暗鲈</mark>") {
		t.Errorf("chs highlight = %q, want the matched bigram marked", highlight)
	}
}

func TestAdvancedSearchUnfieldedTermMatchesLikePlainSearch(t *testing.T) {
	st := newAdvancedTestStore(t)
	langs := []string{"chs", "en"}

	plain, err := st.Search("onion", langs, "", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	advanced := advancedSearch(t, st, "onion", langs)

	if got, want := rowIDsOf(advanced.Items), rowIDsOf(plain.Items); !slices.Equal(got, want) {
		t.Errorf("advanced rows = %v, plain rows = %v; a bare word should behave the same", got, want)
	}
}

// -word drops a row that has the word in any searched language, not only in the one
// the rest of the query happened to match in.
func TestAdvancedSearchExclusionAppliesToEveryLanguage(t *testing.T) {
	st := newAdvancedTestStore(t)

	result := advancedSearch(t, st, "onion -sardine", []string{"chs", "en"})

	if rank := rankOf(result.Items, "Item", "1"); rank != -1 {
		t.Errorf("Item#1 has sardine in en and must be excluded, got rank %d", rank)
	}
	if got, want := sortedRowIDsOf(result.Items), []string{"Item#2", "Item#3", "Other#4"}; !slices.Equal(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

// +a +b needs each word in some language, not both in the same one.
func TestAdvancedSearchRequiredTermsMayMatchDifferentLanguages(t *testing.T) {
	st := newAdvancedTestStore(t)

	result := advancedSearch(t, st, "+暗鲈 +bass", []string{"chs", "en"})

	if got, want := rowIDsOf(result.Items), []string{"Item#7"}; !slices.Equal(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestAdvancedSearchFieldedClauseSearchesOnlyThatField(t *testing.T) {
	st := newAdvancedTestStore(t)

	// en is not among the languages, but the clause names it.
	result := advancedSearch(t, st, "en:onion", []string{"chs"})

	if got, want := sortedRowIDsOf(result.Items), []string{"Item#2", "Item#3", "Other#4"}; !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if highlight := itemAt(t, result, "Item", "2").Highlights["en"]; highlight != "<mark>onion</mark>" {
		t.Errorf("en highlight = %q, want the match marked even though en was not searched by default", highlight)
	}
}

func TestAdvancedSearchFiltersBySheetField(t *testing.T) {
	st := newAdvancedTestStore(t)

	result := advancedSearch(t, st, "+onion +sheet:Other", []string{"chs", "en"})

	if got, want := rowIDsOf(result.Items), []string{"Other#4"}; !slices.Equal(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestAdvancedSearchRespectsSheetParameter(t *testing.T) {
	st := newAdvancedTestStore(t)

	result, err := st.AdvancedSearch("onion", []string{"chs", "en"}, "Other", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("advanced search: %v", err)
	}

	if got, want := rowIDsOf(result.Items), []string{"Other#4"}; !slices.Equal(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestAdvancedSearchReportsUnparsableQuery(t *testing.T) {
	st := newAdvancedTestStore(t)

	_, err := st.AdvancedSearch(`"unclosed`, []string{"chs"}, "", 0, 100, searchTestFields)

	var invalid *InvalidQueryError
	if !errors.As(err, &invalid) {
		t.Fatalf("err = %v, want an *InvalidQueryError", err)
	}
}

func TestParseAdvancedQueryWeightsLanguagesLikePlainSearch(t *testing.T) {
	langs := []string{"chs", "en"}

	parsed, err := parseAdvancedQuery("鲈^2", langs, "")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	dismax := findDisMax(t, parsed)
	if len(dismax.disjuncts) != len(langs) {
		t.Fatalf("dis_max has %d clauses, want one per language", len(dismax.disjuncts))
	}
	for i, disjunct := range dismax.disjuncts {
		// The query's own ^2 times the weight parseSearchQuery would give this language,
		// which for a CJK only query includes damping the Latin language.
		want := 2 * languageBoost(i, langs[i], true)
		if got := disjunct.(query.BoostableQuery).Boost(); got != want {
			t.Errorf("%s boost = %v, want %v", langs[i], got, want)
		}
	}
}

// findDisMax returns the only dis_max inside the boolean query a query string parses to.
func findDisMax(t *testing.T, q query.Query) *disMaxQuery {
	t.Helper()

	var found []*disMaxQuery
	var walk func(query.Query)
	walk = func(q query.Query) {
		switch q := q.(type) {
		case *disMaxQuery:
			found = append(found, q)
		case *query.BooleanQuery:
			for _, child := range []query.Query{q.Must, q.Should, q.MustNot, q.Filter} {
				walk(child)
			}
		case *query.ConjunctionQuery:
			for _, child := range q.Conjuncts {
				walk(child)
			}
		case *query.DisjunctionQuery:
			for _, child := range q.Disjuncts {
				walk(child)
			}
		}
	}
	walk(q)

	if len(found) != 1 {
		t.Fatalf("found %d dis_max queries in %T, want 1", len(found), q)
	}
	return found[0]
}
