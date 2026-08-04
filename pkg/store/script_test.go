package store

import (
	"sort"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// The shape reported from production: rows whose en column was never translated and
// still holds CJK, next to rows where the Chinese column has the real match. A CJK
// query is cut into single characters on the en field, so a row like 大爆発 matches on
// nothing but the character 大, yet scores high because the value is short and that
// character is vanishingly rare in an English column.
func scriptFixture() []*Item {
	return []*Item{
		{Sheet: "Action", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "大爆発", "tc": "大爆発", "en": "大爆発", "ja": "大爆発",
		}},
		{Sheet: "Action", RowID: "2", Index: 1, Values: map[string]string{
			"chs": "大放電", "tc": "大放電", "en": "大放電", "ja": "大放電",
		}},
		{Sheet: "ENpcResident", RowID: "3", Index: 0, Values: map[string]string{
			"chs": "街人A", "tc": "街人A", "en": "街人A", "ja": "街人A",
		}},
		// The rows a search for 师傅大人 should actually surface.
		{Sheet: "PlaceName", RowID: "4", Index: 0, Values: map[string]string{
			"chs": "魔晶石师傅", "tc": "魔晶石師傅", "en": "Materia Master", "ja": "マテリア師範",
		}},
		{Sheet: "RacingChocoboName", RowID: "5", Index: 0, Values: map[string]string{
			"chs": "大人", "tc": "大人", "en": "Lord", "ja": "ロード",
		}},
		// Padding so term statistics are not degenerate.
		{Sheet: "Item", RowID: "6", Index: 0, Values: map[string]string{
			"chs": "洋葱", "tc": "洋蔥", "en": "onion", "ja": "オニオン",
		}},
		{Sheet: "Item", RowID: "7", Index: 1, Values: map[string]string{
			"chs": "洋葱汤", "tc": "洋蔥湯", "en": "onion soup", "ja": "オニオンスープ",
		}},
	}
}

func newScriptStore(t *testing.T) *Store {
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
	if err := indexItems(idx, scriptFixture()); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

var scriptFields = []string{"chs", "tc", "en", "ja"}

func TestSearchRanksChineseMatchAboveEnglishColumnCJKNoise(t *testing.T) {
	st := newScriptStore(t)

	result, err := st.Search("师傅大人", []string{"chs", "en", "ja"}, "", 0, 100, scriptFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	got := rowIDsOf(result.Items)
	wanted := rankOf(result.Items, "PlaceName", "4")
	if wanted == -1 {
		t.Fatalf("魔晶石师傅 missing from %v", got)
	}

	// Rows that only match because a single character leaked into their en column.
	for _, noise := range [][2]string{{"Action", "1"}, {"Action", "2"}, {"ENpcResident", "3"}} {
		rank := rankOf(result.Items, noise[0], noise[1])
		if rank != -1 && rank < wanted {
			t.Errorf("%s#%s matches only a stray character in its en column but outranked 魔晶石师傅: %v",
				noise[0], noise[1], got)
		}
	}
}

// Damping is a multiplier applied to every document in the field alike, so searching
// that one language on its own must come back in exactly the same order. This is what
// keeps looking for untranslated text in the en column working.
func TestSearchKeepsSingleLanguageOrderForNonLatinQueries(t *testing.T) {
	st := newScriptStore(t)

	for _, q := range []string{"大爆発", "街人", "师傅大人"} {
		t.Run(q, func(t *testing.T) {
			got, err := st.Search(q, []string{"en"}, "", 0, 100, scriptFields)
			if err != nil {
				t.Fatalf("search: %v", err)
			}

			// Control: the same query with no boost at all.
			plain := bleve.NewMatchQuery(q)
			plain.SetField("en")
			request := bleve.NewSearchRequestOptions(plain, 100, 0, false)
			request.Fields = []string{"sheet", "id"}
			control, err := st.index.Search(request)
			if err != nil {
				t.Fatalf("control search: %v", err)
			}

			wantIDs := make([]string, 0, len(control.Hits))
			for _, hit := range control.Hits {
				wantIDs = append(wantIDs, hit.ID)
			}
			gotIDs := make([]string, 0, len(got.Items))
			for _, item := range got.Items {
				gotIDs = append(gotIDs, item.Sheet+"@"+item.RowID)
			}

			if got.Total != control.Total {
				t.Errorf("total = %d, want %d — damping must not change recall", got.Total, control.Total)
			}
			if strings.Join(gotIDs, " ") != strings.Join(wantIDs, " ") {
				t.Errorf("order changed\n got: %v\nwant: %v", gotIDs, wantIDs)
			}
		})
	}
}

func TestIsNonLatinQuery(t *testing.T) {
	cases := []struct {
		query string
		want  bool
	}{
		{"师傅大人", true},
		{"大爆発", true},
		{"街人", true},
		{"オニオンスープ", true}, // the prolonged sound mark ー is Common script, not Katakana
		{"ー", true},
		{"한국어", true},
		{"onion", false},
		{"Zwiebel", false},
		{"Fräulein", false}, // ä is Latin
		{"café", false},     // é is Latin
		{"wizard 茄子", false},
		{"茄子 wizard", false},
		{"123", false},
		{"!!!", false},
		{"", false},
		{"   ", false},
	}

	for _, tc := range cases {
		if got := isNonLatinQuery(tc.query); got != tc.want {
			t.Errorf("isNonLatinQuery(%q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}

// Locks cjkAnalyzedLanguages against the mapping it is meant to describe.
func TestItemIndexMappingAnalyzers(t *testing.T) {
	m, ok := buildItemIndexMapping().(*mapping.IndexMappingImpl)
	if !ok {
		t.Fatalf("mapping type = %T, want *mapping.IndexMappingImpl", buildItemIndexMapping())
	}

	want := map[string]string{
		"chs": "cjk", "tc": "cjk", "ja": "cjk", "ko": "cjk",
		"en": "en", "de": "de", "fr": "fr",
	}
	for lang, wantAnalyzer := range want {
		if got := m.AnalyzerNameForPath(lang); got != wantAnalyzer {
			t.Errorf("analyzer for %s = %q, want %q", lang, got, wantAnalyzer)
		}
	}

	gotCJK := make([]string, 0, len(cjkAnalyzedLanguages))
	gotCJK = append(gotCJK, cjkAnalyzedLanguages...)
	sort.Strings(gotCJK)

	wantCJK := []string{"chs", "ja", "ko", "tc"}
	if strings.Join(gotCJK, ",") != strings.Join(wantCJK, ",") {
		t.Errorf("cjkAnalyzedLanguages = %v, want %v", gotCJK, wantCJK)
	}
	for _, lang := range cjkAnalyzedLanguages {
		if !usesCJKAnalyzer(lang) {
			t.Errorf("usesCJKAnalyzer(%q) = false", lang)
		}
	}
	for _, lang := range []string{"en", "de", "fr"} {
		if usesCJKAnalyzer(lang) {
			t.Errorf("usesCJKAnalyzer(%q) = true", lang)
		}
	}
}
