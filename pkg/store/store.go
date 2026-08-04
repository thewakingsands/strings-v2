package store

import (
	"fmt"
	"log"
	"time"

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

// LoadStore loads all JSON files from dataDir into memory.
func LoadStore(dataDir string, indexDir string) (*Store, error) {
	idx, err := bleve.Open(indexDir)
	if err == bleve.ErrorIndexPathDoesNotExist {
		idx, err = buildItemIndex(dataDir, indexDir)
		if err != nil {
			return nil, fmt.Errorf("build item index: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("create bleve index: %w", err)
	}

	s := &Store{
		index: idx,
	}

	return s, nil
}

var metaFields = []string{"sheet", "id", "index"}

// parseSearchQuery matches q against every language in langs, ranking earlier
// languages above later ones.
func parseSearchQuery(q string, langs []string, sheet string) query.Query {
	disjuncts := make([]query.Query, 0, len(langs))
	for i, lang := range langs {
		matchQuery := bleve.NewMatchQuery(q)
		matchQuery.SetField(lang)
		// Position in langs is the caller's preference order, so langs=chs,en,ja means
		// chs^3 en^2 ja^1. A single language is left unboosted so that its scores stay
		// bit for bit identical to a search that never knew about multiple languages.
		if len(langs) > 1 {
			matchQuery.SetBoost(float64(len(langs) - i))
		}
		disjuncts = append(disjuncts, matchQuery)
	}

	var textQuery query.Query
	if len(disjuncts) == 1 {
		textQuery = disjuncts[0]
	} else {
		textQuery = bleve.NewDisjunctionQuery(disjuncts...)
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
