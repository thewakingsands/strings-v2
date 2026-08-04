package store

import (
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// longEnglishValue is longer than the 200 rune fragment size of Bleve's simple
// fragmenter, so a highlight fragment of it is necessarily truncated.
const longEnglishValue = "wizard eggplant, a violet vegetable that thrives in the damp soil of " +
	"the north shroud, prized by culinarians for its firm flesh and by botanists for " +
	"its stubborn refusal to grow anywhere else, is sold by most gardening vendors."

// markupEnglishValue contains every character Bleve's HTML fragment formatter escapes.
const markupEnglishValue = `the wives' tale & "the sea" <lore>`

var searchTestFields = []string{"chs", "en"}

// searchTestItems covers the interesting combinations for a query of "onion":
// a chs only match, an en only match, matches in both, and a match in another sheet.
func searchTestItems() []*Item {
	return []*Item{
		{Sheet: "Item", RowID: "1", Index: 0, Values: map[string]string{"chs": "onion", "en": "sardine"}},
		{Sheet: "Item", RowID: "2", Index: 1, Values: map[string]string{"chs": "沙丁鱼", "en": "onion"}},
		{Sheet: "Item", RowID: "3", Index: 2, Values: map[string]string{"chs": "onion", "en": "onion"}},
		{Sheet: "Other", RowID: "4", Index: 0, Values: map[string]string{"chs": "onion", "en": "onion"}},
		{Sheet: "Item", RowID: "5", Index: 3, Values: map[string]string{"chs": "巫师茄子", "en": longEnglishValue}},
		{Sheet: "Item", RowID: "6", Index: 4, Values: map[string]string{"chs": "妻子的故事", "en": markupEnglishValue}},
	}
}

func newTestStore(t *testing.T) *Store {
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

	if err := indexItems(idx, searchTestItems()); err != nil {
		t.Fatalf("index items: %v", err)
	}

	return &Store{index: idx}
}

// rankOf returns the position of an item in the result list, or -1 when absent.
func rankOf(items []*Item, sheet, rowID string) int {
	for i, item := range items {
		if item.Sheet == sheet && item.RowID == rowID {
			return i
		}
	}
	return -1
}

func rowIDsOf(items []*Item) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Sheet+"#"+item.RowID)
	}
	return ids
}

func TestParseSearchQuerySingleLanguageStaysUnboosted(t *testing.T) {
	q := parseSearchQuery("onion", []string{"en"}, "")

	matchQuery, ok := q.(*query.MatchQuery)
	if !ok {
		t.Fatalf("query type = %T, want *query.MatchQuery", q)
	}
	if matchQuery.Match != "onion" {
		t.Errorf("match = %q, want %q", matchQuery.Match, "onion")
	}
	if matchQuery.FieldVal != "en" {
		t.Errorf("field = %q, want %q", matchQuery.FieldVal, "en")
	}
	if matchQuery.BoostVal != nil {
		t.Errorf("boost = %v, want unset so scores match a single language search", *matchQuery.BoostVal)
	}
}

func TestParseSearchQueryBoostsByLanguageOrder(t *testing.T) {
	langs := []string{"chs", "en", "ja"}
	q := parseSearchQuery("onion", langs, "")

	disjunction, ok := q.(*query.DisjunctionQuery)
	if !ok {
		t.Fatalf("query type = %T, want *query.DisjunctionQuery", q)
	}
	if len(disjunction.Disjuncts) != len(langs) {
		t.Fatalf("got %d disjuncts, want %d", len(disjunction.Disjuncts), len(langs))
	}

	wantBoosts := []float64{3, 2, 1}
	for i, disjunct := range disjunction.Disjuncts {
		matchQuery, ok := disjunct.(*query.MatchQuery)
		if !ok {
			t.Fatalf("disjunct %d type = %T, want *query.MatchQuery", i, disjunct)
		}
		if matchQuery.FieldVal != langs[i] {
			t.Errorf("disjunct %d field = %q, want %q", i, matchQuery.FieldVal, langs[i])
		}
		if matchQuery.BoostVal == nil {
			t.Fatalf("disjunct %d boost is unset, want %v", i, wantBoosts[i])
		}
		if got := float64(*matchQuery.BoostVal); got != wantBoosts[i] {
			t.Errorf("disjunct %d boost = %v, want %v", i, got, wantBoosts[i])
		}
	}
}

