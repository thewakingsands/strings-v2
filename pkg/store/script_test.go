package store

import (
	"fmt"
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

	result, err := st.Search(t.Context(), "师傅大人", []string{"chs", "en", "ja"}, SheetFilter{}, 0, 100, scriptFields)
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
			got, err := st.Search(t.Context(), q, []string{"en"}, SheetFilter{}, 0, 100, scriptFields)
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

func TestScriptOf(t *testing.T) {
	cases := []struct {
		query string
		want  queryScript
	}{
		{"师傅大人", nonLatinScript},
		{"大爆発", nonLatinScript},
		{"街人", nonLatinScript},
		{"ー", nonLatinScript}, // the prolonged sound mark is Common script, not Katakana
		{"한국어", nonLatinScript},
		{"オニオンスープ", kanaScript},
		{"ことになった", kanaScript},
		{"ｵﾆｵﾝ", kanaScript},
		{"Aランク", kanaScript},
		{"味方のＨＰを回復する。", kanaScript},
		{"wizard の茄子", kanaScript},
		{"onion", latinScript},
		{"Zwiebel", latinScript},
		{"Fräulein", latinScript}, // ä is Latin
		{"café", latinScript},     // é is Latin
		{"But how?", latinScript},
		{"Lv. 50", latinScript},
		{"wizard 茄子", mixedScript},
		{"茄子 wizard", mixedScript},
		{"・", mixedScript},
		{"123", mixedScript},
		{"!!!", mixedScript},
		{"", mixedScript},
		{"   ", mixedScript},
	}

	for _, tc := range cases {
		if got := scriptOf(tc.query); got != tc.want {
			t.Errorf("scriptOf(%q) = %v, want %v", tc.query, got, tc.want)
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

// Rows whose CJK columns hold English nobody translated, the way the corpus's credits and
// debug strings do, and two rows where a Latin letter is part of a Japanese name.
func latinInCJKFixture() []*Item {
	return []*Item{
		{Sheet: "Addon", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "But how could it be?", "tc": "But how could it be?",
			"en": "But how could it be?", "ja": "But how could it be?",
		}},
		{Sheet: "Addon", RowID: "2", Index: 1, Values: map[string]string{
			"chs": "时候到了，出发吧。", "tc": "時候到了，出發吧。",
			"en": "Now the time has come to depart.", "ja": "Now the time has come to depart.",
		}},
		{Sheet: "Rank", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "A级", "tc": "A級", "en": "Rank A", "ja": "Aランク",
		}},
		{Sheet: "Rank", RowID: "2", Index: 1, Values: map[string]string{
			"chs": "B级", "tc": "B級", "en": "Rank B", "ja": "Bランク",
		}},
		{Sheet: "Item", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "洋葱", "tc": "洋蔥", "en": "onion", "ja": "オニオン",
		}},
	}
}

func newLatinInCJKStore(t *testing.T) *Store {
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
	if err := indexItems(idx, latinInCJKFixture()); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

// A query the en column has no term for has none in any other column either, whichever
// languages are searched and in either mode. It is also what the UI's zero-result probe
// asks each language alone, so a match here would offer to switch to that language.
func TestSearchOfOnlyStopWordsFindsNothingInAnyColumn(t *testing.T) {
	st := newLatinInCJKStore(t)

	for _, langs := range [][]string{{"en"}, {"chs"}, {"ja"}, {"en", "chs", "ja"}, {"ja", "chs", "tc"}} {
		for _, q := range []string{"But how", "but how could it be", `"but how"`} {
			plain, err := st.Search(t.Context(), q, langs, SheetFilter{}, 0, 10, scriptFields)
			if err != nil {
				t.Fatalf("search %q in %v: %v", q, langs, err)
			}
			if plain.Total != 0 {
				t.Errorf("search %q in %v = %v, want nothing", q, langs, rowIDsOf(plain.Items))
			}

			advanced, err := st.AdvancedSearch(t.Context(), q, langs, SheetFilter{}, 0, 10, scriptFields)
			if err != nil {
				t.Fatalf("advanced search %q in %v: %v", q, langs, err)
			}
			if advanced.Total != 0 {
				t.Errorf("advanced search %q in %v = %v, want nothing", q, langs, rowIDsOf(advanced.Items))
			}
		}
	}
}

// Dropping the stop words leaves the rest of the query matching untranslated English as
// before, and none of them are marked.
func TestSearchFindsUntranslatedEnglishByItsOtherWords(t *testing.T) {
	st := newLatinInCJKStore(t)

	for _, langs := range [][]string{{"ja"}, {"en", "ja"}, {"en"}} {
		result, err := st.Search(t.Context(), "the depart", langs, SheetFilter{}, 0, 10, scriptFields)
		if err != nil {
			t.Fatalf("search in %v: %v", langs, err)
		}
		if got := rowIDsOf(result.Items); strings.Join(got, " ") != "Addon#2" {
			t.Fatalf("search in %v = %v, want Addon#2 alone", langs, got)
		}

		// en alone leaves ja to the display fragments; the others search it.
		highlight := result.Items[0].Highlights["ja"]
		if !strings.Contains(highlight, "<mark>depart</mark>") {
			t.Errorf("search in %v: highlights[ja] = %q, want depart marked", langs, highlight)
		}
		if strings.Contains(highlight, "<mark>the</mark>") {
			t.Errorf("search in %v: highlights[ja] = %q, marks a stop word", langs, highlight)
		}
	}
}

// A Latin letter beside kana is part of a Japanese name, and the A of Aランク is what
// tells it from Bランク, stop word or not.
func TestSearchKeepsStopWordsInQueriesWithOtherScripts(t *testing.T) {
	st := newLatinInCJKStore(t)

	for _, langs := range [][]string{{"ja"}, {"ja", "chs", "en"}} {
		q := parseSearchQuery("Aランク", langs, SheetFilter{})
		a, b := scoreOf(t, st, q, "Rank@1"), scoreOf(t, st, q, "Rank@2")
		if b <= 0 || a <= b {
			t.Errorf("search in %v scored Aランク %v and Bランク %v, want both matched and A ahead",
				langs, a, b)
		}
	}
}

// The analyzer is the CJK columns' own with the stop list after it, so everything else
// it does to a query, such as folding full-width letters, still agrees with the index.
func TestLatinInCJKAnalyzerIsTheCJKAnalyzerWithoutStopWords(t *testing.T) {
	m, ok := buildItemIndexMapping().(*mapping.IndexMappingImpl)
	if !ok {
		t.Fatalf("mapping type = %T", buildItemIndexMapping())
	}
	analyzer := m.AnalyzerNamed(latinInCJKAnalyzer)
	if analyzer == nil {
		t.Fatalf("no analyzer named %s", latinInCJKAnalyzer)
	}

	var got []string
	for _, token := range analyzer.Analyze([]byte("But how ＤＥＰＡＲＴ, the Lalafell's")) {
		got = append(got, string(token.Term))
	}
	if want := []string{"depart", "lalafell's"}; strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("tokens = %q, want %q", got, want)
	}
}

func TestQueryAnalyzer(t *testing.T) {
	for _, lang := range cjkAnalyzedLanguages {
		if got := queryAnalyzer(lang, latinScript); got != latinInCJKAnalyzer {
			t.Errorf("queryAnalyzer(%s, latin) = %q, want %q", lang, got, latinInCJKAnalyzer)
		}
		for _, script := range []queryScript{mixedScript, nonLatinScript, kanaScript} {
			if got := queryAnalyzer(lang, script); got != "" {
				t.Errorf("queryAnalyzer(%s, %v) = %q, want the field's own", lang, script, got)
			}
		}
	}
	for _, lang := range []string{"en", "de", "fr"} {
		for _, script := range []queryScript{mixedScript, latinScript, nonLatinScript, kanaScript} {
			if got := queryAnalyzer(lang, script); got != "" {
				t.Errorf("queryAnalyzer(%s, %v) = %q, want the field's own", lang, script, got)
			}
		}
	}
}

// The shape latinInCJKBoost answers: the line a query comes from sits in the en column of
// one row, while another row's CJK columns kept a single word of it in English. time is
// common in en and rare in the CJK columns, so undamped, the one-word value outscores the
// line.
func untranslatedWordFixture() []*Item {
	items := []*Item{
		{Sheet: "Quest", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "出发的时候到了。", "tc": "出發的時候到了。",
			"en": "Now the time has come to depart.", "ja": "出発の時が来た。",
		}},
		{Sheet: "Addon", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "TIME", "tc": "TIME", "en": "Duration", "ja": "TIME",
		}},
	}
	for i, en := range []string{"Time flies.", "A time to rest.", "Time and tide.", "Some other time.",
		"Time's up.", "Time out.", "In time.", "Time after time."} {
		items = append(items, &Item{Sheet: "Balloon", RowID: fmt.Sprint(i), Index: uint32(i), Values: map[string]string{
			"chs": "时光飞逝", "tc": "時光飛逝", "en": en, "ja": "光陰矢の如し",
		}})
	}
	return items
}

