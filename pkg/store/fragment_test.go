package store

import (
	"slices"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// The reported shape: a licence text repeated across every column, long enough that the
// complete value dwarfs a fragment. The stored text uses curly quotes while a user types
// straight ones, which is why the display layer cannot match it by comparing characters
// and the matching has to run through an analyzer.
const curlyQuoteValue = "------------------------\nlua 5.1.4 / MIT License\n" +
	"------------------------\nCopyright (C) 1994-2008 Lua.org, PUC-Rio.\n\n" +
	"Permission is hereby granted, free of charge, to any person obtaining a copy " +
	"of this software and associated documentation files (the “Software”), to deal " +
	"in the Software without restriction, including without limitation the rights " +
	"to use, copy, modify, merge, publish, distribute, sublicense, and/or sell " +
	"copies of the Software, and to permit persons to whom the Software is " +
	"furnished to do so, subject to the following conditions:\n\n" +
	"The above copyright notice and this permission notice shall be included in " +
	"all copies or substantial portions of the Software."

const straightQuoteQuery = `including without limitation the rights to use, copy, ` +
	`modify, merge, publish, distribute, sublicense, and/or sell copies of the "Software"`

func fragmentFixture() []*Item {
	return []*Item{
		// Untranslated: every column holds the same licence text.
		{Sheet: "AddonTransient", RowID: "173", Index: 0, Values: map[string]string{
			"chs": curlyQuoteValue,
			"tc":  curlyQuoteValue,
			"en":  curlyQuoteValue,
			"ja":  curlyQuoteValue,
		}},
		// Properly translated: searching Chinese cannot match the en column at all.
		{Sheet: "Item", RowID: "8166", Index: 0, Values: map[string]string{
			"chs": "萨维奈圆葱\n\n圆形的野菜，原产于近东地区，可以用来制作陆行鸟的饲料。",
			"tc":  "薩維奈圓蔥\n\n圓形的野菜，原產於近東地區，可以用來製作陸行鳥的飼料。",
			"en":  "Thavnairian onion\nThavnairian onions\nA pungent, tear-inducing vegetable.",
			"ja":  "サベネアの野菜\n\n近東原産のまん丸い根菜。チョコボの飼料として知られる",
		}},
		// Stemming: chs carries the literal word so a chs search reaches this row, and the
		// en column only matches once both sides are stemmed.
		{Sheet: "Balloon", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "hardly 勉强",
			"tc":  "勉強",
			"en":  "Such hardliness is rarely rewarded.",
			"ja":  "ほとんど",
		}},
		// Bigrams: untranslated, so an en search reaches this row and the chs column has to
		// be highlighted through the cjk analyzer.
		{Sheet: "Addon", RowID: "2", Index: 1, Values: map[string]string{
			"chs": "萨维奈圆葱",
			"tc":  "薩維奈圓蔥",
			"en":  "萨维奈圆葱",
			"ja":  "萨维奈圆葱",
		}},
	}
}

