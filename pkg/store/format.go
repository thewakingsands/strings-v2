package store

import (
	"github.com/blevesearch/bleve/v2/search"
)

func formatItemDocument(item *Item) *ItemDocument {
	return &ItemDocument{
		Sheet: item.Sheet,
		Id:    item.RowID,
		Index: item.Index,
		Chs:   item.Values["chs"],
		Tc:    item.Values["tc"],
		Ja:    item.Values["ja"],
		Ko:    item.Values["ko"],
		En:    item.Values["en"],
		De:    item.Values["de"],
		Fr:    item.Values["fr"],
	}
}

func formatItemFromHit(hit *search.DocumentMatch) *Item {
	values := make(map[string]string)
	var highlights map[string]string

	for key, field := range hit.Fields {
		if key == "sheet" || key == "id" || key == "index" {
			continue
		}

		text, ok := field.(string)
		if !ok {
			continue
		}
		values[key] = text

		// Every requested language gets a fragment, whether or not it matched. One that
		// matched carries <mark>; one that did not is the head of the value, which is
		// what a column with nothing to highlight should show. Both are trimmed by the
		// same fragmenter, which is what keeps the columns comparable in length.
		fragments, ok := hit.Fragments[key]
		if !ok || len(fragments) == 0 {
			continue
		}
		if highlights == nil {
			highlights = make(map[string]string)
		}
		highlights[key] = fragments[0]
	}

	return &Item{
		Sheet:      hit.Fields["sheet"].(string),
		RowID:      hit.Fields["id"].(string),
		Values:     values,
		Highlights: highlights,
		Index:      uint32(hit.Fields["index"].(float64)),
	}
}
