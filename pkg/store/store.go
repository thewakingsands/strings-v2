package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"
	"unicode"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/blevesearch/bleve/v2/search/query"
)

// Store keeps all items in memory and provides simple lookup helpers.
type Store struct {
	index bleve.Index // Bleve search index
}

type SearchResult struct {
	Items   []*Item
	Total   uint64
	Elapsed time.Duration
}

// indexOpenTimeout bounds the wait for the index lock. Opening takes milliseconds
// when nothing holds it, so a wait this long means someone else has it and failing
// with a message beats blocking forever, which gives no clue at all.
const indexOpenTimeout = "30s"

// openIndexReadOnly opens an index for querying only, waiting at most boltTimeout
// (a Go duration string) for the index lock.
//
// Read only means scorch takes a shared lock on root.bolt instead of an exclusive
// one, so several instances can serve from one index directory at once, and it also
// leaves the persister and merger goroutines unstarted.
//
// This makes "the running server never writes" a hard requirement rather than a
// preference: Scorch.Batch does not check the read only flag, so an accidental write
// would block waiting for a persister that was never started, instead of failing.
// Writes belong solely to BuildIndex, which opens its own writable index and closes
// it before anything serves queries from it.
func openIndexReadOnly(indexDir string, boltTimeout string) (bleve.Index, error) {
	return bleve.OpenUsing(indexDir, map[string]interface{}{
		"read_only":    true,
		"bolt_timeout": boltTimeout,
	})
}

// LoadStore opens the index at indexDir for querying, building it from the JSON files
// in dataDir first if it is not there yet. An index built with a mapping other than this
// binary's is refused with a *MappingMismatchError; see MappingFingerprint.
func LoadStore(dataDir string, indexDir string) (*Store, error) {
	idx, err := openIndexReadOnly(indexDir, indexOpenTimeout)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Building needs write access, so it runs on its own index instance and closes
		// it. Only then is the result opened read only for serving.
		if err := BuildIndex(dataDir, indexDir); err != nil {
			return nil, fmt.Errorf("build item index: %w", err)
		}
		idx, err = openIndexReadOnly(indexDir, indexOpenTimeout)
	}
	if err != nil {
		return nil, fmt.Errorf("open bleve index %s (another instance may hold a write lock on it): %w",
			indexDir, err)
	}

	if err := verifyMapping(idx, indexDir); err != nil {
		return nil, errors.Join(err, idx.Close())
	}
	log.Printf("index mapping %s matches this binary", MappingFingerprint())

	s := &Store{
		index: idx,
	}

	return s, nil
}

var metaFields = []string{"sheet", "id", "index"}

// languageBoostStep separates consecutive languages in the lang parameter. It is
// deliberately tiny: dis_max multiplies it into the winning clause's score, so a large
// factor would let a poor match in an earlier language beat a perfect match in a later
// one. This only settles the order when the languages score about the same.
const languageBoostStep = 0.01

// nonNativeScriptBoost damps the languages whose analyzer can only match the query
// character by character.
//
// A query with no Latin letters is cut into single characters on an en, de or fr field,
// so it matches rows whose column was never translated and still holds CJK. Those
// matches score absurdly high: the values are short, and a lone CJK character appears
// in so few English rows that its IDF dwarfs a proper word match in a Chinese column.
// dis_max cannot pick the better language when one of the scores is inflated to begin
// with, so the inflated one is scaled back into a comparable range.
//
// A multiplier is the point: it applies to every document in that field alike, so
// searching only that language comes back in exactly the same order, and looking for
// untranslated text in the en column keeps working. Recall is untouched either way.
//
// Measured against the real corpus, anything at or below 0.5 already pushes the noise
// off the first page; this sits an order of magnitude further down so that queries with
// an even wider IDF gap between the columns are covered too.
const nonNativeScriptBoost = 0.05

