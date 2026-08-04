package store

import (
	"fmt"
	"log"
	"time"
	"unicode"

	"github.com/blevesearch/bleve/v2"
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
// in dataDir first if it is not there yet.
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

// parseSearchQuery matches q against every language in langs, ranking earlier
// languages above later ones when they are otherwise equally good.
func parseSearchQuery(q string, langs []string, sheet string) query.Query {
	nonLatin := isNonLatinQuery(q)

	disjuncts := make([]query.Query, 0, len(langs))
	for i, lang := range langs {
		matchQuery := bleve.NewMatchQuery(q)
		matchQuery.SetField(lang)

		boost := 1.0
		if i > 0 {
			boost = 1 - float64(i)*languageBoostStep
		}
		if nonLatin && !usesCJKAnalyzer(lang) {
			boost *= nonNativeScriptBoost
		}
		// Left alone at exactly 1, which keeps a single language Latin search scoring
		// bit for bit as it did before this parameter accepted a list.
		if boost != 1 {
			matchQuery.SetBoost(boost)
		}

		disjuncts = append(disjuncts, matchQuery)
	}

	var textQuery query.Query
	if len(disjuncts) == 1 {
		textQuery = disjuncts[0]
	} else {
		textQuery = newDisMaxQuery(disjuncts)
	}

	if sheet == "" {
		return textQuery
	}

	sheetQuery := bleve.NewTermQuery(sheet)
	sheetQuery.SetField("sheet")

	return bleve.NewConjunctionQuery(
		textQuery,
		sheetQuery,
	)
}

// isNonLatinQuery reports whether the query has letters and none of them are Latin.
// Such a query carries no word an en, de or fr analyzer can recognise, so those fields
// can only match it a character at a time. German ü and French é are Latin letters, and
// digits or punctuation on their own do not make a query non-Latin.
func isNonLatinQuery(q string) bool {
	found := false
	for _, r := range q {
		if !unicode.IsLetter(r) {
			continue
		}
		if unicode.Is(unicode.Latin, r) {
			return false
		}
		found = true
	}
	return found
}

// Search finds items whose value in any of the given languages matches the query.
// Items matching in an earlier language rank above items matching in a later one.
// If sheetFilter is non-empty, only items from that sheet are considered.
// Uses Bleve full-text search for better performance and relevance.
func (s *Store) Search(q string, langs []string, sheet string, offset, limit int, fields []string) (*SearchResult, error) {
	if s.index == nil {
		return nil, fmt.Errorf("index is not loaded")
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("no search language given")
	}

	query := parseSearchQuery(q, langs, sheet)

	searchFields := make([]string, 0, len(fields)+len(metaFields))
	searchFields = append(searchFields, metaFields...)
	searchFields = append(searchFields, fields...)

	request := bleve.NewSearchRequestOptions(query, limit, offset, false)
	request.Fields = searchFields
	request.Highlight = bleve.NewHighlightWithStyle("html")
	// Without an explicit list Bleve highlights every field the query touched, which
	// includes the sheet filter term. Restrict it to the languages actually searched.
	for _, lang := range langs {
		request.Highlight.AddField(lang)
	}

	searchResults, err := s.index.Search(request)
	if err != nil {
		log.Printf("search error: %v", err)
		return nil, fmt.Errorf("search error: %w", err)
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
func (s *Store) GetBySheet(sheet string, offset, limit int, fields []string) (*SearchResult, error) {
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

	searchResults, err := s.index.Search(request)
	if err != nil {
		log.Printf("search error: %v", err)
		return nil, fmt.Errorf("search error: %w", err)
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
