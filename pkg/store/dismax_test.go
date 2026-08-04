package store

import (
	"sort"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
)

// The shape reported from production: a short UI string left untranslated in every
// language next to a real sentence that only the en column can match.
const untranslatedShortValue = "WIN"

const talkValue = "No! I'll never win the tournament at this rate! Please, " +
	"you've got to help me train harder before the next match begins."

func rankingFixture() []*Item {
	return []*Item{
		// Matches "win" in all four columns because nobody translated it.
		{Sheet: "Addon", RowID: "1", Index: 0, Values: map[string]string{
			"chs": untranslatedShortValue,
			"tc":  untranslatedShortValue,
			"en":  untranslatedShortValue,
			"ja":  untranslatedShortValue,
		}},
		// Same, for a second query word.
		{Sheet: "Addon", RowID: "2", Index: 1, Values: map[string]string{
			"chs": "Rating",
			"tc":  "Rating",
			"en":  "Rating",
			"ja":  "Rate",
		}},
		// The valuable hit: many query words, but only in en.
		{Sheet: "DefaultTalk", RowID: "3", Index: 0, Values: map[string]string{
			"chs": "不行！照这样下去我永远赢不了比赛！求你了，下一场开始前得再练练！",
			"tc":  "不行！照這樣下去我永遠贏不了比賽！求你了，下一場開始前得再練練！",
			"en":  talkValue,
			"ja":  "ダメだ！このままじゃ試合に勝てない！次の試合までもっと練習しないと！",
		}},
		// Padding so the term statistics are not degenerate.
		{Sheet: "Addon", RowID: "4", Index: 2, Values: map[string]string{
			"chs": "胜", "tc": "勝", "en": "Victory", "ja": "勝利",
		}},
		{Sheet: "Addon", RowID: "5", Index: 3, Values: map[string]string{
			"chs": "段位积分", "tc": "段位積分", "en": "Rating", "ja": "Rate",
		}},
	}
}