func TestSearchRanksTheLineAboveAWordLeftUntranslatedInCJK(t *testing.T) {
	st := newScorchTestStore(t, untranslatedWordFixture())

	for _, langs := range [][]string{{"en", "chs", "ja"}, {"ja", "chs", "en"}, {"chs", "ja", "en"}} {
		result, err := st.Search(t.Context(), "the time", langs, SheetFilter{}, 0, 20, scriptFields)
		if err != nil {
			t.Fatalf("search %v: %v", langs, err)
		}
		line, word := rankOf(result.Items, "Quest", "1"), rankOf(result.Items, "Addon", "1")
		if line == -1 || word == -1 || word < line {
			t.Errorf("lang=%v = %v, want Quest#1 above Addon#1", langs, rowIDsOf(result.Items))
		}
	}
}

// Like nonNativeScriptBoost, the damping is a multiplier on every document in the field,
// so searching a CJK column alone for English left untranslated there keeps its order.
func TestSearchKeepsSingleLanguageOrderForLatinQueries(t *testing.T) {
	st := newScorchTestStore(t, untranslatedWordFixture())

	for _, q := range []string{"the time", "Now the time has come", "TIME duration"} {
		got, err := st.Search(t.Context(), q, []string{"ja"}, SheetFilter{}, 0, 100, scriptFields)
		if err != nil {
			t.Fatalf("search %q: %v", q, err)
		}

		// Control: the same analysis with no boost at all.
		plain := bleve.NewMatchQuery(q)
		plain.SetField("ja")
		plain.Analyzer = latinInCJKAnalyzer
		request := bleve.NewSearchRequestOptions(plain, 100, 0, false)
		control, err := st.index.Search(request)
		if err != nil {
			t.Fatalf("control search %q: %v", q, err)
		}

		gotIDs := make([]string, 0, len(got.Items))
		for _, item := range got.Items {
			gotIDs = append(gotIDs, item.Sheet+"@"+item.RowID)
		}
		wantIDs := make([]string, 0, len(control.Hits))
		for _, hit := range control.Hits {
			wantIDs = append(wantIDs, hit.ID)
		}
		if got.Total != control.Total || strings.Join(gotIDs, " ") != strings.Join(wantIDs, " ") {
			t.Errorf("search %q = %v (total %d), want %v (total %d)",
				q, gotIDs, got.Total, wantIDs, control.Total)
		}
	}
}

