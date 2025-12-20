package store

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
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
func LoadStore(dataDir string) (*Store, error) {
	indexDir := filepath.Join(dataDir, "index")
	idx, err := bleve.Open(indexDir)
	if err == bleve.ErrorIndexPathDoesNotExist {
		log.Printf("Creating new index...")

		mapping := buildItemIndex()
		idx, err = bleve.New(indexDir, mapping)
		if err != nil {
			return nil, fmt.Errorf("create bleve index: %w", err)
		}

		files, err := scanDataFiles(dataDir)
		if err != nil {
			return nil, fmt.Errorf("scan data files: %w", err)
		}

		log.Printf("loading %d files", len(files))

		count := 0
		sheetIndex := make(map[string]uint32)
		for _, path := range files {
			items, err := loadFile(path)
			if err != nil {
				return nil, fmt.Errorf("load file %s: %w", path, err)
			}

			for _, it := range items {
				// Assign index based on sheet and order in file
				// Index is unique per sheet and follows the order items appear in files
				if _, ok := sheetIndex[it.Sheet]; !ok {
					sheetIndex[it.Sheet] = 0
				}
				it.Index = sheetIndex[it.Sheet]
				sheetIndex[it.Sheet]++
			}

			// Index the item in Bleve
			err = indexItems(idx, items)
			if err != nil {
				return nil, fmt.Errorf("index items: %w", err)
			}

			count += len(items)
		}

		log.Printf("loaded %d items from %d files", count, len(files))
	} else if err != nil {
		return nil, fmt.Errorf("create bleve index: %w", err)
	}

	s := &Store{
		index: idx,
	}

	return s, nil
}

// Search finds items whose value in the given language contains the query substring.
// If sheetFilter is non-empty, only items from that sheet are considered.
// Uses Bleve full-text search for better performance and relevance.
func (s *Store) Search(lang string, queryStr string, offset, limit int) (*SearchResult, error) {
	q := strings.TrimSpace(queryStr)

	if q == "" {
		return nil, fmt.Errorf("query is empty")
	}

	query := bleve.NewMatchQuery(q)
	query.SetField(lang)

	request := bleve.NewSearchRequestOptions(query, limit, offset, false)
	request.Fields = []string{"*"}
	request.Highlight = bleve.NewHighlightWithStyle("html")

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
func (s *Store) GetBySheet(sheet string, offset, limit int) (*SearchResult, error) {
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

	request := bleve.NewSearchRequestOptions(query, limit, 0, false)
	request.Fields = []string{"*"}
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