func newRankingStore(t *testing.T) *Store {
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
	if err := indexItems(idx, rankingFixture()); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

const rankingQuery = "I'll never win at rate"

var rankingFields = []string{"chs", "tc", "en", "ja"}

// A sentence matching many query words must outrank a one word string that merely
// repeats across every language column. Searching en alone already gets this right,
// so multi-language must not do worse.
func TestSearchRanksRealSentenceAboveRepeatedShortString(t *testing.T) {
	st := newRankingStore(t)

	single, err := st.Search(rankingQuery, []string{"en"}, "", 0, 100, rankingFields)
	if err != nil {
		t.Fatalf("search en: %v", err)
	}
	if got := rankOf(single.Items, "DefaultTalk", "3"); got != 0 {
		t.Fatalf("lang=en put the sentence at rank %d, want 0 — fixture no longer reproduces the case: %v",
			got, rowIDsOf(single.Items))
	}

	for _, langs := range [][]string{
		{"en", "ja"},
		{"chs", "en"},
		{"chs", "en", "ja"},
		{"chs", "tc", "en", "ja"},
	} {
		t.Run(strings.Join(langs, ","), func(t *testing.T) {
			result, err := st.Search(rankingQuery, langs, "", 0, 100, rankingFields)
			if err != nil {
				t.Fatalf("search: %v", err)
			}

			sentence := rankOf(result.Items, "DefaultTalk", "3")
			if sentence == -1 {
				t.Fatalf("sentence missing from %v", rowIDsOf(result.Items))
			}
			for _, short := range [][2]string{{"Addon", "1"}, {"Addon", "2"}} {
				if r := rankOf(result.Items, short[0], short[1]); r != -1 && r < sentence {
					t.Errorf("%s#%s (%q in every language) outranked the sentence: %v",
						short[0], short[1], untranslatedShortValue, rowIDsOf(result.Items))
				}
			}
		})
	}
}

// Guards the hand written iterator: taking the max instead of the sum may only change
// scores and order, never which documents match.
func TestDisMaxMatchesSameDocumentsAsDisjunction(t *testing.T) {
	st := newRankingStore(t)

	cases := []struct {
		query string
		langs []string
	}{
		{rankingQuery, []string{"chs", "en", "ja"}},
		{rankingQuery, []string{"chs", "tc", "en", "ja"}},
		{"win", []string{"chs", "en"}},
		{"rate", []string{"en", "ja"}},
		{"勝利", []string{"chs", "tc", "ja"}},
		{"tournament", []string{"chs", "en", "ja"}},
		{"nothingmatchesthis", []string{"chs", "en", "ja"}},
	}

	for _, tc := range cases {
		t.Run(tc.query+"/"+strings.Join(tc.langs, ","), func(t *testing.T) {
			disMax := parseSearchQuery(tc.query, tc.langs, "")

			disjuncts := make([]query.Query, 0, len(tc.langs))
			for _, lang := range tc.langs {
				matchQuery := bleve.NewMatchQuery(tc.query)
				matchQuery.SetField(lang)
				disjuncts = append(disjuncts, matchQuery)
			}
			disjunction := bleve.NewDisjunctionQuery(disjuncts...)

			gotTotal, gotIDs := runQuery(t, st, disMax)
			wantTotal, wantIDs := runQuery(t, st, disjunction)

			if gotTotal != wantTotal {
				t.Errorf("total = %d, want %d", gotTotal, wantTotal)
			}
			if strings.Join(gotIDs, " ") != strings.Join(wantIDs, " ") {
				t.Errorf("hit set differs\n dis_max: %v\n disjunct: %v", gotIDs, wantIDs)
			}
		})
	}
}

// runQuery returns the total and the sorted document ids, so callers compare hit sets
// rather than ranking.
func runQuery(t *testing.T, st *Store, q query.Query) (uint64, []string) {
	t.Helper()

	request := bleve.NewSearchRequestOptions(q, 100, 0, false)
	request.Fields = []string{"sheet", "id"}
	results, err := st.index.Search(request)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	ids := make([]string, 0, len(results.Hits))
	for _, hit := range results.Hits {
		ids = append(ids, hit.ID)
	}
	sort.Strings(ids)
	return results.Total, ids
}

// scoreOf runs a query and returns the score of one document, or -1 when it misses.
func scoreOf(t *testing.T, st *Store, q query.Query, docID string) float64 {
	t.Helper()

	request := bleve.NewSearchRequestOptions(q, 100, 0, false)
	results, err := st.index.Search(request)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, hit := range results.Hits {
		if hit.ID == docID {
			return hit.Score
		}
	}
	return -1
}

// The whole point of dis_max: the score is the best clause, not the sum of them.
//
// Asserted through the explanation rather than by comparing against separate
// per-language searches, because a searcher pushes its query norm down into its
// children, so the same clause scores differently once it is wrapped.
func TestDisMaxScoresByBestLanguageNotBySum(t *testing.T) {
	st := newRankingStore(t)

	// Addon#1 is the untranslated "WIN", so every language clause matches it.
	const docID = "Addon@1"
	langs := []string{"chs", "tc", "en", "ja"}

	request := bleve.NewSearchRequestOptions(parseSearchQuery("win", langs, ""), 100, 0, true)
	results, err := st.index.Search(request)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	var hit *search.DocumentMatch
	for _, candidate := range results.Hits {
		if candidate.ID == docID {
			hit = candidate
		}
	}
	if hit == nil {
		t.Fatalf("%s missing from the results", docID)
	}
	if hit.Expl == nil {
		t.Fatal("no explanation returned")
	}
	if hit.Expl.Message != "max of:" {
		t.Errorf("explanation = %q, want %q", hit.Expl.Message, "max of:")
	}
	if len(hit.Expl.Children) != len(langs) {
		t.Fatalf("got %d clause explanations, want %d — fixture should match every language",
			len(hit.Expl.Children), len(langs))
	}

	best, sum := 0.0, 0.0
	for _, child := range hit.Expl.Children {
		sum += child.Value
		if child.Value > best {
			best = child.Value
		}
	}
	if sum <= best {
		t.Fatalf("fixture is degenerate: sum %v is not above the best %v", sum, best)
	}

	if hit.Expl.Value != best {
		t.Errorf("explained score = %v, want the best clause %v, not the sum %v",
			hit.Expl.Value, best, sum)
	}
	if hit.Score != best {
		t.Errorf("score = %v, want the best clause %v", hit.Score, best)
	}
}

// Cross-checks the scoring against bleve's own disjunction on the same data: summing
// four matching languages has to come out well above taking the best one.
func TestDisMaxScoresBelowDisjunctionForRepeatedMatch(t *testing.T) {
	st := newRankingStore(t)

	const docID = "Addon@1"
	langs := []string{"chs", "tc", "en", "ja"}

	disjuncts := make([]query.Query, 0, len(langs))
	for _, lang := range langs {
		matchQuery := bleve.NewMatchQuery("win")
		matchQuery.SetField(lang)
		disjuncts = append(disjuncts, matchQuery)
	}

	summed := scoreOf(t, st, bleve.NewDisjunctionQuery(disjuncts...), docID)
	best := scoreOf(t, st, parseSearchQuery("win", langs, ""), docID)
	if summed < 0 || best < 0 {
		t.Fatalf("%s did not match both queries (disjunction %v, dis_max %v)", docID, summed, best)
	}

	if summed <= best {
		t.Errorf("disjunction scored %v and dis_max %v; dis_max must not be summing clauses",
			summed, best)
	}
}

// Highlights come from Locations, which the searcher has to merge across clauses.
func TestDisMaxKeepsHighlightsFromEveryMatchingLanguage(t *testing.T) {
	st := newRankingStore(t)

	result, err := st.Search("win", []string{"chs", "tc", "en", "ja"}, "", 0, 100, rankingFields)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	rank := rankOf(result.Items, "Addon", "1")
	if rank == -1 {
		t.Fatalf("Addon#1 missing from %v", rowIDsOf(result.Items))
	}
	item := result.Items[rank]

	// "WIN" sits in all four columns, so all four must come back highlighted.
	for _, lang := range rankingFields {
		got, ok := item.Highlights[lang]
		if !ok {
			t.Errorf("highlights[%s] missing, got %v — clause locations were not merged",
				lang, item.Highlights)
			continue
		}
		if !strings.Contains(got, "<mark>") {
			t.Errorf("highlights[%s] = %q, want a marked match", lang, got)
		}
	}
}

// Paging happens inside bleve, so a wrong iterator shows up as duplicated or skipped
// rows between consecutive pages.
func TestDisMaxPaginatesWithoutOverlapOrGaps(t *testing.T) {
	st := newRankingStore(t)
	langs := []string{"chs", "tc", "en", "ja"}

	all, err := st.Search(rankingQuery, langs, "", 0, 100, rankingFields)
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if len(all.Items) < 4 {
		t.Fatalf("got %d items, need at least 4 to page through", len(all.Items))
	}

	var paged []string
	const pageSize = 2
	for offset := 0; offset < len(all.Items); offset += pageSize {
		page, err := st.Search(rankingQuery, langs, "", offset, pageSize, rankingFields)
		if err != nil {
			t.Fatalf("search offset %d: %v", offset, err)
		}
		paged = append(paged, rowIDsOf(page.Items)...)
	}

	want := rowIDsOf(all.Items)
	if strings.Join(paged, " ") != strings.Join(want, " ") {
		t.Errorf("paged through %v, want %v", paged, want)
	}
}