// latinInCJKBoost damps the CJK columns for a query written in Latin letters alone, the
// same problem as nonNativeScriptBoost in the other direction.
//
// The Latin text in those columns is English nobody translated, mostly short: credits,
// labels, markup. A word of it is rare in a CJK column, so its IDF there dwarfs the same
// word's in en, and a value holding that one word outscores the row whose en column
// holds the whole line. The CJK stop words used to hide this by diluting the clause; see
// latinInCJKAnalyzer.
//
// Measured with the relevance eval on publish-20260728-3b64471, each case searching its
// own language and the other two, first and then last. Undamped, 37 rows the cases name
// as noise reached a first page; at 0.5, 10 or 11; at 0.2 and below, 1, and every English
// case ranks exactly where searching en alone puts it. 0.2 down to 0.01 score the same,
// so this takes nonNativeScriptBoost's value from the middle of that range.
const latinInCJKBoost = 0.05

// kanaInChineseKoreanBoost damps chs, tc and ko for a query with kana in it, the same
// problem as latinInCJKBoost from the Japanese side.
//
// Kana in those columns is Japanese nobody translated, often a whole line of it. Its
// bigrams are rare there, so their IDF dwarfs the same bigrams' in ja, and the
// untranslated copy outscores the row whose ja column holds the line.
//
// Measured with the relevance eval on publish-20260728-3b64471, each ja case searching
// ja, chs and en. Undamped, 12 cases rank below where ja alone puts them, 11 of them
// behind Japanese left in chs. At 0.7 or 0.8 about half still do; every value from 0.5
// down to 0.05 leaves only the one written in kanji alone. The cost falls on the 137
// kana lines that sit in chs or tc and in no ja column, searched with that column
// first: at 0.5 the worst falls to rank 7, at 0.05 58 of them leave the first 50. So
// this is the largest value of the flat range rather than its middle.
const kanaInChineseKoreanBoost = 0.5

// latinInCJKAnalyzer is what a query written in Latin letters alone is matched with in
// the CJK columns: their own cjk analyzer, followed by the stop list the en column uses.
//
// The cjk analyzer has no stop list, so English left untranslated in those columns is
// indexed down to its "the" and "but". Without this, a query of nothing but stop words,
// which finds nothing in en, found those rows instead, and every English query matched
// them on its commonest words. The Latin text in the CJK columns is English, markup and
// credits nearly without exception, so the English list is the one that applies.
//
// A query with a letter of any other script keeps its stop words: the A of Aランク is
// part of the name, and the stop list would drop it.
//
// It is registered with bleve rather than added to the index mapping, so it changes how
// a query is analyzed and leaves the index and its fingerprint alone.
const latinInCJKAnalyzer = "cjk_stop_en"

func init() {
	if err := registry.RegisterAnalyzer(latinInCJKAnalyzer, newLatinInCJKAnalyzer); err != nil {
		panic(err)
	}
}

func newLatinInCJKAnalyzer(_ map[string]interface{}, cache *registry.Cache) (analysis.Analyzer, error) {
	analyzer, err := cache.AnalyzerNamed(cjk.AnalyzerName)
	if err != nil {
		return nil, err
	}
	cjkAnalyzer, ok := analyzer.(*analysis.DefaultAnalyzer)
	if !ok {
		return nil, fmt.Errorf("analyzer %s is a %T, not a token filter chain", cjk.AnalyzerName, analyzer)
	}
	stop, err := cache.TokenFilterNamed(en.StopName)
	if err != nil {
		return nil, err
	}

	rv := *cjkAnalyzer
	rv.TokenFilters = append(slices.Clone(cjkAnalyzer.TokenFilters), stop)
	return &rv, nil
}

// queryAnalyzer names the analyzer a query in script is matched with in lang, or returns
// "" for the one the index analyzed lang with.
func queryAnalyzer(lang string, script queryScript) string {
	if script == latinScript && usesCJKAnalyzer(lang) {
		return latinInCJKAnalyzer
	}
	return ""
}

