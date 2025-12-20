package store

import (
	"fmt"
	"log"
	"strings"
)

// Store keeps all items in memory and provides simple lookup helpers.
type Store struct {
	items      []*Item
	bySheet    map[string][]*Item // sheet -> items
	sheetIndex map[string]int     // sheet -> next index counter
}

// LoadStore loads all JSON files from dataDir into memory.
func LoadStore(dataDir string) (*Store, error) {
	s := &Store{
		bySheet:    make(map[string][]*Item),
		sheetIndex: make(map[string]int),
	}

	files, err := scanDataFiles(dataDir)
	if err != nil {
		return nil, fmt.Errorf("scan data files: %w", err)
	}
	if len(files) == 0 {
		log.Printf("no data files found in %s", dataDir)
	}

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
}

// Search finds items whose value in the given language contains the query substring.
// If sheetFilter is non-empty, only items from that sheet are considered.
// Returns early when offset+limit items are found to optimize performance.
func (s *Store) Search(lang, query, sheetFilter string, offset, limit int) []*Item {
	lang = strings.ToLower(strings.TrimSpace(lang))
	q := strings.ToLower(query)

	// We need to find (offset + limit) items total, then return the last 'limit' items
	needed := offset + limit
	results := make([]*Item, 0, needed)

	for _, it := range s.items {
		if sheetFilter != "" && it.Sheet != sheetFilter {
			continue
		}
		if it.Values == nil {
			continue
		}

		value, ok := it.Values[lang]
		if !ok {
			continue
		}

		if strings.Contains(strings.ToLower(value), q) {
			results = append(results, it)
			// Return early if we have enough results
			if len(results) >= needed {
				break
			}
		}
	}

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