func TestParseSearchQueryAddsSheetFilter(t *testing.T) {
	cases := []struct {
		name          string
		langs         []string
		wantTextQuery any
	}{
		{name: "single language", langs: []string{"en"}, wantTextQuery: &query.MatchQuery{}},
		{name: "multiple languages", langs: []string{"chs", "en"}, wantTextQuery: &query.DisjunctionQuery{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := parseSearchQuery("onion", tc.langs, "Item")

			conjunction, ok := q.(*query.ConjunctionQuery)
			if !ok {
				t.Fatalf("query type = %T, want *query.ConjunctionQuery", q)
			}
			if len(conjunction.Conjuncts) != 2 {
				t.Fatalf("got %d conjuncts, want 2", len(conjunction.Conjuncts))
			}
			if got, want := conjunction.Conjuncts[0], tc.wantTextQuery; !sameQueryType(got, want) {
				t.Errorf("text conjunct type = %T, want %T", got, want)
			}

			termQuery, ok := conjunction.Conjuncts[1].(*query.TermQuery)
			if !ok {
				t.Fatalf("sheet conjunct type = %T, want *query.TermQuery", conjunction.Conjuncts[1])
			}
			if termQuery.FieldVal != "sheet" {
				t.Errorf("sheet conjunct field = %q, want %q", termQuery.FieldVal, "sheet")
			}
			if termQuery.Term != "Item" {
				t.Errorf("sheet conjunct term = %q, want %q", termQuery.Term, "Item")
			}
		})
	}
}

func sameQueryType(got, want any) bool {
	switch want.(type) {
	case *query.MatchQuery:
		_, ok := got.(*query.MatchQuery)
		return ok
	case *query.DisjunctionQuery:
		_, ok := got.(*query.DisjunctionQuery)
		return ok
	}
	return false
}