func newFragmentStore(t *testing.T) *Store {
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
	if err := indexItems(idx, fragmentFixture()); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

var fragmentFields = []string{"chs", "tc", "en", "ja"}

func itemAt(t *testing.T, result *SearchResult, sheet, rowID string) *Item {
	t.Helper()

	rank := rankOf(result.Items, sheet, rowID)
	if rank == -1 {
		t.Fatalf("%s#%s missing from %v", sheet, rowID, rowIDsOf(result.Items))
	}
	return result.Items[rank]
}

// plainLen is the fragment's length with the mark tags and the ellipsis taken out, so it
// can be compared against the complete value.
func plainLen(fragment string) int {
	stripped := strings.NewReplacer("<mark>", "", "</mark>", "", "…", "").Replace(fragment)
	return len([]rune(stripped))
}

// A language the search never touched still gets its match highlighted, which only works
// because the matching runs through the field's analyzer: the query has "Software" with
// straight quotes and the stored text has “Software” with curly ones.
func TestSearchHighlightsLanguagesOutsideTheSearch(t *testing.T) {
	st := newFragmentStore(t)

	result, err := st.Search(straightQuoteQuery, []string{"chs"}, "", 0, 10, fragmentFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	item := itemAt(t, result, "AddonTransient", "173")

	for _, lang := range fragmentFields {
		got := item.Highlights[lang]
		if got == "" {
			t.Errorf("highlights[%s] missing", lang)
			continue
		}
		if !strings.Contains(got, "<mark>") {
			t.Errorf("highlights[%s] = %q, want a marked match", lang, got)
		}
	}

	// Guards the reason this lives in the backend: comparing characters cannot match.
	if strings.Contains(curlyQuoteValue, `"Software"`) {
		t.Fatal("fixture no longer uses curly quotes, the case it guards is gone")
	}
}

func TestSearchHighlightsCoverEveryRequestedField(t *testing.T) {
	st := newFragmentStore(t)

	result, err := st.Search(straightQuoteQuery, []string{"chs"}, "", 0, 10, fragmentFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	item := itemAt(t, result, "AddonTransient", "173")

	got := make([]string, 0, len(item.Highlights))
	for lang := range item.Highlights {
		got = append(got, lang)
	}
	slices.Sort(got)

	want := slices.Clone(fragmentFields)
	slices.Sort(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("highlights keys = %v, want %v", got, want)
	}
}

// When a language genuinely has no match it still gets a fragment, just without a mark.
// That is the "show me the beginning at least" case, and it keeps the columns comparable.
func TestSearchPreviewsLanguagesWithoutAMatch(t *testing.T) {
	st := newFragmentStore(t)

	result, err := st.Search("圆葱", []string{"chs"}, "", 0, 10, fragmentFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	item := itemAt(t, result, "Item", "8166")

	if got := item.Highlights["chs"]; !strings.Contains(got, "<mark>") {
		t.Errorf("highlights[chs] = %q, want the match marked", got)
	}

	preview, ok := item.Highlights["en"]
	if !ok {
		t.Fatalf("highlights[en] missing, got %v", item.Highlights)
	}
	if strings.Contains(preview, "<mark>") {
		t.Errorf("highlights[en] = %q, want no mark because en has no match", preview)
	}
	if !strings.HasPrefix(preview, "Thavnairian onion") {
		t.Errorf("highlights[en] = %q, want it to start at the beginning of the value", preview)
	}
}

// The complaint that started this: one column showed ~200 characters and the rest showed
// 4500. Every column must now land in the same ballpark.
func TestSearchFragmentsAreComparableInLength(t *testing.T) {
	st := newFragmentStore(t)

	result, err := st.Search(straightQuoteQuery, []string{"chs"}, "", 0, 10, fragmentFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	item := itemAt(t, result, "AddonTransient", "173")

	full := len([]rune(curlyQuoteValue))
	shortest, longest := full, 0
	for _, lang := range fragmentFields {
		n := plainLen(item.Highlights[lang])
		if n > longest {
			longest = n
		}
		if n < shortest {
			shortest = n
		}
	}

	if longest >= full {
		t.Errorf("longest fragment is %d runes against a %d rune value, nothing was trimmed",
			longest, full)
	}
	if shortest == 0 {
		t.Fatal("some fragment came back empty")
	}
	// Same fragmenter for every column, so they cannot differ by an order of magnitude.
	if longest > shortest*3 {
		t.Errorf("fragment lengths %d..%d are too far apart", shortest, longest)
	}
}

// Matching goes through the analyzer, so it sees what the index saw. For en that means
// stemming: the query says hardly and the value says hardliness.
func TestSearchHighlightsStemmedMatchOutsideTheSearch(t *testing.T) {
	st := newFragmentStore(t)

	result, err := st.Search("hardly", []string{"chs"}, "", 0, 10, fragmentFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	item := itemAt(t, result, "Balloon", "1")

	if got := item.Highlights["en"]; !strings.Contains(got, "<mark>hardliness</mark>") {
		t.Errorf("highlights[en] = %q, want hardliness marked through stemming", got)
	}
}

// And for chs it means bigrams, on a language the search never touched.
func TestSearchHighlightsCJKOutsideTheSearch(t *testing.T) {
	st := newFragmentStore(t)

	result, err := st.Search("圆葱", []string{"en"}, "", 0, 10, fragmentFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	item := itemAt(t, result, "Addon", "2")

	if got := item.Highlights["chs"]; !strings.Contains(got, "<mark>圆葱</mark>") {
		t.Errorf("highlights[chs] = %q, want 圆葱 marked through the cjk analyzer", got)
	}
}

func TestItemIndexMappingExposesAnalyzers(t *testing.T) {
	m, ok := buildItemIndexMapping().(*mapping.IndexMappingImpl)
	if !ok {
		t.Fatalf("mapping type = %T", buildItemIndexMapping())
	}
	if got := m.AnalyzerNamed(m.AnalyzerNameForPath("en")); got == nil {
		t.Fatal("no analyzer for en")
	}
}
