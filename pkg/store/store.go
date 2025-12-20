package store

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

// Store keeps all items in memory and provides simple lookup helpers.
type Store struct {
	items      []*Item
	bySheet    map[string][]*Item // sheet -> items
	sheetIndex map[string]int     // sheet -> next index counter
	index      bleve.Index        // Bleve search index
}

// LoadStore loads all JSON files from dataDir into memory.
func LoadStore(dataDir string) (*Store, error) {
	// Create Bleve index mapping
	indexMapping := bleve.NewIndexMapping()

	// Define document mapping for Item
	itemMapping := bleve.NewDocumentMapping()

	// Sheet field mapping - use keyword analyzer for exact matching
	sheetFieldMapping := bleve.NewTextFieldMapping()
	sheetFieldMapping.Analyzer = "keyword"
	sheetFieldMapping.Store = false
	sheetFieldMapping.Index = true
	itemMapping.AddFieldMappingsAt("sheet", sheetFieldMapping)

	// Set default mapping to use keyword analyzer for all fields (language values)
	// This enables wildcard substring matching for CJK and other languages
	indexMapping.DefaultMapping = itemMapping
	indexMapping.DefaultAnalyzer = "keyword"

	// Create in-memory Bleve index
	idx, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		return nil, fmt.Errorf("create bleve index: %w", err)
	}

	s := &Store{
		bySheet:    make(map[string][]*Item),
		sheetIndex: make(map[string]int),
		index:      idx,
	}

	files, err := scanDataFiles(dataDir)
	if err != nil {
		return nil, fmt.Errorf("scan data files: %w", err)
	}

	log.Printf("loading %d files", len(files))
	for _, path := range files {
		items, err := loadFile(path)
		if err != nil {
			return nil, fmt.Errorf("load file %s: %w", path, err)
		}
		for _, it := range items {
			s.addItem(it)
		}
	}

	log.Printf("loaded %d items from %d files", len(s.items), len(files))
	return s, nil
}

func (s *Store) addItem(it *Item) {
	// Assign index based on sheet and order in file
	// Index is unique per sheet and follows the order items appear in files
	if _, ok := s.sheetIndex[it.Sheet]; !ok {
		s.sheetIndex[it.Sheet] = 0
	}
	it.Index = s.sheetIndex[it.Sheet]
	s.sheetIndex[it.Sheet]++

	s.items = append(s.items, it)

	if _, ok := s.bySheet[it.Sheet]; !ok {
		s.bySheet[it.Sheet] = make([]*Item, 0)
	}
	s.bySheet[it.Sheet] = append(s.bySheet[it.Sheet], it)

	// Index the item in Bleve
	// Create a document ID using sheet, rowId, and index with a special delimiter
	// Use "|@|" as delimiter which is unlikely to appear in sheet names or rowIds
	docID := fmt.Sprintf("%s|@|%s|@|%d", it.Sheet, it.RowID, it.Index)

	// Create a searchable document with all language values
	doc := map[string]interface{}{
		"sheet": it.Sheet,
		"rowId": it.RowID,
		"index": it.Index,
	}

	// Add all language values to the document
	// Lowercase values for case-insensitive wildcard matching with keyword analyzer
	if it.Values != nil {
		for lang, value := range it.Values {
			doc[lang] = strings.ToLower(value)
		}
	}

	// Index the document
	if err := s.index.Index(docID, doc); err != nil {
		log.Printf("warning: failed to index item %s: %v", docID, err)
	}
}

// Search finds items whose value in the given language contains the query substring.
// If sheetFilter is non-empty, only items from that sheet are considered.
// Uses Bleve full-text search for better performance and relevance.
// Supports multi-word queries: "悲伤 表情" will match text containing both terms.
func (s *Store) Search(lang, queryStr, sheetFilter string, offset, limit int) []*Item {
	lang = strings.ToLower(strings.TrimSpace(lang))
	q := strings.TrimSpace(queryStr)

	if q == "" {
		return []*Item{}
	}

	// Build Bleve query - use wildcard query for substring matching
	var bleveQuery query.Query

	// Split query by whitespace to support multi-word search
	// e.g., "悲伤 表情" should match text containing both "悲伤" and "表情"
	terms := strings.Fields(q)

	if len(terms) == 0 {
		return []*Item{}
	}

	// Create wildcard queries for each term
	var termQueries []query.Query
	for _, term := range terms {
		// Wrap each term with wildcards for substring search
		wildcardStr := "*" + strings.ToLower(term) + "*"
		wildcardQuery := query.NewWildcardQuery(wildcardStr)
		wildcardQuery.SetField(lang)
		termQueries = append(termQueries, wildcardQuery)
	}

	// If multiple terms, combine them with AND logic (conjunction)
	var matchQuery query.Query
	if len(termQueries) == 1 {
		matchQuery = termQueries[0]
	} else {
		matchQuery = query.NewConjunctionQuery(termQueries)
	}

	if sheetFilter != "" {
		// If sheet filter is provided, combine with a conjunction query
		sheetQuery := query.NewMatchQuery(sheetFilter)
		sheetQuery.SetField("sheet")

		conjQuery := query.NewConjunctionQuery([]query.Query{matchQuery, sheetQuery})
		bleveQuery = conjQuery
	} else {
		bleveQuery = matchQuery
	}

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.From = offset
	searchRequest.Size = limit
	searchRequest.Fields = []string{"sheet", "rowId", "index"}

	// Execute search
	searchResults, err := s.index.Search(searchRequest)
	if err != nil {
		log.Printf("search error: %v", err)
		return []*Item{}
	}

	// Build result set by looking up items from the store
	results := make([]*Item, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		// Parse the document ID to find the item
		// Format: sheet|@|rowId|@|index (using |@| as delimiter)
		parts := strings.Split(hit.ID, "|@|")
		if len(parts) != 3 {
			continue
		}

		sheet := parts[0]
		rowId := parts[1]
		indexStr := parts[2]

		idx, err := strconv.Atoi(indexStr)
		if err != nil {
			continue
		}

		// Find the item in our store
		if sheetItems, ok := s.bySheet[sheet]; ok {
			for _, item := range sheetItems {
				if item.RowID == rowId && item.Index == idx {
					results = append(results, item)
					break
				}
			}
		}
	}

	return results
}

// GetBySheet returns items for a given sheet with pagination.
// Returns early when offset+limit items are found to optimize performance.
func (s *Store) GetBySheet(sheet string, offset, limit int) []*Item {
	sheetMap, ok := s.bySheet[sheet]
	if !ok {
		return nil
	}

	// We need to find (offset + limit) items total, then return the last 'limit' items
	needed := offset + limit
	results := make([]*Item, 0, needed)

	for _, item := range sheetMap {
		results = append(results, item)
		// Return early if we have enough results
		if len(results) >= needed {
			goto done
		}
	}

done:
	// Apply offset and limit
	if offset >= len(results) {
		return []*Item{}
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end]
}

// Close closes the Bleve index and releases resources.
func (s *Store) Close() error {
	if s.index != nil {
		return s.index.Close()
	}
	return nil
}