// parseSearchQuery matches q against every language in langs, ranking earlier
// languages above later ones when they are otherwise equally good.
func parseSearchQuery(q string, langs []string, sheet SheetFilter) query.Query {
	script := scriptOf(q)

	disjuncts := make([]query.Query, 0, len(langs))
	for i, lang := range langs {
		matchQuery := bleve.NewMatchQuery(q)
		matchQuery.SetField(lang)
		matchQuery.Analyzer = queryAnalyzer(lang, script)

		// Left alone at exactly 1, which keeps a single language Latin search scoring
		// bit for bit as it did before this parameter accepted a list.
		if boost := languageBoost(i, lang, script); boost != 1 {
			matchQuery.SetBoost(boost)
		}

		disjuncts = append(disjuncts, newTermsMatchQuery(matchQuery, foldsKana(lang)))
	}

	var textQuery query.Query
	if len(disjuncts) == 1 {
		textQuery = disjuncts[0]
	} else {
		textQuery = newDisMaxQuery(disjuncts)
	}

	return sheet.apply(textQuery)
}

// languageBoost weights the clause searching lang, the i-th language asked for, with a
// query written in script.
func languageBoost(i int, lang string, script queryScript) float64 {
	boost := 1.0
	if i > 0 {
		boost = 1 - float64(i)*languageBoostStep
	}
	if (script == nonLatinScript || script == kanaScript) && !usesCJKAnalyzer(lang) {
		boost *= nonNativeScriptBoost
	}
	if script == latinScript && usesCJKAnalyzer(lang) {
		boost *= latinInCJKBoost
	}
	// The kana folded column is ja, the one CJK column where kana is the language's own.
	if script == kanaScript && usesCJKAnalyzer(lang) && !foldsKana(lang) {
		boost *= kanaInChineseKoreanBoost
	}
	return boost
}

// queryScript is what the letters of a query say about the analyzers that can make
// sense of it.
type queryScript int

const (
	// mixedScript has letters of both kinds, or none at all, and no kana.
	mixedScript queryScript = iota
	// latinScript has letters, and every one of them is Latin.
	latinScript
	// nonLatinScript has letters and none of them are Latin or kana. Such a query
	// carries no word an en, de or fr analyzer can recognise, so those fields can only
	// match it a character at a time.
	nonLatinScript
	// kanaScript has a hiragana or katakana letter, which makes it Japanese whatever
	// else it holds: the ＨＰ of ＨＰを回復 and the A of Aランク are part of the Japanese,
	// not English words. A query of kanji alone has no such mark and stays nonLatinScript.
	kanaScript
)

// scriptOf reports the script q is written in. German ü and French é are Latin letters,
// ー and ・ are not kana since chs, tc and ko write them too, and digits or punctuation
// say nothing either way.
func scriptOf(q string) queryScript {
	latin, other := false, false
	for _, r := range q {
		if !unicode.IsLetter(r) {
			continue
		}
		switch {
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			return kanaScript
		case unicode.Is(unicode.Latin, r):
			latin = true
		default:
			other = true
		}
	}
	switch {
	case latin && !other:
		return latinScript
	case other && !latin:
		return nonLatinScript
	}
	return mixedScript
}

// Search finds items whose value in any of the given languages matches the query.
// Items matching in an earlier language rank above items matching in a later one.
// Only items from the sheets that sheet lets through are considered; filtering never
// changes the order of the items that remain.
// Uses Bleve full-text search for better performance and relevance.
//
// A query that expands past maxClauseCount is reported as a *QueryTooBroadError, and
// one still running when ctx ends fails with ctx's error.
func (s *Store) Search(ctx context.Context, q string, langs []string, sheet SheetFilter, offset, limit int, fields []string) (*SearchResult, error) {
	if s.index == nil {
		return nil, fmt.Errorf("index is not loaded")
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("no search language given")
	}

	return s.search(ctx, parseSearchQuery(q, langs, sheet), q, langs, offset, limit, fields, false)
}