func TestSearchSingleLanguageOnlyMatchesThatLanguage(t *testing.T) {
	st := newTestStore(t)

	result, err := st.Search("onion", []string{"en"}, "", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	// "onion" sits in the en column of Item#2, Item#3 and Other#4 only.
	if result.Total != 3 {
		t.Errorf("total = %d, want 3 (got %v)", result.Total, rowIDsOf(result.Items))
	}
	if rank := rankOf(result.Items, "Item", "1"); rank != -1 {
		t.Errorf("Item#1 matches chs only and must not appear in an en search, got rank %d", rank)
	}
}

func TestSearchAcrossLanguagesRanksByLanguageOrder(t *testing.T) {
	st := newTestStore(t)

	// Item#1 matches in chs only, Item#2 in en only. The two documents are otherwise
	// symmetric, so the boost derived from the lang order is what separates them.
	chsFirst, err := st.Search("onion", []string{"chs", "en"}, "Item", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search chs,en: %v", err)
	}
	chsHit, enHit := rankOf(chsFirst.Items, "Item", "1"), rankOf(chsFirst.Items, "Item", "2")
	if chsHit == -1 || enHit == -1 {
		t.Fatalf("lang=chs,en returned %v, want both Item#1 and Item#2", rowIDsOf(chsFirst.Items))
	}
	if chsHit > enHit {
		t.Errorf("lang=chs,en ranked the en match above the chs match: %v", rowIDsOf(chsFirst.Items))
	}

	enFirst, err := st.Search("onion", []string{"en", "chs"}, "Item", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search en,chs: %v", err)
	}
	chsHit, enHit = rankOf(enFirst.Items, "Item", "1"), rankOf(enFirst.Items, "Item", "2")
	if chsHit == -1 || enHit == -1 {
		t.Fatalf("lang=en,chs returned %v, want both Item#1 and Item#2", rowIDsOf(enFirst.Items))
	}
	if enHit > chsHit {
		t.Errorf("lang=en,chs ranked the chs match above the en match: %v", rowIDsOf(enFirst.Items))
	}
}

func TestSearchAcrossLanguagesDeduplicatesTotal(t *testing.T) {
	st := newTestStore(t)

	result, err := st.Search("onion", []string{"chs", "en"}, "", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	// chs matches 3 documents and en matches 3, but they overlap on Item#3 and Other#4,
	// so the union is 4. A client fanning out per language could only report 6.
	if result.Total != 4 {
		t.Errorf("total = %d, want 4 unique documents (got %v)", result.Total, rowIDsOf(result.Items))
	}
	if len(result.Items) != 4 {
		t.Errorf("got %d items, want 4 (%v)", len(result.Items), rowIDsOf(result.Items))
	}
}

func TestSearchAcrossLanguagesRespectsSheetFilter(t *testing.T) {
	st := newTestStore(t)

	result, err := st.Search("onion", []string{"chs", "en"}, "Item", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("total = %d, want 3 (got %v)", result.Total, rowIDsOf(result.Items))
	}
	if rank := rankOf(result.Items, "Other", "4"); rank != -1 {
		t.Errorf("Other#4 is outside the sheet filter but appeared at rank %d", rank)
	}
}

func TestSearchReturnsCompleteValuesAndTruncatedHighlight(t *testing.T) {
	if utf8.RuneCountInString(longEnglishValue) <= 200 {
		t.Fatalf("fixture is %d runes, it must exceed the 200 rune fragment size to be meaningful",
			utf8.RuneCountInString(longEnglishValue))
	}

	st := newTestStore(t)

	result, err := st.Search("eggplant", []string{"en"}, "Item", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1 (%v)", len(result.Items), rowIDsOf(result.Items))
	}
	item := result.Items[0]

	if item.Values["en"] != longEnglishValue {
		t.Errorf("values[en] = %q, want the complete value %q", item.Values["en"], longEnglishValue)
	}
	if item.Values["chs"] != "巫师茄子" {
		t.Errorf("values[chs] = %q, want %q", item.Values["chs"], "巫师茄子")
	}

	highlight := item.Highlights["en"]
	if !strings.Contains(highlight, "<mark>eggplant</mark>") {
		t.Errorf("highlights[en] = %q, want it to mark the match", highlight)
	}
	if !strings.HasSuffix(highlight, "…") {
		t.Errorf("highlights[en] = %q, want a truncated fragment ending in an ellipsis", highlight)
	}
}

func TestSearchHighlightsAreEscapedWhileValuesAreRaw(t *testing.T) {
	st := newTestStore(t)

	result, err := st.Search("wives", []string{"en"}, "Item", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1 (%v)", len(result.Items), rowIDsOf(result.Items))
	}
	item := result.Items[0]

	if item.Values["en"] != markupEnglishValue {
		t.Errorf("values[en] = %q, want the raw value %q", item.Values["en"], markupEnglishValue)
	}

	want := `the <mark>wives</mark>&#39; tale &amp; &#34;the sea&#34; &lt;lore&gt;`
	if item.Highlights["en"] != want {
		t.Errorf("highlights[en] = %q, want %q", item.Highlights["en"], want)
	}
}

func TestSearchOmitsHighlightForLanguagesThatDidNotMatch(t *testing.T) {
	st := newTestStore(t)

	// Item#1 matches in chs only. Bleve still fragments the en column because it is a
	// highlighted field, but that fragment holds no match and must not be reported.
	result, err := st.Search("onion", []string{"chs", "en"}, "Item", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	rank := rankOf(result.Items, "Item", "1")
	if rank == -1 {
		t.Fatalf("Item#1 missing from %v", rowIDsOf(result.Items))
	}
	item := result.Items[rank]

	if _, ok := item.Highlights["en"]; ok {
		t.Errorf("highlights[en] = %q, want no entry because en did not match", item.Highlights["en"])
	}
	if got := item.Highlights["chs"]; !strings.Contains(got, "<mark>onion</mark>") {
		t.Errorf("highlights[chs] = %q, want it to mark the match", got)
	}
	if item.Values["en"] != "sardine" {
		t.Errorf("values[en] = %q, want the complete value %q", item.Values["en"], "sardine")
	}
}

func TestSearchWithoutLanguageFails(t *testing.T) {
	st := newTestStore(t)

	if _, err := st.Search("onion", nil, "", 0, 100, searchTestFields); err == nil {
		t.Error("search without a language succeeded, want an error")
	}
}

func TestGetBySheetHasNoHighlights(t *testing.T) {
	st := newTestStore(t)

	result, err := st.GetBySheet("Item", 0, 100, searchTestFields)
	if err != nil {
		t.Fatalf("get by sheet: %v", err)
	}
	if len(result.Items) == 0 {
		t.Fatal("got no items")
	}

	for _, item := range result.Items {
		if item.Highlights != nil {
			t.Errorf("%s#%s has highlights %v, want none for a plain sheet lookup",
				item.Sheet, item.RowID, item.Highlights)
		}
	}
	if got := result.Items[0].Values["en"]; got != "sardine" {
		t.Errorf("values[en] = %q, want %q", got, "sardine")
	}
}