// untranslated fills every column but ja with the Japanese line, the way rows nobody
// translated reach the corpus.
func untranslated(ja string) map[string]string {
	values := map[string]string{}
	for _, lang := range []string{"chs", "tc", "ko", "en", "de", "fr", "ja"} {
		values[lang] = ja
	}
	return values
}

// The shape kanaInChineseKoreanBoost answers: the line a query comes from sits in the ja
// column of one row, while another row kept a Japanese line in every column and shares
// its kana. Those bigrams are common in ja and rare everywhere else, so undamped, the
// untranslated copy outscores the line. The second pair puts ＨＰ in the query, whose
// Latin letters must not exempt en, de and fr from their damping.
func untranslatedJapaneseFixture() []*Item {
	items := []*Item{
		{Sheet: "Quest", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "决定尝试模仿魔物的咆哮。", "tc": "決定嘗試模仿魔物的咆哮。", "ko": "마물의 포효를 흉내 내기로 했다.",
			"en": "We resolved to imitate the beast's roar.", "de": "Wir ahmen das Brüllen nach.",
			"fr": "Nous imiterons le rugissement.", "ja": "魔物の咆哮を試みることになった。",
		}},
		{Sheet: "Leve", RowID: "1", Index: 0, Values: untranslated("掃除することになった")},
		{Sheet: "Action", RowID: "1", Index: 0, Values: map[string]string{
			"chs": "恢复目标的体力。", "tc": "恢復目標的體力。", "ko": "대상의 HP를 회복한다.",
			"en": "Restores target's HP.", "de": "Stellt LP des Ziels wieder her.",
			"fr": "Restaure les PV de la cible.", "ja": "味方のＨＰを回復する。",
		}},
		{Sheet: "Status", RowID: "1", Index: 0, Values: untranslated("ＨＰを回復する")},
	}
	for i, ja := range []string{"旅に出ることになった。", "店を継ぐことになった。", "村を守ることになった。",
		"城へ行くことになった。", "海を渡ることになった。", "森を調べることになった。",
		"ＨＰを回復する薬。", "毎秒ＨＰを回復する。", "仲間のＨＰを回復する。"} {
		items = append(items, &Item{Sheet: "Balloon", RowID: fmt.Sprint(i), Index: uint32(i), Values: map[string]string{
			"chs": "旅途还在继续", "tc": "旅途還在繼續", "ko": "여행은 계속된다", "en": "The journey goes on.",
			"de": "Die Reise geht weiter.", "fr": "Le voyage continue.", "ja": ja,
		}})
	}
	return items
}

