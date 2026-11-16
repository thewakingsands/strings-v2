package store

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
)

// sheetTestItems all match "onion" in en, at scores spread by the value length, and
// the sheets hold different numbers of rows so that a sheet term scored into the query
// would carry a different IDF for each sheet and reorder the rows.
func sheetTestItems() []*Item {
	sheets := []string{"Item", "Item", "Item", "Action", "Action", "quest/001/A", "quest/002/B", "cut_scene/050/C", "custom/000/D"}

	items := make([]*Item, 0, len(sheets)*2)
	for i, sheet := range sheets {
		for j := range 2 {
			n := len(items)
			items = append(items, &Item{
				Sheet:  sheet,
				RowID:  strconv.Itoa(n + 1),
				Index:  uint32(i*2 + j),
				Values: map[string]string{"en": "onion" + strings.Repeat(" red", n)},
			})
		}
	}
	return items
}

func newSheetTestStore(t *testing.T) *Store {
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

	if err := indexItems(idx, sheetTestItems()); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

// globMatch states the pattern contract independently of bleve. path.Match will not
// do, because its * stops at a /.
func globMatch(pattern, name string) bool {
	var re strings.Builder
	re.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			re.WriteString(".*")
		case '?':
			re.WriteString(".")
		default:
			re.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	re.WriteString("$")
	return regexp.MustCompile(re.String()).MatchString(name)
}

func (f SheetFilter) passes(sheet string) bool {
	matchesAny := func(patterns []string) bool {
		return slices.ContainsFunc(patterns, func(p string) bool { return globMatch(p, sheet) })
	}
	return (len(f.Include) == 0 || matchesAny(f.Include)) && !matchesAny(f.Exclude)
}

var sheetFilterCases = []struct {
	name   string
	filter SheetFilter
}{
	{"none", SheetFilter{}},
	{"exact name", SheetFilter{Include: []string{"Item"}}},
	{"prefix", SheetFilter{Include: []string{"quest/*"}}},
	{"two prefixes", SheetFilter{Include: []string{"quest/*", "cut_scene/*"}}},
	{"one exclusion", SheetFilter{Exclude: []string{"Item"}}},
	{"two exclusions", SheetFilter{Exclude: []string{"Item", "Action"}}},
	{"prefix minus a prefix", SheetFilter{Include: []string{"quest/*"}, Exclude: []string{"quest/002/*"}}},
	{"single character", SheetFilter{Include: []string{"?tem"}}},
	{"everything", SheetFilter{Include: []string{"*"}}},
	{"two exact names", SheetFilter{Include: []string{"Item", "custom/000/D"}}},
}

// assertFilteredOrder checks that filtered is all, with the rows filter rejects removed.
func assertFilteredOrder(t *testing.T, filter SheetFilter, all, filtered *SearchResult) {
	t.Helper()

	var want []string
	for _, item := range all.Items {
		if filter.passes(item.Sheet) {
			want = append(want, item.Sheet+"#"+item.RowID)
		}
	}
	got := rowIDsOf(filtered.Items)
	if !slices.Equal(got, want) {
		t.Errorf("order changed by the filter\n  got:  %v\n  want: %v", got, want)
	}
	if filtered.Total != uint64(len(want)) {
		t.Errorf("total = %d, want %d", filtered.Total, len(want))
	}
}

func TestSheetFilterKeepsTheUnfilteredOrder(t *testing.T) {
	st := newSheetTestStore(t)
	langs := []string{"en"}

	all, err := st.Search(t.Context(), "onion", langs, SheetFilter{}, 0, 100, langs)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if all.Total != uint64(len(sheetTestItems())) {
		t.Fatalf("fixture: unfiltered total = %d, want every row", all.Total)
	}

	for _, tc := range sheetFilterCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered, err := st.Search(t.Context(), "onion", langs, tc.filter, 0, 100, langs)
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			assertFilteredOrder(t, tc.filter, all, filtered)
		})
	}
}

func TestAdvancedSheetFilterKeepsTheUnfilteredOrder(t *testing.T) {
	st := newSheetTestStore(t)
	langs := []string{"en"}

	all, err := st.AdvancedSearch(t.Context(), "onion", langs, SheetFilter{}, 0, 100, langs)
	if err != nil {
		t.Fatalf("advanced search: %v", err)
	}

	for _, filter := range []SheetFilter{
		{Include: []string{"quest/*"}},
		{Exclude: []string{"Item"}},
	} {
		filtered, err := st.AdvancedSearch(t.Context(), "onion", langs, filter, 0, 100, langs)
		if err != nil {
			t.Fatalf("advanced search with %+v: %v", filter, err)
		}
		assertFilteredOrder(t, filter, all, filtered)
	}
}

func TestSheetFilterMatchingNothingIsEmpty(t *testing.T) {
	st := newSheetTestStore(t)

	for _, filter := range []SheetFilter{
		{Include: []string{"nothing/*"}},
		// Sheet names are matched case-sensitively.
		{Include: []string{"item"}},
		{Include: []string{"ITEM*"}},
	} {
		result, err := st.Search(t.Context(), "onion", []string{"en"}, filter, 0, 100, []string{"en"})
		if err != nil {
			t.Fatalf("search with %+v: %v", filter, err)
		}
		if result.Total != 0 {
			t.Errorf("%+v: total = %d, want 0: %v", filter, result.Total, rowIDsOf(result.Items))
		}
	}
}