// AdvancedSearch is Search with q read as a query string; see parseAdvancedQuery for
// the syntax. A query that does not parse is reported as an *InvalidQueryError.
func (s *Store) AdvancedSearch(ctx context.Context, q string, langs []string, sheet SheetFilter, offset, limit int, fields []string) (*SearchResult, error) {
	if s.index == nil {
		return nil, fmt.Errorf("index is not loaded")
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("no search language given")
	}

	query, err := parseAdvancedQuery(q, langs, sheet)
	if err != nil {
		return nil, &InvalidQueryError{err: err}
	}

	return s.search(ctx, query, q, langs, offset, limit, fields, true)
}

// InvalidQueryError is a query the caller got wrong, as opposed to a search that failed.
type InvalidQueryError struct {
	err error
}

func (e *InvalidQueryError) Error() string { return e.err.Error() }
func (e *InvalidQueryError) Unwrap() error { return e.err }

// search runs query, matching q in langs.
//
// With highlightAll, bleve highlights every field asked for, instead of only langs. An
// advanced query can name fields outside langs, and its text is syntax rather than
// words, so it cannot be analyzed by hand the way addDisplayFragments does.
func (s *Store) search(ctx context.Context, query query.Query, q string, langs []string, offset, limit int, fields []string, highlightAll bool) (*SearchResult, error) {
	searchFields := make([]string, 0, len(fields)+len(metaFields))
	searchFields = append(searchFields, metaFields...)
	searchFields = append(searchFields, fields...)

	request := bleve.NewSearchRequestOptions(query, limit, offset, false)
	request.Fields = searchFields
	request.Highlight = bleve.NewHighlightWithStyle("html")
	// Without an explicit list Bleve highlights every field the query touched, which
	// includes the sheet filter term. Restrict it to the languages.
	highlightFields := langs
	if highlightAll {
		highlightFields = slices.Concat(langs, fields)
	}
	for _, lang := range highlightFields {
		if !slices.Contains(request.Highlight.Fields, lang) {
			request.Highlight.AddField(lang)
		}
	}

	searchResults, err := s.index.SearchInContext(ctx, request)
	if err != nil {
		return nil, searchFailed(err)
	}

	// Bleve only fragmented the languages it searched. Give the rest one too, so every
	// column of a row is trimmed alike instead of one showing a snippet next to thousands
	// of characters.
	if !highlightAll {
		s.addDisplayFragments(searchResults.Hits, q, langs, fields)
	}

	items := make([]*Item, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		items = append(items, formatItemFromHit(hit))
	}

	return &SearchResult{
		Items:   items,
		Total:   searchResults.Total,
		Elapsed: searchResults.Took,
	}, nil
}

// GetBySheet returns items for a given sheet with pagination.
// Returns early when offset+limit items are found to optimize performance.
func (s *Store) GetBySheet(ctx context.Context, sheet string, offset, limit int, fields []string) (*SearchResult, error) {
	if s.index == nil {
		return nil, fmt.Errorf("index is not loaded")
	}

	from := float64(offset)
	to := float64(offset + limit)

	indexQuery := bleve.NewNumericRangeQuery(&from, &to)
	indexQuery.SetField("index")

	sheetQuery := bleve.NewTermQuery(sheet)
	sheetQuery.SetField("sheet")

	query := bleve.NewConjunctionQuery(
		indexQuery,
		sheetQuery,
	)

	searchFields := make([]string, 0, len(fields)+len(metaFields))
	searchFields = append(searchFields, metaFields...)
	searchFields = append(searchFields, fields...)

	request := bleve.NewSearchRequestOptions(query, limit, 0, false)
	request.Fields = searchFields
	request.SortBy([]string{"index"})

	searchResults, err := s.index.SearchInContext(ctx, request)
	if err != nil {
		return nil, searchFailed(err)
	}

	items := make([]*Item, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		items = append(items, formatItemFromHit(hit))
	}

	return &SearchResult{
		Items:   items,
		Total:   searchResults.Total,
		Elapsed: searchResults.Took,
	}, nil
}

// Close closes the Bleve index and releases resources.
func (s *Store) Close() error {
	if s.index != nil {
		return s.index.Close()
	}
	return nil
}