var untranslatedJapaneseFields = []string{"chs", "tc", "ko", "en", "de", "fr", "ja"}

func TestSearchRanksTheLineAboveJapaneseLeftUntranslated(t *testing.T) {
	st := newScorchTestStore(t, untranslatedJapaneseFixture())

	cases := []struct {
		q          string
		line, copy [2]string
	}{
		{"試みることになった", [2]string{"Quest", "1"}, [2]string{"Leve", "1"}},
		{"味方のＨＰを回復する", [2]string{"Action", "1"}, [2]string{"Status", "1"}},
	}
	for _, tc := range cases {
		for _, langs := range [][]string{
			{"ja", "chs", "en"}, {"chs", "en", "ja"}, {"ja", "tc", "ko"}, {"ko", "ja"},
			{"ja", "en", "de", "fr"}, untranslatedJapaneseFields,
		} {
			result, err := st.Search(t.Context(), tc.q, langs, SheetFilter{}, 0, 20, untranslatedJapaneseFields)
			if err != nil {
				t.Fatalf("search %q in %v: %v", tc.q, langs, err)
			}
			line := rankOf(result.Items, tc.line[0], tc.line[1])
			copied := rankOf(result.Items, tc.copy[0], tc.copy[1])
			if line == -1 || copied == -1 || copied < line {
				t.Errorf("search %q in %v = %v, want %s#%s above %s#%s",
					tc.q, langs, rowIDsOf(result.Items), tc.line[0], tc.line[1], tc.copy[0], tc.copy[1])
			}
		}
	}
}

// Japanese left untranslated in a Chinese or Korean column stays findable by searching
// that column alone, in the same order as with no damping at all.
func TestSearchKeepsSingleLanguageOrderForKanaQueries(t *testing.T) {
	st := newScorchTestStore(t, untranslatedJapaneseFixture())

	for _, lang := range []string{"chs", "tc", "ko"} {
		for _, q := range []string{"ことになった", "味方のＨＰを回復する"} {
			got, err := st.Search(t.Context(), q, []string{lang}, SheetFilter{}, 0, 100, untranslatedJapaneseFields)
			if err != nil {
				t.Fatalf("search %q in %s: %v", q, lang, err)
			}

			// Control: the same query with no boost at all.
			plain := bleve.NewMatchQuery(q)
			plain.SetField(lang)
			request := bleve.NewSearchRequestOptions(plain, 100, 0, false)
			control, err := st.index.Search(request)
			if err != nil {
				t.Fatalf("control search %q in %s: %v", q, lang, err)
			}

			gotIDs := make([]string, 0, len(got.Items))
			for _, item := range got.Items {
				gotIDs = append(gotIDs, item.Sheet+"@"+item.RowID)
			}
			wantIDs := make([]string, 0, len(control.Hits))
			for _, hit := range control.Hits {
				wantIDs = append(wantIDs, hit.ID)
			}
			if got.Total != control.Total || strings.Join(gotIDs, " ") != strings.Join(wantIDs, " ") {
				t.Errorf("search %q in %s = %v (total %d), want %v (total %d)",
					q, lang, gotIDs, got.Total, wantIDs, control.Total)
			}
		}
	}
}

// ja is where kana belongs, so it is the one CJK column a kana query leaves at full weight.
func TestLanguageBoostForKanaQueries(t *testing.T) {
	want := map[string]float64{
		"ja": 1, "chs": kanaInChineseKoreanBoost, "tc": kanaInChineseKoreanBoost, "ko": kanaInChineseKoreanBoost,
		"en": nonNativeScriptBoost, "de": nonNativeScriptBoost, "fr": nonNativeScriptBoost,
	}
	for lang, boost := range want {
		if got := languageBoost(0, lang, kanaScript); got != boost {
			t.Errorf("languageBoost(0, %s, kana) = %v, want %v", lang, got, boost)
		}
	}
}
